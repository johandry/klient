package klient

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/cli-runtime/pkg/resource"
)

// Use this function to switch between the different patches
func patch(info *resource.Info, current runtime.Object) error {
	return patchV2(info, current)
}

// patchV2 is the same implementation of patch but as Helm do it. It's here
// in case the existing patch (from kubectl) fails
func patchV2(info *resource.Info, current runtime.Object) error {
	patch, patchType, err := createPatch(info, current)
	if err != nil {
		return failedTo("create patch", info, err)
	}

	if patch == nil || string(patch) == "{}" {
		// there is nothing to update
		if err := info.Get(); err != nil {
			return failedTo("refresh", info, err)
		}
		return nil
	}

	// TODO: Find out how to get the forceConflicts from the client
	forceConflicts := false

	if forceConflicts {
		return replace(info, nil)
	}

	// Server Side Apply:
	obj, err := resource.NewHelper(info.Client, info.Mapping).Patch(info.Namespace, info.Name, patchType, patch, &metav1.PatchOptions{})
	if err != nil {
		return failedTo("serverside patch", info, err)
	}
	info.Refresh(obj, true)

	return nil
}

func createPatch(info *resource.Info, current runtime.Object) ([]byte, types.PatchType, error) {
	currentJSON, err := json.Marshal(current)
	if err != nil {
		return nil, types.StrategicMergePatchType, fmt.Errorf("serializing current configuration: %s", err)
	}
	infoJSON, err := json.Marshal(info.Object)
	if err != nil {
		return nil, types.StrategicMergePatchType, fmt.Errorf("serializing info configuration: %s", err)
	}

	// Fetch the current object for the three way merge
	currentObj, err := resource.NewHelper(info.Client, info.Mapping).Get(info.Namespace, info.Name, info.Export)
	if err != nil && !errors.IsNotFound(err) {
		return nil, types.StrategicMergePatchType, fmt.Errorf("get object: %s", err)
	}

	// Even if currentObj is nil (because it was not found), it will marshal just fine
	currentData, err := json.Marshal(currentObj)
	if err != nil {
		return nil, types.StrategicMergePatchType, fmt.Errorf("serializing live configuration: %s", err)
	}

	var gv = runtime.GroupVersioner(schema.GroupVersions(k8sNativeScheme.PrioritizedVersionsAllGroups()))
	if info.Mapping != nil {
		gv = info.Mapping.GroupVersionKind.GroupVersion()
	}
	versionedObject, _ := runtime.ObjectConvertor(k8sNativeScheme).ConvertToVersion(info.Object, gv)

	// Unstructured objects, such as CRDs, may not have an not registered error
	// returned from ConvertToVersion. Anything that's unstructured should
	// use the jsonpatch.CreateMergePatch. Strategic Merge Patch is not supported
	// on objects like CRDs.
	_, isUnstructured := versionedObject.(runtime.Unstructured)

	// On newer K8s versions, CRDs aren't unstructured but has this dedicated type
	_, isCRD := versionedObject.(*apiextv1beta1.CustomResourceDefinition)

	if isUnstructured || isCRD {
		// fall back to generic JSON merge patch
		patch, err := jsonpatch.CreateMergePatch(currentJSON, infoJSON)
		return patch, types.MergePatchType, err
	}

	patchMeta, err := strategicpatch.NewPatchMetaFromStruct(versionedObject)
	if err != nil {
		return nil, types.StrategicMergePatchType, fmt.Errorf("cannot create patch metadata from object. %s", err)
	}

	patch, err := strategicpatch.CreateThreeWayMergePatch(currentJSON, infoJSON, currentData, patchMeta, true)
	return patch, types.StrategicMergePatchType, err
}
