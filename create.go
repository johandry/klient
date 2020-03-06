package klient

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/resource"
)

// Create creates a resource with the given content
func (c *Client) Create(content []byte) error {
	r := c.ResultForContent(content, true)
	return c.CreateResource(r)
}

// CreateFile creates a resource with the given content
func (c *Client) CreateFile(filenames ...string) error {
	r := c.ResultForFilenameParam(filenames, true)
	return c.CreateResource(r)
}

// CreateResource creates the given resource. Create the resources with `ResultForFilenameParam` or `ResultForContent`
func (c *Client) CreateResource(r *resource.Result) error {
	return r.Visit(create)
}

func create(info *resource.Info, err error) error {
	if err != nil {
		return failedTo("create", info, err)
	}

	options := metav1.CreateOptions{}
	obj, err := resource.NewHelper(info.Client, info.Mapping).Create(info.Namespace, true, info.Object, &options)
	if err != nil {
		return failedTo("create", info, err)
	}
	info.Refresh(obj, true)
	return nil
}

func reCreate(info *resource.Info) error {
	// TODO: this method is to delete and create the resource. Requires the
	// implementation of a delete method
	return nil
}
