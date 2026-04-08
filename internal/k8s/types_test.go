package k8s

import (
	"testing"
)

func TestK8sResourceGetMetadata(t *testing.T) {
	r := K8sResource{
		"metadata": map[string]interface{}{"name": "test"},
	}
	meta := r.GetMetadata()
	if meta == nil {
		t.Fatal("expected metadata")
	}
	if meta["name"] != "test" {
		t.Fatalf("expected 'test', got %v", meta["name"])
	}
}

func TestK8sResourceGetMetadataNil(t *testing.T) {
	var r K8sResource
	if r.GetMetadata() != nil {
		t.Fatal("expected nil for nil resource")
	}
}

func TestK8sResourceGetMetadataMissing(t *testing.T) {
	r := K8sResource{"kind": "Service"}
	if r.GetMetadata() != nil {
		t.Fatal("expected nil when no metadata")
	}
}

func TestK8sResourceGetName(t *testing.T) {
	r := K8sResource{
		"metadata": map[string]interface{}{"name": "my-app"},
	}
	if r.GetName() != "my-app" {
		t.Fatalf("expected 'my-app', got %q", r.GetName())
	}
}

func TestK8sResourceGetNameEmpty(t *testing.T) {
	r := K8sResource{"kind": "Service"}
	if r.GetName() != "" {
		t.Fatalf("expected empty string, got %q", r.GetName())
	}
}

func TestK8sResourceSetNamespace(t *testing.T) {
	r := K8sResource{
		"metadata": map[string]interface{}{"name": "app"},
	}
	r.SetNamespace("production")

	meta := r.GetMetadata()
	if meta["namespace"] != "production" {
		t.Fatalf("expected 'production', got %v", meta["namespace"])
	}
}

func TestK8sResourceSetNamespaceCreatesMetadata(t *testing.T) {
	r := K8sResource{"kind": "ConfigMap"}
	r.SetNamespace("default")

	meta := r.GetMetadata()
	if meta == nil {
		t.Fatal("expected metadata to be created")
	}
	if meta["namespace"] != "default" {
		t.Fatalf("expected 'default', got %v", meta["namespace"])
	}
}

func TestK8sResourceSetNamespaceNilResource(t *testing.T) {
	var r K8sResource
	// Should not panic
	r.SetNamespace("test")
}

func TestChartExportNamespaceName(t *testing.T) {
	export := ChartExport{
		Namespace: K8sResource{
			"metadata": map[string]interface{}{"name": "my-ns"},
		},
	}
	if export.NamespaceName() != "my-ns" {
		t.Fatalf("expected 'my-ns', got %q", export.NamespaceName())
	}
}

func TestChartExportNamespaceNameEmpty(t *testing.T) {
	export := ChartExport{}
	if export.NamespaceName() != "" {
		t.Fatalf("expected empty, got %q", export.NamespaceName())
	}
}
