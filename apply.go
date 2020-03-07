package klient

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/resource"
)

// Apply creates a resource with the given content
func (c *Client) Apply(content []byte) error {
	r := c.ResultForContent(content, nil)
	return c.ApplyResource(r)
}

// ApplyFiles create the resource(s) from the given filenames (file, directory or STDIN) or HTTP URLs
func (c *Client) ApplyFiles(filenames ...string) error {
	r := c.ResultForFilenameParam(filenames, nil)
	return c.ApplyResource(r)
}

// ApplyResource applies the given resource. Create the resources with `ResultForFilenameParam` or `ResultForContent`
func (c *Client) ApplyResource(r *resource.Result) error {
	return r.Visit(apply)
}

func apply(info *resource.Info, err error) error {
	if err != nil {
		return failedTo("apply", info, err)
	}

	// If it does not exists, just create it
	originalObj, err := resource.NewHelper(info.Client, info.Mapping).Get(info.Namespace, info.Name, info.Export)
	if err != nil {
		if !errors.IsNotFound(err) {
			return failedTo("retrieve current configuration", info, err)
		}
		return create(info, nil)
	}

	// If exists, patch it
	return patch(info, originalObj)
}
