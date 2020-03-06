package kubectl

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/resource"
)

// Delete creates a resource with the given content
func (c *Client) Delete(content []byte) error {
	r := c.ResultForContent(content, true)
	return c.DeleteResource(r)
}

// DeleteFiles create the resource(s) from the given filenames (file, directory or STDIN) or HTTP URLs
func (c *Client) DeleteFiles(filenames ...string) error {
	r := c.ResultForFilenameParam(filenames, true)
	return c.DeleteResource(r)
}

// DeleteResource applies the given resource. Create the resources with `ResultForFilenameParam` or `ResultForContent`
func (c *Client) DeleteResource(r *resource.Result) error {
	return r.Visit(delete)
}

func delete(info *resource.Info, err error) error {
	if err != nil {
		return failedTo("delete", info, err)
	}

	policy := metav1.DeletePropagationBackground
	options := metav1.DeleteOptions{PropagationPolicy: &policy}

	if _, err := resource.NewHelper(info.Client, info.Mapping).DeleteWithOptions(info.Namespace, info.Name, &options); err != nil {
		return failedTo("delete", info, err)
	}
	return nil
}
