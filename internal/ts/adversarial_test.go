package ts

import (
	"os"
	"path/filepath"
	"testing"
)

// Run with namespace set but components key missing
func TestRunNamespaceWithoutComponents(t *testing.T) {
	dir := helperPkgJson(t)

	code := `var c8x = { default: function() { return {
		namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "my-ns" } }
	} } };`

	result, err := Run(code, filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Components) != 0 {
		t.Fatalf("expected 0 components, got %d", len(result.Components))
	}
}

// Run with components as non-array
func TestRunComponentsNotArray(t *testing.T) {
	dir := helperPkgJson(t)

	code := `var c8x = { default: function() { return {
		namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "my-ns" } },
		components: "not an array"
	} } };`

	_, err := Run(code, filepath.Join(dir, "index.ts"))
	if err == nil {
		t.Fatal("expected error when components is not an array")
	}
}

// Run with default export that throws
func TestRunDefaultFunctionThrows(t *testing.T) {
	dir := helperPkgJson(t)

	code := `var c8x = { default: function() { throw new Error("chart broken"); } };`

	_, err := Run(code, filepath.Join(dir, "index.ts"))
	if err == nil {
		t.Fatal("expected error when default function throws")
	}
}

// Run with default export returning null
func TestRunDefaultReturnsNull(t *testing.T) {
	dir := helperPkgJson(t)

	code := `var c8x = { default: function() { return null; } };`

	_, err := Run(code, filepath.Join(dir, "index.ts"))
	if err == nil {
		t.Fatal("expected error when default returns null")
	}
}

// Run with default export returning a primitive
func TestRunDefaultReturnsPrimitive(t *testing.T) {
	dir := helperPkgJson(t)

	code := `var c8x = { default: function() { return 42; } };`

	_, err := Run(code, filepath.Join(dir, "index.ts"))
	if err == nil {
		t.Fatal("expected error when default returns primitive")
	}
}

// Run with malformed package.json
func TestRunMalformedPackageJson(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{not valid json`), 0644)

	code := `var c8x = { default: function() { return { components: [] } } };`

	_, err := Run(code, filepath.Join(dir, "index.ts"))
	if err == nil {
		t.Fatal("expected error for malformed package.json")
	}
}

// Run with no package.json should still work
func TestRunNoPackageJson(t *testing.T) {
	dir := t.TempDir()

	code := `var c8x = { default: function() { return {
		namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "default" } },
		components: []
	} } };`

	result, err := Run(code, filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatalf("expected success without package.json, got: %v", err)
	}

	if result.Namespace == nil {
		t.Fatal("expected non-nil namespace")
	}
}

// __jsEnvGet with env var whose value contains =
func TestJsEnvGetValueWithEquals(t *testing.T) {
	t.Setenv("C8X_CONNSTR", "postgres://u:p@host/db?ssl=true&timeout=30")

	result := __jsEnvGet("CONNSTR")
	expected := "postgres://u:p@host/db?ssl=true&timeout=30"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

// __jsEnvGet with empty value
func TestJsEnvGetEmptyValue(t *testing.T) {
	t.Setenv("C8X_EMPTYVAL", "")

	result := __jsEnvGet("EMPTYVAL")
	if result != "" {
		t.Fatalf("expected empty string, got %v (%T)", result, result)
	}
}

// __jsEnvGet where name is a prefix of another var
func TestJsEnvGetPartialMatch(t *testing.T) {
	t.Setenv("C8X_TEST_EXTRA", "nope")

	result := __jsEnvGet("TEST")
	if result != nil {
		t.Fatalf("expected nil (no exact match), got %v", result)
	}
}

// Component with metadata as string instead of map
func TestRunComponentMetadataNotMap(t *testing.T) {
	dir := helperPkgJson(t)

	code := `var c8x = { default: function() { return {
		namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "ns" } },
		components: [
			{ apiVersion: "v1", kind: "Service", metadata: "not a map" }
		]
	} } };`

	// Should not panic - SetNamespace handles non-map metadata gracefully
	result, err := Run(code, filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The component's metadata was overwritten by SetNamespace since it wasn't a map
	meta := result.Components[0].GetMetadata()
	if meta == nil {
		t.Fatal("expected metadata to exist after SetNamespace")
	}
}

// Component with no metadata at all
func TestRunComponentNoMetadata(t *testing.T) {
	dir := helperPkgJson(t)

	code := `var c8x = { default: function() { return {
		namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "ns" } },
		components: [
			{ apiVersion: "v1", kind: "ConfigMap" }
		]
	} } };`

	result, err := Run(code, filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := result.Components[0].GetMetadata()
	if meta == nil {
		t.Fatal("expected metadata map to be created by SetNamespace")
	}
	if meta["namespace"] != "ns" {
		t.Fatalf("expected namespace 'ns', got %v", meta["namespace"])
	}
}
