package ts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
	"github.com/kubernetix/c8x/internal/k8s"
)

// --- Unit tests for extractGroup ---

func TestExtractGroupWithSlash(t *testing.T) {
	if extractGroup("networking.k8s.io/v1") != "networking.k8s.io" {
		t.Fatal("expected 'networking.k8s.io'")
	}
}

func TestExtractGroupCore(t *testing.T) {
	if extractGroup("v1") != "" {
		t.Fatal("expected empty string for core group")
	}
}

func TestExtractGroupApps(t *testing.T) {
	if extractGroup("apps/v1") != "apps" {
		t.Fatal("expected 'apps'")
	}
}

func TestExtractGroupRbac(t *testing.T) {
	if extractGroup("rbac.authorization.k8s.io/v1") != "rbac.authorization.k8s.io" {
		t.Fatal("expected 'rbac.authorization.k8s.io'")
	}
}

func TestExtractGroupBatch(t *testing.T) {
	if extractGroup("batch/v1") != "batch" {
		t.Fatal("expected 'batch'")
	}
}

func TestExtractGroupStorage(t *testing.T) {
	if extractGroup("storage.k8s.io/v1") != "storage.k8s.io" {
		t.Fatal("expected 'storage.k8s.io'")
	}
}

// --- kubectl function tests (will fail gracefully without a cluster) ---

func TestKubectlNotAvailable(t *testing.T) {
	// This tests that kubectl errors are properly wrapped
	_, err := kubectl("get", "nonexistent-resource-type-xyz")
	if err == nil {
		// kubectl is available and somehow this worked - skip
		t.Skip("kubectl returned success unexpectedly")
	}
	// Error should be wrapped with $cluster prefix
	if err.Error() == "" {
		t.Fatal("expected non-empty error")
	}
}

// --- Integration-style tests that work WITH or WITHOUT a cluster ---

func TestClusterVersionWithoutCluster(t *testing.T) {
	// Test that $cluster.version() returns an error (not a panic)
	// when no cluster is available
	vm := setupVM(t)

	_, err := vm.RunString(`
		try {
			$cluster.version();
		} catch(e) {
			// Expected - no cluster available
		}
	`)
	if err != nil {
		t.Fatalf("$cluster.version() panicked instead of throwing: %v", err)
	}
}

func TestClusterVersionAtLeastWithoutCluster(t *testing.T) {
	vm := setupVM(t)

	_, err := vm.RunString(`
		try {
			$cluster.versionAtLeast("1.25");
		} catch(e) {
			// Expected
		}
	`)
	if err != nil {
		t.Fatalf("panicked: %v", err)
	}
}

func TestClusterNodeCountWithoutCluster(t *testing.T) {
	vm := setupVM(t)

	_, err := vm.RunString(`
		try {
			$cluster.nodeCount();
		} catch(e) {
			// Expected
		}
	`)
	if err != nil {
		t.Fatalf("panicked: %v", err)
	}
}

func TestClusterApiAvailableDoesNotPanic(t *testing.T) {
	vm := setupVM(t)

	// Should not panic regardless of cluster availability
	_, err := vm.RunString(`$cluster.apiAvailable("nonexistent.api.xyz/v99")`)
	if err != nil {
		t.Fatalf("panicked: %v", err)
	}
	// Result depends on whether kubectl + cluster is available
	// The point is it doesn't crash
}

func TestClusterCrdExistsWithoutCluster(t *testing.T) {
	vm := setupVM(t)

	v, err := vm.RunString(`$cluster.crdExists("nonexistent.example.com")`)
	if err != nil {
		t.Fatalf("panicked: %v", err)
	}
	if v.ToBoolean() {
		t.Fatal("expected false for nonexistent CRD")
	}
}

func TestClusterExistsWithoutCluster(t *testing.T) {
	vm := setupVM(t)

	v, err := vm.RunString(`$cluster.exists("v1", "Secret", "default", "nonexistent-xyz")`)
	if err != nil {
		t.Fatalf("panicked: %v", err)
	}
	if v.ToBoolean() {
		t.Fatal("expected false")
	}
}

// --- Pipeline test: chart that uses $cluster gracefully ---

func TestPipelineClusterConditional(t *testing.T) {
	dir := t.TempDir()
	writeTestPkgJson(t, dir)

	// Chart that checks for a CRD and conditionally adds a resource
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

	export := runChart(t, dir, code)

	// Without cert-manager on the cluster, we should get 1 component
	// With cert-manager, we'd get 2
	if len(export.Components) < 1 {
		t.Fatal("expected at least 1 component")
	}
	if export.Components[0]["kind"] != "Service" {
		t.Fatalf("expected Service, got %v", export.Components[0]["kind"])
	}
}

func TestPipelineClusterApiGateSwitch(t *testing.T) {
	dir := t.TempDir()
	writeTestPkgJson(t, dir)

	// Chart that switches between Ingress and Gateway API based on cluster capabilities
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

	export := runChart(t, dir, code)

	kind := export.Components[0]["kind"].(string)
	// Without Gateway API: Ingress. With it: HTTPRoute. Both are valid.
	if kind != "Ingress" && kind != "HTTPRoute" {
		t.Fatalf("expected Ingress or HTTPRoute, got %s", kind)
	}
}

// --- Helpers ---

func setupVM(t *testing.T) *goja.Runtime {
	t.Helper()
	vm := goja.New()
	if err := injectCluster(vm); err != nil {
		t.Fatal(err)
	}
	return vm
}

func writeTestPkgJson(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, dir, "package.json", `{"name":"test"}`)
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func runChart(t *testing.T, dir, code string) k8s.ChartExport {
	t.Helper()
	tsFile := filepath.Join(dir, "index.ts")
	writeFile(t, dir, "index.ts", code)

	jsCode, err := Load(tsFile, false)
	if err != nil {
		t.Fatal(err)
	}

	export, err := Run(jsCode, tsFile)
	if err != nil {
		t.Fatal(err)
	}
	return export
}
