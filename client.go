package kube

import (
	"bytes"
	"io"

	"k8s.io/cli-runtime/pkg/genericclioptions/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/kubectl/validation"
)

// Client is a kubernetes client, like `kubectl`
type Client struct {
	Config *Config
	// validator        validation.Schema
	namespace        string
	enforceNamespace bool
	clientset        *kubernetes.Clientset
}

// NewClientE creates a kubernetes client, returns an error if fail
func NewClientE(context, kubeconfig string) (*Client, error) {
	config := NewConfig(context, kubeconfig)

	// validator, _ := config.Validator(true)

	namespace, enforceNamespace, err := config.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, err
	}
	clientset, err := config.KubernetesClientSet()
	if err != nil {
		return nil, err
	}

	return &Client{
		Config:           config,
		validator:        validation.NullSchema{},
		namespace:        namespace,
		enforceNamespace: enforceNamespace,
		clientset:        clientset,
	}, nil
}

// NewClient creates a kubernetes client
func NewClient(context, kubeconfig string) *Client {
	client, _ := NewClientE(context, kubeconfig)
	return client
}

// UnstructuredBuilder creates an unstructure builder for the given namespace
func (c *Client) UnstructuredBuilder() *resource.Builder {
	return c.Config.
		NewBuilder().
		Unstructured().
		Schema(c.validator).
		ContinueOnError().
		NamespaceParam(c.namespace).DefaultNamespace()
}

// Builder creates a builder for the given namespace
func (c *Client) Builder() *resource.Builder {
	return c.Config.
		NewBuilder().
		Schema(c.validator).
		ContinueOnError().
		NamespaceParam(c.namespace).DefaultNamespace()
}

// ResultForFilenameParam returns the builder results for the given list of files or URLs
func (c *Client) ResultForFilenameParam(filenames []string, unstructured bool) *resource.Result {
	filenameOptions := &resource.FilenameOptions{
		Recursive: false,
		Filenames: filenames,
	}

	var b *resource.Builder
	if unstructured {
		b = c.UnstructuredBuilder()
	} else {
		b = c.Builder()
	}

	return b.
		FilenameParam(c.enforceNamespace, filenameOptions).
		Flatten().
		Do()
}

// ResultForReader returns the builder results for the given reader
func (c *Client) ResultForReader(r io.Reader, unstructured bool) *resource.Result {
	var b *resource.Builder
	if unstructured {
		b = c.UnstructuredBuilder()
	} else {
		b = c.Builder()
	}

	return b.
		Stream(r, "").
		Flatten().
		Do()
}

// ResultForContent returns the builder results for the given content
func (c *Client) ResultForContent(content []byte, unstructured bool) *resource.Result {
	b := bytes.NewBuffer(content)
	return c.ResultForReader(b, unstructured)
}
