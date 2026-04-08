package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps the Kubernetes API clients needed by c8x.
type Client struct {
	clientset       kubernetes.Interface
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
	mapper          *restmapper.DeferredDiscoveryRESTMapper
}

// NewClient creates a new Kubernetes client using the default kubeconfig
// resolution order: $KUBECONFIG → ~/.kube/config → in-cluster.
func NewClient() (*Client, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules, &clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("loading kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	cachedDiscovery := memory.NewMemCacheClient(clientset.Discovery())
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscovery)

	return &Client{
		clientset:       clientset,
		dynamicClient:   dynamicClient,
		discoveryClient: clientset.Discovery(),
		mapper:          mapper,
	}, nil
}

// Apply performs a server-side apply for each YAML document in the given bytes.
// Documents are separated by "---". Returns a summary of what was applied.
func (c *Client) Apply(yamlBytes []byte) (string, error) {
	var results []string

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(yamlBytes), 4096)
	for {
		var obj unstructured.Unstructured
		err := decoder.Decode(&obj)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("decoding YAML: %w", err)
		}

		if obj.GetAPIVersion() == "" || obj.GetKind() == "" {
			continue
		}

		gvk := obj.GroupVersionKind()
		mapping, err := c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return "", fmt.Errorf("mapping %s: %w", gvk.String(), err)
		}

		var resource dynamic.ResourceInterface
		if obj.GetNamespace() != "" {
			resource = c.dynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
		} else {
			resource = c.dynamicClient.Resource(mapping.Resource)
		}

		result, err := resource.Apply(
			context.TODO(),
			obj.GetName(),
			&obj,
			metav1.ApplyOptions{FieldManager: "c8x", Force: true},
		)
		if err != nil {
			return "", fmt.Errorf("applying %s/%s: %w", obj.GetKind(), obj.GetName(), err)
		}

		results = append(results, fmt.Sprintf("%s/%s configured", strings.ToLower(result.GetKind()), result.GetName()))
	}

	return strings.Join(results, "\n"), nil
}

// ServerVersion returns the cluster's Kubernetes version as "major.minor".
func (c *Client) ServerVersion() (string, error) {
	sv, err := c.discoveryClient.ServerVersion()
	if err != nil {
		return "", fmt.Errorf("getting server version: %w", err)
	}
	return sv.Major + "." + strings.TrimRight(sv.Minor, "+"), nil
}

// NodeCount returns the number of nodes in the cluster.
func (c *Client) NodeCount() (int, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, fmt.Errorf("listing nodes: %w", err)
	}
	return len(nodes.Items), nil
}

// APIAvailable returns true if the given API group is available on the cluster.
func (c *Client) APIAvailable(apiVersion string) bool {
	group := ExtractGroup(apiVersion)
	_, resources, err := c.discoveryClient.ServerGroupsAndResources()
	if err != nil {
		return false
	}
	for _, r := range resources {
		if group == "" {
			// core group: check for "v1"
			if r.GroupVersion == apiVersion {
				return true
			}
		} else {
			if strings.HasPrefix(r.GroupVersion, group+"/") {
				return true
			}
		}
	}
	return false
}

// CRDExists returns true if a CRD with the given name exists.
func (c *Client) CRDExists(name string) bool {
	crdGVR := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
	_, err := c.dynamicClient.Resource(crdGVR).Get(context.TODO(), name, metav1.GetOptions{})
	return err == nil
}

// ResourceExists returns true if the specified resource exists in the cluster.
func (c *Client) ResourceExists(kind, namespace, name string) bool {
	gvr, err := c.resolveGVR(kind)
	if err != nil {
		return false
	}

	var resource dynamic.ResourceInterface
	if namespace != "" {
		resource = c.dynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		resource = c.dynamicClient.Resource(gvr)
	}

	_, err = resource.Get(context.TODO(), name, metav1.GetOptions{})
	return err == nil
}

// ListResources lists resources of the given kind, optionally in a namespace.
func (c *Client) ListResources(kind string, namespace string) ([]map[string]interface{}, error) {
	gvr, err := c.resolveGVR(kind)
	if err != nil {
		return nil, err
	}

	var resource dynamic.ResourceInterface
	if namespace != "" {
		resource = c.dynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		resource = c.dynamicClient.Resource(gvr)
	}

	list, err := resource.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", kind, err)
	}

	result := make([]map[string]interface{}, len(list.Items))
	for i, item := range list.Items {
		result[i] = item.Object
	}
	return result, nil
}

// resolveGVR resolves a kind string (like "Service", "Deployment") to a GroupVersionResource.
func (c *Client) resolveGVR(kind string) (schema.GroupVersionResource, error) {
	resources, err := c.discoveryClient.ServerPreferredResources()
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("discovering resources: %w", err)
	}

	for _, resList := range resources {
		gv, _ := schema.ParseGroupVersion(resList.GroupVersion)
		for _, r := range resList.APIResources {
			if strings.EqualFold(r.Kind, kind) {
				return schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: r.Name,
				}, nil
			}
		}
	}
	return schema.GroupVersionResource{}, fmt.Errorf("unknown resource kind: %s", kind)
}

// ExtractGroup extracts the API group from an apiVersion string.
// "networking.k8s.io/v1" → "networking.k8s.io", "v1" → ""
func ExtractGroup(apiVersion string) string {
	parts := strings.SplitN(apiVersion, "/", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return ""
}
