package kube

import (
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericclioptions/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/kubernetes/pkg/kubectl/cmd/util/openapi"
	"k8s.io/kubernetes/pkg/kubectl/validation"
)

// Config is the client configuration. It implements RESTClientGetter
type Config struct {
	KubeConfig    string
	Context       string
	openAPIGetter openAPIGetter
}

type openAPIGetter struct {
	once   sync.Once
	getter openapi.Getter
}

var _ genericclioptions.RESTClientGetter = &Config{}

// NewConfig creates a new client configuration
func NewConfig(context, kubeconfig string) *Config {
	return &Config{
		KubeConfig: kubeconfig,
		Context:    context,
	}
}

// BuildRESTConfig builds a kubernetes REST client config using the following
// rules from ToRawKubeConfigLoader()
func BuildRESTConfig(context, kubeconfig string) (*rest.Config, error) {
	return NewConfig(context, kubeconfig).ToRESTConfig()
}

// ToRESTConfig creates a kubernetes REST client config
func (c *Config) ToRESTConfig() (*rest.Config, error) {
	config, err := c.ToRawKubeConfigLoader().ClientConfig()
	if err != nil {
		return nil, err
	}

	rest.SetKubernetesDefaults(config)
	return config, nil
}

// ToRawKubeConfigLoader creates a client config using the following rules:
// 1. builds from the given kubeconfig path, if not empty
// 2. use the in cluster config if running in-cluster
// 3. gets the config from KUBECONFIG env var
// 4. Uses $HOME/.kube/config
func (c *Config) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	loadingRules.ExplicitPath = c.KubeConfig
	configOverrides := &clientcmd.ConfigOverrides{
		ClusterDefaults: clientcmd.ClusterDefaults,
		CurrentContext:  c.Context,
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
}

// overlyCautiousIllegalFileCharacters matches characters that *might* not be supported.  Windows is really restrictive, so this is really restrictive
var overlyCautiousIllegalFileCharacters = regexp.MustCompile(`[^(\w/\.)]`)

// ToDiscoveryClient returns a CachedDiscoveryInterface using a computed RESTConfig
func (c *Config) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config, err := c.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	config.Burst = 100
	defaultHTTPCacheDir := filepath.Join(homedir.HomeDir(), ".kube", "http-cache")

	// takes the parentDir and the host and comes up with a "usually non-colliding" name for the discoveryCacheDir
	parentDir := filepath.Join(homedir.HomeDir(), ".kube", "cache", "discovery")
	// strip the optional scheme from host if its there:
	schemelessHost := strings.Replace(strings.Replace(config.Host, "https://", "", 1), "http://", "", 1)
	// now do a simple collapse of non-AZ09 characters.  Collisions are possible but unlikely.  Even if we do collide the problem is short lived
	safeHost := overlyCautiousIllegalFileCharacters.ReplaceAllString(schemelessHost, "_")
	discoveryCacheDir := filepath.Join(parentDir, safeHost)

	return discovery.NewCachedDiscoveryClientForConfig(config, discoveryCacheDir, defaultHTTPCacheDir, time.Duration(10*time.Minute))
}

// ToRESTMapper returns a mapper
func (c *Config) ToRESTMapper() (meta.RESTMapper, error) {
	discoveryClient, err := c.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	expander := restmapper.NewShortcutExpander(mapper, discoveryClient)
	return expander, nil
}

// KubernetesClientSet creates a kubernetes clientset from the configuration
func (c *Config) KubernetesClientSet() (*kubernetes.Clientset, error) {
	config, err := c.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

// DynamicClient creates a dynamic client from the configuration
func (c *Config) DynamicClient() (dynamic.Interface, error) {
	config, err := c.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(config)
}

// NewBuilder returns a new resource builder for structured api objects.
func (c *Config) NewBuilder() *resource.Builder {
	return resource.NewBuilder(c)
}

// RESTClient creates a REST client from the configuration
func (c *Config) RESTClient() (*rest.RESTClient, error) {
	config, err := c.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return rest.RESTClientFor(config)
}

// ClientForMapping creates a resource REST client from the given mappings
func (c *Config) ClientForMapping(mapping *meta.RESTMapping) (resource.RESTClient, error) {
	config, err := c.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	gvk := mapping.GroupVersionKind
	switch gvk.Group {
	case corev1.GroupName:
		config.APIPath = "/api"
	default:
		config.APIPath = "/apis"
	}
	gv := gvk.GroupVersion()
	config.GroupVersion = &gv

	return rest.RESTClientFor(config)
}

// UnstructuredClientForMapping creates a unstructured resource REST client from the given mappings
func (c *Config) UnstructuredClientForMapping(mapping *meta.RESTMapping) (resource.RESTClient, error) {
	config, err := c.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	config.APIPath = "/apis"
	if mapping.GroupVersionKind.Group == corev1.GroupName {
		config.APIPath = "/api"
	}
	gv := mapping.GroupVersionKind.GroupVersion()
	config.ContentConfig = resource.UnstructuredPlusDefaultContentConfig()
	config.GroupVersion = &gv

	return rest.RESTClientFor(config)
}

// Validator returns a schema that can validate objects stored on disk.
func (c *Config) Validator(validate bool) (validation.Schema, error) {
	if !validate {
		return validation.NullSchema{}, nil
	}

	resources, err := c.OpenAPISchema()
	if err != nil {
		return nil, err
	}

	return validation.ConjunctiveSchema{
		openapivalidation.NewSchemaValidation(resources),
		validation.NoDoubleKeySchema{},
	}, nil
}

// OpenAPISchema returns metadata and structural information about Kubernetes object definitions.
func (c *Config) OpenAPISchema() (openapi.Resources, error) {
	discovery, err := c.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	// Lazily initialize the OpenAPIGetter once
	c.openAPIGetter.once.Do(func() {
		// Create the caching OpenAPIGetter
		c.openAPIGetter.getter = openapi.NewOpenAPIGetter(discovery)
	})

	// Delegate to the OpenAPIGetter
	return c.openAPIGetter.getter.Get()
}
