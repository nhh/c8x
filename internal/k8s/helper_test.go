package k8s

import (
	"strings"
	"testing"
)

// --- PatchAndTransform ---

func TestPatchAndTransformBasic(t *testing.T) {
	export := ChartExport{
		Components: []K8sResource{
			{"apiVersion": "apps/v1", "kind": "Deployment"},
		},
	}

	chart := PatchAndTransform(export)

	if !strings.Contains(chart.Content, "Deployment") {
		t.Fatalf("expected Content to contain 'Deployment', got: %s", chart.Content)
	}

	if chart.Namespace != "" {
		t.Fatalf("expected empty Namespace, got: %s", chart.Namespace)
	}
}

func TestPatchAndTransformMultipleComponents(t *testing.T) {
	export := ChartExport{
		Components: []K8sResource{
			{"kind": "Deployment"},
			{"kind": "Service"},
			{"kind": "Ingress"},
		},
	}

	chart := PatchAndTransform(export)

	count := strings.Count(chart.Content, "---\n")
	if count < 3 {
		t.Fatalf("expected at least 3 separators for 3 components, got %d in: %s", count, chart.Content)
	}

	if !strings.Contains(chart.Content, "Deployment") {
		t.Fatal("expected Content to contain 'Deployment'")
	}
	if !strings.Contains(chart.Content, "Service") {
		t.Fatal("expected Content to contain 'Service'")
	}
	if !strings.Contains(chart.Content, "Ingress") {
		t.Fatal("expected Content to contain 'Ingress'")
	}
}

func TestPatchAndTransformWithNamespace(t *testing.T) {
	export := ChartExport{
		Namespace: K8sResource{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata":   map[string]interface{}{"name": "my-ns"},
		},
		Components: []K8sResource{},
	}

	chart := PatchAndTransform(export)

	if chart.Namespace == "" {
		t.Fatal("expected non-empty Namespace")
	}
	if !strings.Contains(chart.Namespace, "Namespace") {
		t.Fatalf("expected Namespace YAML to contain 'Namespace', got: %s", chart.Namespace)
	}
}

func TestPatchAndTransformWithoutNamespace(t *testing.T) {
	export := ChartExport{
		Components: []K8sResource{},
	}

	chart := PatchAndTransform(export)

	if chart.Namespace != "" {
		t.Fatalf("expected empty Namespace, got: %s", chart.Namespace)
	}
}

func TestPatchAndTransformNilComponentSkipped(t *testing.T) {
	export := ChartExport{
		Components: []K8sResource{
			nil,
			{"kind": "Service"},
			nil,
		},
	}

	chart := PatchAndTransform(export)

	if !strings.Contains(chart.Content, "Service") {
		t.Fatal("expected Content to contain 'Service'")
	}
}

// --- Combined ---

func TestCombinedBothSet(t *testing.T) {
	chart := &Chart{
		Namespace: "namespace-yaml\n",
		Content:   "content-yaml\n",
	}

	result := chart.Combined()

	if !strings.Contains(result, "namespace-yaml") {
		t.Fatal("expected Combined to contain namespace")
	}
	if !strings.Contains(result, "content-yaml") {
		t.Fatal("expected Combined to contain content")
	}
	if !strings.Contains(result, "---\n") {
		t.Fatal("expected Combined to contain YAML separator")
	}
}

func TestCombinedEmptyNamespace(t *testing.T) {
	chart := &Chart{
		Namespace: "",
		Content:   "content-yaml\n",
	}

	result := chart.Combined()

	if !strings.Contains(result, "content-yaml") {
		t.Fatal("expected Combined to contain content")
	}
}
