package k8s

// K8sResource represents an arbitrary Kubernetes resource.
// Map-based because spec fields vary per Kind, and YAML marshaling requires it.
type K8sResource map[string]interface{}

// GetMetadata returns the metadata map, or nil if not present.
func (r K8sResource) GetMetadata() map[string]interface{} {
	if r == nil {
		return nil
	}
	m, _ := r["metadata"].(map[string]interface{})
	return m
}

// GetName returns metadata.name, or empty string if not present.
func (r K8sResource) GetName() string {
	meta := r.GetMetadata()
	if meta == nil {
		return ""
	}
	name, _ := meta["name"].(string)
	return name
}

// SetNamespace sets metadata.namespace, creating the metadata map if needed.
func (r K8sResource) SetNamespace(ns string) {
	if r == nil {
		return
	}
	meta := r.GetMetadata()
	if meta == nil {
		meta = make(map[string]interface{})
		r["metadata"] = meta
	}
	meta["namespace"] = ns
}

// ChartExport is the typed result of executing a chart's default function.
type ChartExport struct {
	Namespace  K8sResource
	Components []K8sResource
}

// NamespaceName returns the namespace name from the export, or empty string.
func (e ChartExport) NamespaceName() string {
	return e.Namespace.GetName()
}
