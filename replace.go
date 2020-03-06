package klient

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/cli-runtime/pkg/resource"
)

// Replace creates a resource with the given content
func (c *Client) Replace(content []byte) error {
	r := c.ResultForContent(content, true)
	return c.ReplaceResource(r)
}

// ReplaceFiles create the resource(s) from the given filenames (file, directory or STDIN) or HTTP URLs
func (c *Client) ReplaceFiles(filenames ...string) error {
	r := c.ResultForFilenameParam(filenames, true)
	return c.ReplaceResource(r)
}

// ReplaceResource applies the given resource. Create the resources with `ResultForFilenameParam` or `ResultForContent`
func (c *Client) ReplaceResource(r *resource.Result) error {
	return r.Visit(replace)
}

func replace(info *resource.Info, err error) error {
	if err != nil {
		return failedTo("replace", info, err)
	}

	originalObj, err := resource.NewHelper(info.Client, info.Mapping).Get(info.Namespace, info.Name, info.Export)
	if err != nil {
		return fmt.Errorf("retrieving current configuration of %s. %s", info.String(), err)
	}

	return patch(info, originalObj)
}

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
