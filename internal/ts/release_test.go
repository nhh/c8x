package ts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseNotInjectedWithoutOptions(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)

	// Chart that safely checks if $release exists
	code := `
		var hasRelease = typeof $release !== 'undefined';

		export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [{
				apiVersion: "v1", kind: "ConfigMap",
				metadata: { name: "info" },
				data: { hasRelease: String(hasRelease) }
			}]
		})
	`

	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(code), 0644)
	jsCode, _ := Load(tsFile, false)

	// Without namespace/releaseName → $release is not injected
	export, err := Run(jsCode, tsFile, Permissions{})
	if err != nil {
		t.Fatal(err)
	}

	data := export.Components[0]["data"].(map[string]interface{})
	if data["hasRelease"] != "false" {
		t.Fatalf("expected hasRelease=false, got %v", data["hasRelease"])
	}
}

func TestReleaseConditionalMigration(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)

	// Chart that adds a migration job only on first install
	code := `
		var isFirstInstall = typeof $release === 'undefined' || !$release.exists;

		var components = [
			{ apiVersion: "v1", kind: "Service", metadata: { name: "app" }, spec: { ports: [{ port: 80 }] } }
		];

		if (isFirstInstall) {
			components.push({
				apiVersion: "batch/v1", kind: "Job",
				metadata: { name: "migration" },
				spec: { template: { spec: { containers: [{ name: "migrate", image: "migrate:latest" }], restartPolicy: "Never" } } }
			});
		}

		export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: components
		})
	`

	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(code), 0644)
	jsCode, _ := Load(tsFile, false)

	// Without $release injected → typeof $release === 'undefined' → first install
	export, err := Run(jsCode, tsFile, AllPermissions())
	if err != nil {
		t.Fatal(err)
	}

	if len(export.Components) != 2 {
		t.Fatalf("expected 2 components (service + migration job), got %d", len(export.Components))
	}
	if export.Components[1]["kind"] != "Job" {
		t.Fatalf("expected Job, got %v", export.Components[1]["kind"])
	}
}

func TestReleaseRevisionAsLabel(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)

	code := `
		var rev = typeof $release !== 'undefined' && $release.exists ? $release.revision : 0;

		export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [{
				apiVersion: "apps/v1", kind: "Deployment",
				metadata: { name: "app", labels: { "c8x.io/revision": String(rev) } },
				spec: {}
			}]
		})
	`

	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(code), 0644)
	jsCode, _ := Load(tsFile, false)

	export, err := Run(jsCode, tsFile, AllPermissions())
	if err != nil {
		t.Fatal(err)
	}

	meta := export.Components[0].GetMetadata()
	labels := meta["labels"].(map[string]interface{})
	if labels["c8x.io/revision"] != "0" {
		t.Fatalf("expected revision 0 (no release), got %v", labels["c8x.io/revision"])
	}
}

func TestReleaseEnvAccess(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)

	code := `
		var prevDomain = typeof $release !== 'undefined' && $release.exists
			? $release.env.WP_DOMAIN || "none"
			: "none";

		export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [{
				apiVersion: "v1", kind: "ConfigMap",
				metadata: { name: "info" },
				data: { prevDomain: prevDomain }
			}]
		})
	`

	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(code), 0644)
	jsCode, _ := Load(tsFile, false)

	export, err := Run(jsCode, tsFile, AllPermissions())
	if err != nil {
		t.Fatal(err)
	}

	data := export.Components[0]["data"].(map[string]interface{})
	if data["prevDomain"] != "none" {
		t.Fatalf("expected 'none' (no release), got %v", data["prevDomain"])
	}
}

func TestReleaseDowngradeWarning(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test","version":"1.0.0"}`), 0644)

	code := `
		if (typeof $release !== 'undefined' && $release.exists && $release.chartVersion > $chart.version) {
			$log.warn("Downgrading from " + $release.chartVersion + " to " + $chart.version);
		}

		export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: []
		})
	`

	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(code), 0644)
	jsCode, _ := Load(tsFile, false)

	// Without release → no warning, no crash
	_, err := Run(jsCode, tsFile, AllPermissions())
	if err != nil {
		t.Fatal(err)
	}
}

// Test that $release access patterns don't crash even without injection
func TestReleaseUndefinedSafe(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)

	patterns := []string{
		`typeof $release === 'undefined'`,
		`typeof $release !== 'undefined' && $release.exists`,
		`typeof $release !== 'undefined' ? $release.revision : 0`,
	}

	for _, pattern := range patterns {
		code := `
			var result = ` + pattern + `;
			export default () => ({
				namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
				components: []
			})
		`

		tsFile := filepath.Join(dir, "index.ts")
		os.WriteFile(tsFile, []byte(code), 0644)
		jsCode, _ := Load(tsFile, false)

		_, err := Run(jsCode, tsFile, AllPermissions())
		if err != nil {
			if strings.Contains(err.Error(), "not defined") {
				// Expected – $release not injected
				continue
			}
			t.Fatalf("pattern %q failed: %v", pattern, err)
		}
	}
}
