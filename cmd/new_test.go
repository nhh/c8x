package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderPackageJson(t *testing.T) {
	result, err := renderPackageJson("my-chart")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(result), `"name": "my-chart"`) {
		t.Fatalf("expected name 'my-chart' in output, got: %s", string(result))
	}
}

func TestRenderPackageJsonValidJSON(t *testing.T) {
	result, err := renderPackageJson("test-app")
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("expected valid JSON, got parse error: %v\nContent: %s", err, string(result))
	}

	if parsed["name"] != "test-app" {
		t.Fatalf("expected name 'test-app', got %v", parsed["name"])
	}
}

func TestInitChart(t *testing.T) {
	dir := t.TempDir()

	err := initChart(dir, "wordpress")
	if err != nil {
		t.Fatal(err)
	}

	chartDir := filepath.Join(dir, "wordpress")

	// Check directory was created
	info, err := os.Stat(chartDir)
	if err != nil {
		t.Fatalf("chart directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected directory, got file")
	}

	// Check package.json exists
	pkgPath := filepath.Join(chartDir, "package.json")
	if _, err := os.Stat(pkgPath); err != nil {
		t.Fatalf("package.json not created: %v", err)
	}

	// Check chart.ts exists
	chartPath := filepath.Join(chartDir, "chart.ts")
	if _, err := os.Stat(chartPath); err != nil {
		t.Fatalf("chart.ts not created: %v", err)
	}
}

func TestInitChartContent(t *testing.T) {
	dir := t.TempDir()

	err := initChart(dir, "my-app")
	if err != nil {
		t.Fatal(err)
	}

	// Verify package.json content
	pkgContent, err := os.ReadFile(filepath.Join(dir, "my-app", "package.json"))
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(pkgContent, &parsed); err != nil {
		t.Fatalf("package.json is not valid JSON: %v", err)
	}

	if parsed["name"] != "my-app" {
		t.Fatalf("expected name 'my-app', got %v", parsed["name"])
	}

	// Verify chart.ts content
	chartContent, err := os.ReadFile(filepath.Join(dir, "my-app", "chart.ts"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(chartContent), "import {Chart} from") {
		t.Fatal("chart.ts should contain c8x import")
	}

	if !strings.Contains(string(chartContent), "export default") {
		t.Fatal("chart.ts should contain default export")
	}
}

func TestInitChartAlreadyExists(t *testing.T) {
	dir := t.TempDir()

	// Create the directory first
	os.Mkdir(filepath.Join(dir, "existing"), 0777)

	err := initChart(dir, "existing")
	if err == nil {
		t.Fatal("expected error when chart directory already exists")
	}
}
