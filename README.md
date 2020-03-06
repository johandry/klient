# A Simple Kubernetes Client in Go

The goal of this package is to provide an easy interface to communicate with a Kubernetes cluster similar to using the `kubectl` commands `apply`, `create`, `delete` and `replace`.

This package is not a replacement of `k8s.io/client-go`, its main purpose is to apply, create, delete or replace resources using the YAML or JSON representation of Kubernetes objects. The package `k8s.io/client-go` requires to know what objects are going to be managed and uses the Kubernetes API to manage them. If you have a file, URL, stream or string with a Kubernetes object to apply (not knowing exactly what's inside) then `klient` will help you.

Many thanks to the contributors of [Kubectl](https://github.com/kubernetes/kubectl) and [Helm](https://github.com/helm/helm). This package was made inspired by these two amazing projects, some of the code in this package was copied from them, there is a reference to the source whenever a code was copied. Although it's not so easy or intuitive to use the kubectl code from your Go projects, it's not complex to import and use the Helm Kubernetes client package, so you could either use this package or the Helm Kubernetes client package to communicate with a Kubernetes cluster.

This package is part of the blog article [Building a Kubernetes Client in Go](http://blog.johandry.com/post/kubernetes_client/).

## How to use

Start by importing the package using (`import github.com/johandry/klient`) and making sure it is in your `go.mod` file using the latest version executing any go tool command or just `go mod tidy`. Create the client providing - optionally - the Kubernetes context and the Kubeconfig file location, finally use the methods `Apply()`, `Create()`, `Delete()`, `Replace()` or it's variations to interact with the Kubernetes cluster.

## Example

The following example is to apply a ConfigMap from a `[]byte` variable. It assumes you have a Kubernetes cluster running, accessible and with the Kubeconfig in the default location (`~/.kube/config`).

It uses `klient` to apply the ConfigMap and uses the Kubernetes client `k8s.io/client-go` to get it. This simple example can be done using only `k8s.io/client-go` to do it all as we know, in advance the object to work with (ConfigMap), however we are dealing with a JSON string representing the ConfigMap, that's why we use `klient`.

It's unpractical in a real program to apply a resource and then delete it, but just for the purpose of explain how to delete a resource the example deletes the ConfigMap once it's over.

```go
package main

import (
	"log"

	"github.com/johandry/klient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	name := "apple"
	cm := []byte(`{"apiVersion": "v1", "kind": "ConfigMap", "metadata": { "name": "fruit" }, "data": {	"name": "` + name + `" } }`)

	c := klient.NewClient("", "") // Take the Kubernetes config from the default location (~/.kube/config) and using the default context.
	if err := c.Apply(cm); err != nil {
		log.Fatal("failed to apply the ConfigMap")
	}

	cmFruit, err := c.Clientset.CoreV1().ConfigMaps("default").Get("fruit", metav1.GetOptions{})
	if err != nil {
		log.Fatal("Failed to get the ConfigMap fruits")
	}
	log.Printf("Fruit name: %s", cmFruit.Data["name"])

	if err := c.Delete(cm); err != nil {
		log.Fatal("failed to delete the ConfigMap")
	}
}

```

