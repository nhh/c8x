package ts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
	"github.com/kubernetix/c8x/internal/k8s"
)

// --- Unit tests for compareVersions ---

func TestCompareVersionsEqual(t *testing.T) {
	if !compareVersions("1.25", "1.25") {
		t.Fatal("1.25 >= 1.25 should be true")
	}
}

func TestCompareVersionsHigher(t *testing.T) {
	if !compareVersions("1.31", "1.25") {
		t.Fatal("1.31 >= 1.25 should be true")
	}
}

func TestCompareVersionsLower(t *testing.T) {
	if compareVersions("1.20", "1.25") {
		t.Fatal("1.20 >= 1.25 should be false")
	}
}

func TestCompareVersionsMajorHigher(t *testing.T) {
	if !compareVersions("2.0", "1.99") {
		t.Fatal("2.0 >= 1.99 should be true")
	}
}

func TestCompareVersionsWithPlus(t *testing.T) {
	if !compareVersions("1.31+", "1.25") {
		t.Fatal("1.31+ >= 1.25 should be true")
	}
}

func TestCompareVersionsMajorOnly(t *testing.T) {
	if !compareVersions("2", "1.25") {
		t.Fatal("2 >= 1.25 should be true")
	}
}

// --- Unit tests for ExtractGroup (now in k8s package) ---

func TestExtractGroupWithSlash(t *testing.T) {
	if k8s.ExtractGroup("networking.k8s.io/v1") != "networking.k8s.io" {
		t.Fatal("expected 'networking.k8s.io'")
	}
}

func TestExtractGroupCore(t *testing.T) {
	if k8s.ExtractGroup("v1") != "" {
		t.Fatal("expected empty string for core group")
	}
}

func TestExtractGroupApps(t *testing.T) {
	if k8s.ExtractGroup("apps/v1") != "apps" {
		t.Fatal("expected 'apps'")
	}
}

func TestExtractGroupRbac(t *testing.T) {
	if k8s.ExtractGroup("rbac.authorization.k8s.io/v1") != "rbac.authorization.k8s.io" {
		t.Fatal("expected 'rbac.authorization.k8s.io'")
	}
}

// --- $cluster injection tests (work without a cluster) ---

func TestClusterVersionWithoutCluster(t *testing.T) {
	vm := setupClusterVM(t)

	_, err := vm.RunString(`
		try { $cluster.version(); } catch(e) { /* expected */ }
	`)
	if err != nil {
		t.Fatalf("$cluster.version() panicked instead of throwing: %v", err)
	}
}

func TestClusterVersionAtLeastWithoutCluster(t *testing.T) {
	vm := setupClusterVM(t)

	_, err := vm.RunString(`
		try { $cluster.versionAtLeast("1.25"); } catch(e) { /* expected */ }
	`)
	if err != nil {
		t.Fatalf("panicked: %v", err)
	}
}

func TestClusterNodeCountWithoutCluster(t *testing.T) {
	vm := setupClusterVM(t)

	_, err := vm.RunString(`
		try { $cluster.nodeCount(); } catch(e) { /* expected */ }
	`)
	if err != nil {
		t.Fatalf("panicked: %v", err)
	}
}

func TestClusterApiAvailableDoesNotPanic(t *testing.T) {
	vm := setupClusterVM(t)

	_, err := vm.RunString(`$cluster.apiAvailable("nonexistent.api.xyz/v99")`)
	if err != nil {
		t.Fatalf("panicked: %v", err)
	}
}

func TestClusterCrdExistsDoesNotPanic(t *testing.T) {
	vm := setupClusterVM(t)

	_, err := vm.RunString(`$cluster.crdExists("nonexistent.example.com")`)
	if err != nil {
		t.Fatalf("panicked: %v", err)
	}
}

func TestClusterExistsDoesNotPanic(t *testing.T) {
	vm := setupClusterVM(t)

	_, err := vm.RunString(`$cluster.exists("v1", "Secret", "default", "nonexistent-xyz")`)
	if err != nil {
		t.Fatalf("panicked: %v", err)
	}
}

// --- Pipeline tests ---

func TestPipelineClusterConditional(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "package.json", `{"name":"test"}`)

	code := `
		var hasCertManager = $cluster.crdExists("certificates.cert-manager.io");
		var components = [
			{ apiVersion: "v1", kind: "Service", metadata: { name: "app" }, spec: { ports: [{ port: 80 }] } }
		];
		if (hasCertManager) {
			components.push({
				apiVersion: "cert-manager.io/v1", kind: "Certificate",
				metadata: { name: "app-tls" },
				spec: { secretName: "app-tls", issuerRef: { name: "letsencrypt" } }
			});
		}
		export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: components
		})
	`

	export := runTestChart(t, dir, code)

	if len(export.Components) < 1 {
		t.Fatal("expected at least 1 component")
	}
	if export.Components[0]["kind"] != "Service" {
		t.Fatalf("expected Service, got %v", export.Components[0]["kind"])
	}
}

func TestPipelineClusterApiGateSwitch(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "package.json", `{"name":"test"}`)

	code := `
		var useGateway = $cluster.apiAvailable("gateway.networking.k8s.io/v1");
		var ingress = useGateway
			? { apiVersion: "gateway.networking.k8s.io/v1", kind: "HTTPRoute", metadata: { name: "app" }, spec: {} }
			: { apiVersion: "networking.k8s.io/v1", kind: "Ingress", metadata: { name: "app" }, spec: {} };
		export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [ingress]
		})
	`

	export := runTestChart(t, dir, code)

	kind := export.Components[0]["kind"].(string)
	if kind != "Ingress" && kind != "HTTPRoute" {
		t.Fatalf("expected Ingress or HTTPRoute, got %s", kind)
	}
}

// --- Helpers ---

func setupClusterVM(t *testing.T) *goja.Runtime {
	t.Helper()
	vm := goja.New()
	if err := injectCluster(vm, AllPermissions()); err != nil {
		t.Fatal(err)
	}
	return vm
}

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func runTestChart(t *testing.T, dir, code string) k8s.ChartExport {
	t.Helper()
	tsFile := filepath.Join(dir, "index.ts")
	writeTestFile(t, dir, "index.ts", code)

	jsCode, err := Load(tsFile, false)
	if err != nil {
		t.Fatal(err)
	}

	export, err := Run(jsCode, tsFile, AllPermissions())
	if err != nil {
		t.Fatal(err)
	}
	return export
}
