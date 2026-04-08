package ts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- __jsEnvGet ---

func TestJsEnvGetString(t *testing.T) {
	t.Setenv("C8X_FOO", "hello")

	result := __jsEnvGet("FOO")
	if result != "hello" {
		t.Fatalf("expected 'hello', got %v", result)
	}
}

func TestJsEnvGetInt(t *testing.T) {
	t.Setenv("C8X_PORT", "8080")

	result := __jsEnvGet("PORT")
	if result != 8080 {
		t.Fatalf("expected 8080 (int), got %v (%T)", result, result)
	}
}

func TestJsEnvGetBoolTrue(t *testing.T) {
	t.Setenv("C8X_ENABLED", "true")

	result := __jsEnvGet("ENABLED")
	if result != true {
		t.Fatalf("expected true, got %v", result)
	}
}

func TestJsEnvGetBoolFalse(t *testing.T) {
	t.Setenv("C8X_DISABLED", "false")

	result := __jsEnvGet("DISABLED")
	if result != false {
		t.Fatalf("expected false, got %v", result)
	}
}

func TestJsEnvGetNotFound(t *testing.T) {
	result := __jsEnvGet("NONEXISTENT_VAR_XYZ")
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestJsEnvGetTrimsWhitespace(t *testing.T) {
	t.Setenv("C8X_SPACED", " bar ")

	result := __jsEnvGet("SPACED")
	if result != "bar" {
		t.Fatalf("expected 'bar', got '%v'", result)
	}
}

func TestJsEnvGetIgnoresNonC8XVars(t *testing.T) {
	t.Setenv("PLAIN_VAR", "nope")

	result := __jsEnvGet("PLAIN_VAR")
	if result != nil {
		t.Fatalf("expected nil for non-C8X_ var, got %v", result)
	}
}

// --- __jsEnvGetAsObject ---

func TestJsEnvGetAsObjectBasic(t *testing.T) {
	t.Setenv("C8X_ANNOTATIONS_KEY_1", "nginx.ingress.kubernetes.io/app-root")
	t.Setenv("C8X_ANNOTATIONS_VALUE_1", "/var/www/html")

	result := __jsEnvGetAsObject("ANNOTATIONS")
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if m["nginx.ingress.kubernetes.io/app-root"] != "/var/www/html" {
		t.Fatalf("expected '/var/www/html', got %v", m["nginx.ingress.kubernetes.io/app-root"])
	}
}

func TestJsEnvGetAsObjectMultiple(t *testing.T) {
	t.Setenv("C8X_LABELS_KEY_1", "app")
	t.Setenv("C8X_LABELS_VALUE_1", "web")
	t.Setenv("C8X_LABELS_KEY_2", "tier")
	t.Setenv("C8X_LABELS_VALUE_2", "frontend")

	result := __jsEnvGetAsObject("LABELS")
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if m["app"] != "web" {
		t.Fatalf("expected 'web', got %v", m["app"])
	}
	if m["tier"] != "frontend" {
		t.Fatalf("expected 'frontend', got %v", m["tier"])
	}
}

func TestJsEnvGetAsObjectTypeParsing(t *testing.T) {
	t.Setenv("C8X_CFG_KEY_1", "port")
	t.Setenv("C8X_CFG_VALUE_1", "3000")
	t.Setenv("C8X_CFG_KEY_2", "debug")
	t.Setenv("C8X_CFG_VALUE_2", "true")
	t.Setenv("C8X_CFG_KEY_3", "host")
	t.Setenv("C8X_CFG_VALUE_3", "localhost")

	result := __jsEnvGetAsObject("CFG")
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if m["port"] != 3000 {
		t.Fatalf("expected 3000 (int), got %v (%T)", m["port"], m["port"])
	}
	if m["debug"] != true {
		t.Fatalf("expected true, got %v", m["debug"])
	}
	if m["host"] != "localhost" {
		t.Fatalf("expected 'localhost', got %v", m["host"])
	}
}

func TestJsEnvGetAsObjectEmpty(t *testing.T) {
	result := __jsEnvGetAsObject("NONEXISTENT_PREFIX_XYZ")
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if len(m) != 0 {
		t.Fatalf("expected empty map, got %v", m)
	}
}

// --- Load ---

func TestLoadTranspilesTypeScript(t *testing.T) {
	dir := t.TempDir()
	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(`const x: number = 1; export default () => ({ components: [] })`), 0644)

	result, err := Load(tsFile, false)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "c8x") {
		t.Fatalf("expected IIFE with 'c8x' global, got: %s", result[:min(len(result), 200)])
	}
}

