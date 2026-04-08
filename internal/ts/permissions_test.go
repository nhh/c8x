package ts

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

// --- $file permissions ---

func TestFileReadDeniedByDefault(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("password"), 0644)

	vm := goja.New()
	injectFile(vm, dir, NoPermissions())

	_, err := vm.RunString(`$file.read("secret.txt")`)
	if err == nil {
		t.Fatal("expected permission denied")
	}
	if !strings.Contains(err.Error(), "allow-file") {
		t.Fatalf("expected allow-file hint, got %v", err)
	}
}

func TestFileExistsDeniedByDefault(t *testing.T) {
	dir := t.TempDir()

	vm := goja.New()
	injectFile(vm, dir, NoPermissions())

	_, err := vm.RunString(`$file.exists("anything.txt")`)
	if err == nil {
		t.Fatal("expected permission denied")
	}
}

func TestFileReadAllowed(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("allowed"), 0644)

	vm := goja.New()
	injectFile(vm, dir, Permissions{File: true})

	v, err := vm.RunString(`$file.read("ok.txt")`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "allowed" {
		t.Fatalf("expected 'allowed', got %q", v.String())
	}
}

// --- $file path traversal ---

func TestFileReadPathTraversalBlocked(t *testing.T) {
	dir := t.TempDir()

	vm := goja.New()
	injectFile(vm, dir, Permissions{File: true})

	_, err := vm.RunString(`$file.read("../../etc/passwd")`)
	if err == nil {
		t.Fatal("expected path traversal to be blocked")
	}
	if !strings.Contains(err.Error(), "escapes chart directory") {
		t.Fatalf("expected escape error, got %v", err)
	}
}

func TestFileReadAbsolutePathBlocked(t *testing.T) {
	dir := t.TempDir()

	vm := goja.New()
	injectFile(vm, dir, Permissions{File: true})

	_, err := vm.RunString(`$file.read("/etc/passwd")`)
	if err == nil {
		t.Fatal("expected absolute path outside chart dir to be blocked")
	}
}

func TestFileExistsPathTraversalBlocked(t *testing.T) {
	dir := t.TempDir()

	vm := goja.New()
	injectFile(vm, dir, Permissions{File: true})

	_, err := vm.RunString(`$file.exists("../../../etc/passwd")`)
	if err == nil {
		t.Fatal("expected path traversal to be blocked")
	}
}

func TestFileReadSubdirAllowed(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "certs")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "tls.crt"), []byte("CERT"), 0644)

	vm := goja.New()
	injectFile(vm, dir, Permissions{File: true})

	v, err := vm.RunString(`$file.read("certs/tls.crt")`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "CERT" {
		t.Fatalf("expected 'CERT', got %q", v.String())
	}
}

func TestFileReadDotDotInsideChartDirAllowed(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "a", "b"), 0755)
	os.WriteFile(filepath.Join(dir, "a", "root.txt"), []byte("ok"), 0644)

	vm := goja.New()
	injectFile(vm, dir, Permissions{File: true})

	// a/b/../root.txt resolves to a/root.txt – still inside chart dir
	v, err := vm.RunString(`$file.read("a/b/../root.txt")`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "ok" {
		t.Fatalf("expected 'ok', got %q", v.String())
	}
}

// --- $http permissions ---

func TestHttpDeniedByDefault(t *testing.T) {
	vm := goja.New()
	injectHttp(vm, NoPermissions())

	_, err := vm.RunString(`$http.get("http://example.com")`)
	if err == nil {
		t.Fatal("expected permission denied")
	}
	if !strings.Contains(err.Error(), "allow-http") {
		t.Fatalf("expected allow-http hint, got %v", err)
	}
}

func TestHttpGetTextDenied(t *testing.T) {
	vm := goja.New()
	injectHttp(vm, NoPermissions())

	_, err := vm.RunString(`$http.getText("http://example.com")`)
	if err == nil {
		t.Fatal("expected permission denied")
	}
}

func TestHttpGetJSONDenied(t *testing.T) {
	vm := goja.New()
	injectHttp(vm, NoPermissions())

	_, err := vm.RunString(`$http.getJSON("http://example.com")`)
	if err == nil {
		t.Fatal("expected permission denied")
	}
}

func TestHttpPostDenied(t *testing.T) {
	vm := goja.New()
	injectHttp(vm, NoPermissions())

	_, err := vm.RunString(`$http.post("http://example.com", "body")`)
	if err == nil {
		t.Fatal("expected permission denied")
	}
}

