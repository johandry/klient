package kubectl

import (
	"bytes"
	"fmt"
	"io"

	v1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/validation"
)

// Client is a kubernetes client, like `kubectl`
type Client struct {
	Clientset        *kubernetes.Clientset
	factory          *factory
	validator        validation.Schema
	namespace        string
	enforceNamespace bool
}

// Result is an alias for the Kubernetes CLI runtime resource.Result
type Result = resource.Result

// NewClientE creates a kubernetes client, returns an error if fail
func NewClientE(context, kubeconfig string) (*Client, error) {
	factory := newFactory(context, kubeconfig)

	// If `true` it will always validate the given objects/resources
	validator, _ := factory.Validator(true)

	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		namespace = v1.NamespaceDefault
		enforceNamespace = true
	}
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return nil, err
	}
	if clientset == nil {
		return nil, fmt.Errorf("cannot create a clientset from given context and kubeconfig")
	}

	return &Client{
		factory:          factory,
		Clientset:        clientset,
		validator:        validator,
		namespace:        namespace,
		enforceNamespace: enforceNamespace,
	}, nil
}

// NewClient creates a kubernetes client
func NewClient(context, kubeconfig string) *Client {
	client, _ := NewClientE(context, kubeconfig)
	return client
}

// Builder creates a resource builder
func (c *Client) builder(unstructured bool) *resource.Builder {
	b := c.factory.NewBuilder()

	if unstructured {
		b = b.Unstructured()
	}

	return b.
		Schema(c.validator).
		ContinueOnError().
		NamespaceParam(c.namespace).DefaultNamespace()
}

// ResultForFilenameParam returns the builder results for the given list of files or URLs
func (c *Client) ResultForFilenameParam(filenames []string, unstructured bool) *Result {
	filenameOptions := &resource.FilenameOptions{
		Recursive: false,
		Filenames: filenames,
	}

	return c.builder(unstructured).
		FilenameParam(c.enforceNamespace, filenameOptions).
		Flatten().
		Do()
}

// ResultForReader returns the builder results for the given reader
func (c *Client) ResultForReader(r io.Reader, unstructured bool) *Result {
	return c.builder(unstructured).
		Stream(r, "").
		Flatten().
		Do()
}

// ResultForContent returns the builder results for the given content
func (c *Client) ResultForContent(content []byte, unstructured bool) *Result {
	b := bytes.NewBuffer(content)
	return c.ResultForReader(b, unstructured)
}

func failedTo(action string, info *resource.Info, err error) error {
	var resKind string
	if info.Mapping != nil {
		resKind = info.Mapping.GroupVersionKind.Kind + " "
	}

	return fmt.Errorf("cannot %s object Kind: %q,	Name: %q, Namespace: %q. %s", action, resKind, info.Name, info.Namespace, err)
}
