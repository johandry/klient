package kube

import (
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateNamespace creates a namespace with the given name
func (c *Client) CreateNamespace(namespace string) error {
	if c.clientset == nil {
		clientset, err := c.Config.KubernetesClientSet()
		if err != nil {
			return err
		}
		c.clientset = clientset
	}

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"name": namespace,
			},
		},
	}
	_, err := c.clientset.CoreV1().Namespaces().Create(ns)
	return err
}

// Namespace returns the namespace with the given name. If not found returns a
// IsNotFound error
func (c *Client) Namespace(namespace string) (*v1.Namespace, error) {
	if c.clientset == nil {
		clientset, err := c.Config.KubernetesClientSet()
		if err != nil {
			return nil, err
		}
		c.clientset = clientset
	}

	return c.clientset.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
}

// ApplyNamespace creates the given namespace if does not exists
func (c *Client) ApplyNamespace(namespace string) error {
	_, err := c.Namespace(namespace)
	if err != nil && errors.IsNotFound(err) {
		if err := c.CreateNamespace(namespace); errors.IsAlreadyExists(err) {
			return nil
		}
	}
	return err
}
