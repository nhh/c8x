package k8s

import (
	"testing"
)

// PatchAndTransform with no components
func TestPatchAndTransformEmptyExport(t *testing.T) {
	export := ChartExport{}
	// Should not panic
	PatchAndTransform(export)
}

// PatchAndTransform with nil components slice
func TestPatchAndTransformNilComponents(t *testing.T) {
	export := ChartExport{
		Namespace:  K8sResource{"kind": "Namespace"},
		Components: nil,
	}
	// Should not panic
	PatchAndTransform(export)
}

// Combined with both fields empty
func TestCombinedBothEmpty(t *testing.T) {
	chart := &Chart{}
	result := chart.Combined()
	if result == "" {
		t.Fatal("expected non-empty string with separator")
	}
}
