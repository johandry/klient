package kube

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions/resource"
)

// Apply creates a resource with the given content
func (c *Client) Apply(content []byte) error {
	if err := c.ApplyNamespace(c.namespace); err != nil {
		return err
	}

	// infos, err := c.ResultForContent(content, true).Infos()
	// if err != nil {
	// 	return err
	// }

	r := c.ResultForContent(content, true)
	return r.Visit(apply)
}

// Apply creates a resource with the given content
func (c *Client) ApplyFile(filename string) error {
	if err := c.ApplyNamespace(c.namespace); err != nil {
		return err
	}

	r := c.ResultForFilenameParam(filename)
	return r.Visit(apply)
}

func apply(info *resource.Info, err error) error {
	if err != nil {
		return err
	}

	// modified, err := kubectl.GetModifiedConfiguration(info.Object, true, unstructured.UnstructuredJSONScheme)
	// if err != nil {
	// 	return fmt.Errorf("retrieving modified configuration from %s. %s", info.String(), err)
	// }

	// If does not exists, just create it
	if err := info.Get(); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("retrieving current configuration of %s. %s", info.String(), err)
		}
		return create(info, err)
	}

	// If exists, patch it
	return patch(info, err)
}

// Create creates a resource with the given content
func (c *Client) Create(content []byte) error {
	if err := c.ApplyNamespace(c.namespace); err != nil {
		return err
	}

	r := c.ResultForContent(content, true)
	return r.Visit(create)
}

// CreateFile creates a resource with the given content
func (c *Client) CreateFile(filename string) error {
	if err := c.ApplyNamespace(c.namespace); err != nil {
		return err
	}

	r := c.ResultForFilenameParam(filename)
	return r.Visit(create)
}

func create(info *resource.Info, err error) error {
	// if err := kubectl.CreateApplyAnnotation(info.Object, unstructured.UnstructuredJSONScheme); err != nil {
	// 	return fmt.Errorf("creating %s. %s", info.String(), err)
	// }

	options := metav1.CreateOptions{}
	obj, err := resource.NewHelper(info.Client, info.Mapping).Create(info.Namespace, true, info.Object, &options)
	if err != nil {
		return fmt.Errorf("creating %s. %s", info.String(), err)
	}
	return info.Refresh(obj, true)
}

func patch(info *resource.Info) error {
	return fmt.Errorf("patch is not implemented yet")
}
