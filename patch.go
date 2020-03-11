package klient

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/resource"
)

func patch(info *resource.Info, current runtime.Object) error {
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

	obj, err := resource.NewHelper(info.Client, info.Mapping).Patch(info.Namespace, info.Name, patchType, patch, nil)
	if err != nil {
		return failedTo("patch", info, err)
	}

	info.Refresh(obj, true)
	return nil
}

func createPatch(info *resource.Info, current runtime.Object) ([]byte, types.PatchType, error) {
	oldData, err := json.Marshal(current)
	if err != nil {
		return nil, types.StrategicMergePatchType, fmt.Errorf("serializing current configuration: %s", err)
	}
	newData, err := json.Marshal(info.Object)
	if err != nil {
		return nil, types.StrategicMergePatchType, fmt.Errorf("serializing info configuration: %s", err)
	}

	patch, err := jsonpatch.CreateMergePatch(oldData, newData)
	return patch, types.MergePatchType, err
}