func TestHttpPostJSONDenied(t *testing.T) {
	vm := goja.New()
	injectHttp(vm, NoPermissions())

	_, err := vm.RunString(`$http.postJSON("http://example.com", {})`)
	if err == nil {
		t.Fatal("expected permission denied")
	}
}

func TestHttpAllowed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm, Permissions{Http: true})

	v, err := vm.RunString(fmt.Sprintf(`$http.getText("%s")`, server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "ok" {
		t.Fatalf("expected 'ok', got %q", v.String())
	}
}

// --- $cluster permissions ---

func TestClusterDeniedByDefault(t *testing.T) {
	vm := goja.New()
	injectCluster(vm, NoPermissions())

	_, err := vm.RunString(`$cluster.version()`)
	if err == nil {
		t.Fatal("expected permission denied")
	}
	if !strings.Contains(err.Error(), "allow-cluster") {
		t.Fatalf("expected allow-cluster hint, got %v", err)
	}
}

func TestClusterNodeCountDenied(t *testing.T) {
	vm := goja.New()
	injectCluster(vm, NoPermissions())

	_, err := vm.RunString(`$cluster.nodeCount()`)
	if err == nil {
		t.Fatal("expected permission denied")
	}
}

func TestClusterApiAvailableDeniedReturnsFalse(t *testing.T) {
	vm := goja.New()
	injectCluster(vm, NoPermissions())

	v, err := vm.RunString(`$cluster.apiAvailable("apps/v1")`)
	if err != nil {
		t.Fatal(err)
	}
	if v.ToBoolean() {
		t.Fatal("expected false when permission denied")
	}
}

func TestClusterCrdExistsDeniedReturnsFalse(t *testing.T) {
	vm := goja.New()
	injectCluster(vm, NoPermissions())

	v, err := vm.RunString(`$cluster.crdExists("anything")`)
	if err != nil {
		t.Fatal(err)
	}
	if v.ToBoolean() {
		t.Fatal("expected false when permission denied")
	}
}

func TestClusterExistsDeniedReturnsFalse(t *testing.T) {
	vm := goja.New()
	injectCluster(vm, NoPermissions())

	v, err := vm.RunString(`$cluster.exists("v1", "Secret", "default", "x")`)
	if err != nil {
		t.Fatal(err)
	}
	if v.ToBoolean() {
		t.Fatal("expected false when permission denied")
	}
}

func TestClusterListDenied(t *testing.T) {
	vm := goja.New()
	injectCluster(vm, NoPermissions())

	_, err := vm.RunString(`$cluster.list("pods")`)
	if err == nil {
		t.Fatal("expected permission denied")
	}
}

// --- AllPermissions / --allow-all ---

func TestAllPermissions(t *testing.T) {
	p := AllPermissions()
	if !p.File || !p.Http || !p.Cluster {
		t.Fatal("AllPermissions should enable everything")
	}
}

func TestNoPermissions(t *testing.T) {
	p := NoPermissions()
	if p.File || p.Http || p.Cluster {
		t.Fatal("NoPermissions should disable everything")
	}
}

// --- Pipeline: permissions flow through Run ---

func TestRunDeniesFileByDefault(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)
	os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("password"), 0644)

	code := `export default () => ({
		namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
		components: [{ apiVersion: "v1", kind: "ConfigMap", metadata: { name: "x" },
			data: { secret: $file.read("secret.txt") } }]
	})`

	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(code), 0644)
	jsCode, _ := Load(tsFile, false)

	// Default: denied
	_, err := Run(jsCode, tsFile)
	if err == nil {
		t.Fatal("expected $file.read to be denied by default")
	}

	// With permission: allowed
	_, err = Run(jsCode, tsFile, Permissions{File: true})
	if err != nil {
		t.Fatalf("expected success with File permission, got %v", err)
	}
}

func TestRunDeniesHttpByDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"key":"val"}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)

	code := fmt.Sprintf(`
		var data = $http.getJSON("%s");
		export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: []
		})
	`, server.URL)

	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(code), 0644)
	jsCode, _ := Load(tsFile, false)

	_, err := Run(jsCode, tsFile)
	if err == nil {
		t.Fatal("expected $http to be denied by default")
	}

	_, err = Run(jsCode, tsFile, Permissions{Http: true})
	if err != nil {
		t.Fatalf("expected success with Http permission, got %v", err)
	}
}