func TestLoadTranspilesJavaScript(t *testing.T) {
	dir := t.TempDir()
	jsFile := filepath.Join(dir, "index.js")
	os.WriteFile(jsFile, []byte(`export default () => ({ components: [] })`), 0644)

	result, err := Load(jsFile, false)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "c8x") {
		t.Fatalf("expected IIFE with 'c8x' global, got: %s", result[:min(len(result), 200)])
	}
}

func TestLoadBundlesImports(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "helper.ts"), []byte(`export const greeting = "hello"`), 0644)
	os.WriteFile(filepath.Join(dir, "index.ts"), []byte(`import { greeting } from "./helper"; export default () => ({ components: [], msg: greeting })`), 0644)

	result, err := Load(filepath.Join(dir, "index.ts"), false)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "hello") {
		t.Fatalf("expected bundled output to contain 'hello' from import, got: %s", result[:min(len(result), 300)])
	}
}

func TestLoadReturnsErrorForInvalidFile(t *testing.T) {
	_, err := Load("/nonexistent/path/index.ts", false)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// --- Run ---

func helperPkgJson(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)
	return dir
}

func TestRunSimpleExport(t *testing.T) {
	dir := helperPkgJson(t)

	code := `var c8x = { default: function() { return {
		namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "default" } },
		components: []
	} } };`

	result, err := Run(code, filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Components) != 0 {
		t.Fatalf("expected empty components, got %d", len(result.Components))
	}
}

func TestRunWithEnv(t *testing.T) {
	t.Setenv("C8X_MYVAL", "test-value")
	dir := helperPkgJson(t)

	// $env.get returns into a JS property that becomes part of the raw map.
	// Since ChartExport only captures namespace+components, we test env via a component.
	code := `var c8x = { default: function() { return {
		namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "default" } },
		components: [
			{ kind: "ConfigMap", metadata: { name: $env.get("MYVAL") } }
		]
	} } };`

	result, err := Run(code, filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatal(err)
	}

	if result.Components[0].GetName() != "test-value" {
		t.Fatalf("expected 'test-value', got %v", result.Components[0].GetName())
	}
}

func TestRunPatchesNamespace(t *testing.T) {
	dir := helperPkgJson(t)

	code := `var c8x = { default: function() { return {
		namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "my-ns" } },
		components: [
			{ apiVersion: "apps/v1", kind: "Deployment", metadata: { name: "app" } }
		]
	} } };`

	result, err := Run(code, filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatal(err)
	}

	meta := result.Components[0].GetMetadata()
	if meta["namespace"] != "my-ns" {
		t.Fatalf("expected namespace 'my-ns' patched into component, got %v", meta["namespace"])
	}
}

func TestRunWithoutNamespace(t *testing.T) {
	dir := helperPkgJson(t)

	code := `var c8x = { default: function() { return {
		components: [
			{ apiVersion: "apps/v1", kind: "Deployment", metadata: { name: "app" } }
		]
	} } };`

	result, err := Run(code, filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatal(err)
	}

	meta := result.Components[0].GetMetadata()
	if _, exists := meta["namespace"]; exists {
		t.Fatal("expected no namespace patching when namespace is not set")
	}
}

func TestRunNilComponentsSkipped(t *testing.T) {
	dir := helperPkgJson(t)

	code := `var c8x = { default: function() { return {
		namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "my-ns" } },
		components: [
			null,
			{ apiVersion: "apps/v1", kind: "Deployment", metadata: { name: "app" } },
			null
		]
	} } };`

	result, err := Run(code, filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Components) != 3 {
		t.Fatalf("expected 3 components (including nils), got %d", len(result.Components))
	}
}

func TestRunWithChartInfo(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"my-chart","version":"1.0.0"}`), 0644)

	code := `var c8x = { default: function() { return {
		namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "default" } },
		components: [
			{ kind: "ConfigMap", metadata: { name: $chart.name } }
		]
	} } };`

	result, err := Run(code, filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatal(err)
	}

	if result.Components[0].GetName() != "my-chart" {
		t.Fatalf("expected 'my-chart', got %v", result.Components[0].GetName())
	}
}

func TestRunReturnsErrorForInvalidCode(t *testing.T) {
	dir := helperPkgJson(t)

	_, err := Run("this is not valid javascript {{{{", filepath.Join(dir, "index.ts"))
	if err == nil {
		t.Fatal("expected error for invalid JS code")
	}
}

func TestRunReturnsErrorForMissingDefault(t *testing.T) {
	dir := helperPkgJson(t)

	_, err := Run("var c8x = { notDefault: 1 };", filepath.Join(dir, "index.ts"))
	if err == nil {
		t.Fatal("expected error when default export is missing")
	}
}
