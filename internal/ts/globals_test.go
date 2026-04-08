package ts

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

// --- $base64 ---

func TestBase64Encode(t *testing.T) {
	vm := goja.New()
	injectBase64(vm)

	v, err := vm.RunString(`$base64.encode("hello")`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "aGVsbG8=" {
		t.Fatalf("expected aGVsbG8=, got %s", v.String())
	}
}

func TestBase64Decode(t *testing.T) {
	vm := goja.New()
	injectBase64(vm)

	v, err := vm.RunString(`$base64.decode("aGVsbG8=")`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "hello" {
		t.Fatalf("expected hello, got %s", v.String())
	}
}

func TestBase64Roundtrip(t *testing.T) {
	vm := goja.New()
	injectBase64(vm)

	v, err := vm.RunString(`$base64.decode($base64.encode("kubernetes rocks"))`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "kubernetes rocks" {
		t.Fatalf("roundtrip failed, got %s", v.String())
	}
}

func TestBase64EncodeEmpty(t *testing.T) {
	vm := goja.New()
	injectBase64(vm)

	v, err := vm.RunString(`$base64.encode("")`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "" {
		t.Fatalf("expected empty string, got %s", v.String())
	}
}

func TestBase64DecodeInvalid(t *testing.T) {
	vm := goja.New()
	injectBase64(vm)

	_, err := vm.RunString(`$base64.decode("!!!not-valid!!!")`)
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

// --- $hash ---

func TestHashSha256(t *testing.T) {
	vm := goja.New()
	injectHash(vm)

	v, err := vm.RunString(`$hash.sha256("hello")`)
	if err != nil {
		t.Fatal(err)
	}
	expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if v.String() != expected {
		t.Fatalf("expected %s, got %s", expected, v.String())
	}
}

func TestHashMd5(t *testing.T) {
	vm := goja.New()
	injectHash(vm)

	v, err := vm.RunString(`$hash.md5("hello")`)
	if err != nil {
		t.Fatal(err)
	}
	expected := "5d41402abc4b2a76b9719d911017c592"
	if v.String() != expected {
		t.Fatalf("expected %s, got %s", expected, v.String())
	}
}

func TestHashDeterministic(t *testing.T) {
	vm := goja.New()
	injectHash(vm)

	v, err := vm.RunString(`$hash.sha256("test") === $hash.sha256("test")`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.ToBoolean() {
		t.Fatal("same input should produce same hash")
	}
}

// --- $log ---

func TestLogInfo(t *testing.T) {
	vm := goja.New()
	injectLog(vm)

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	_, err := vm.RunString(`$log.info("deploying chart")`)
	if err != nil {
		t.Fatal(err)
	}

	w.Close()
	os.Stderr = old

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "INFO") || !strings.Contains(output, "deploying chart") {
		t.Fatalf("expected INFO log, got %q", output)
	}
}

func TestLogWarn(t *testing.T) {
	vm := goja.New()
	injectLog(vm)

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	vm.RunString(`$log.warn("password is default")`)

	w.Close()
	os.Stderr = old

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "WARN") {
		t.Fatalf("expected WARN log, got %q", output)
	}
}

func TestLogError(t *testing.T) {
	vm := goja.New()
	injectLog(vm)

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	vm.RunString(`$log.error("something broke")`)

	w.Close()
	os.Stderr = old

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "ERROR") {
		t.Fatalf("expected ERROR log, got %q", output)
	}
}

// --- $assert ---

func TestAssertTrueNoError(t *testing.T) {
	vm := goja.New()
	injectAssert(vm)

	_, err := vm.RunString(`$assert(true, "should not fail")`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAssertFalseThrows(t *testing.T) {
	vm := goja.New()
	injectAssert(vm)

	_, err := vm.RunString(`$assert(false, "this should fail")`)
	if err == nil {
		t.Fatal("expected error for false condition")
	}
	if !strings.Contains(err.Error(), "this should fail") {
		t.Fatalf("expected message in error, got %v", err)
	}
}

func TestAssertNullThrows(t *testing.T) {
	vm := goja.New()
	injectAssert(vm)

	_, err := vm.RunString(`$assert(null, "null is falsy")`)
	if err == nil {
		t.Fatal("expected error for null")
	}
}

func TestAssertNonEmptyStringPasses(t *testing.T) {
	vm := goja.New()
	injectAssert(vm)

	_, err := vm.RunString(`$assert("hello", "should pass")`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAssertEmptyStringThrows(t *testing.T) {
	vm := goja.New()
	injectAssert(vm)

	_, err := vm.RunString(`$assert("", "empty string is falsy")`)
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestAssertZeroThrows(t *testing.T) {
	vm := goja.New()
	injectAssert(vm)

	_, err := vm.RunString(`$assert(0, "zero is falsy")`)
	if err == nil {
		t.Fatal("expected error for 0")
	}
}

func TestAssertNumberPasses(t *testing.T) {
	vm := goja.New()
	injectAssert(vm)

	_, err := vm.RunString(`$assert(42, "non-zero passes")`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// --- $file ---

func TestFileRead(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.txt"), []byte("key=value"), 0644)

	vm := goja.New()
	injectFile(vm, dir)

	v, err := vm.RunString(`$file.read("config.txt")`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "key=value" {
		t.Fatalf("expected 'key=value', got %q", v.String())
	}
}

func TestFileReadNotFound(t *testing.T) {
	dir := t.TempDir()

	vm := goja.New()
	injectFile(vm, dir)

	_, err := vm.RunString(`$file.read("nope.txt")`)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "exists.txt"), []byte("hi"), 0644)

	vm := goja.New()
	injectFile(vm, dir)

	v, err := vm.RunString(`$file.exists("exists.txt")`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.ToBoolean() {
		t.Fatal("expected true for existing file")
	}
}

func TestFileExistsNotFound(t *testing.T) {
	dir := t.TempDir()

	vm := goja.New()
	injectFile(vm, dir)

	v, err := vm.RunString(`$file.exists("nope.txt")`)
	if err != nil {
		t.Fatal(err)
	}
	if v.ToBoolean() {
		t.Fatal("expected false for missing file")
	}
}

func TestFileReadRelativePath(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "certs")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "tls.crt"), []byte("CERT-DATA"), 0644)

	vm := goja.New()
	injectFile(vm, dir)

	v, err := vm.RunString(`$file.read("certs/tls.crt")`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "CERT-DATA" {
		t.Fatalf("expected 'CERT-DATA', got %q", v.String())
	}
}

// --- $http ---

func TestHttpGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("hello from server"))
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm)

	v, err := vm.RunString(fmt.Sprintf(`$http.get("%s")`, server.URL))
	if err != nil {
		t.Fatal(err)
	}

	obj := v.Export().(map[string]interface{})
	status := fmt.Sprintf("%v", obj["status"])
	if status != "200" {
		t.Fatalf("expected status 200, got %v", obj["status"])
	}
	if obj["body"] != "hello from server" {
		t.Fatalf("expected body, got %v", obj["body"])
	}
}

func TestHttpGetText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("plain text"))
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm)

	v, err := vm.RunString(fmt.Sprintf(`$http.getText("%s")`, server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "plain text" {
		t.Fatalf("expected 'plain text', got %q", v.String())
	}
}

func TestHttpGetJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"key": "value", "num": 42})
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm)

	v, err := vm.RunString(fmt.Sprintf(`JSON.stringify($http.getJSON("%s"))`, server.URL))
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(v.String()), &parsed)

	if parsed["key"] != "value" {
		t.Fatalf("expected key=value, got %v", parsed)
	}
}

func TestHttpPost(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		receivedBody = string(b)
		w.WriteHeader(201)
		w.Write([]byte("created"))
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm)

	v, err := vm.RunString(fmt.Sprintf(`$http.post("%s", "hello body")`, server.URL))
	if err != nil {
		t.Fatal(err)
	}

	obj := v.Export().(map[string]interface{})
	status := fmt.Sprintf("%v", obj["status"])
	if status != "201" {
		t.Fatalf("expected status 201, got %v", obj["status"])
	}
	if receivedBody != "hello body" {
		t.Fatalf("expected 'hello body', got %q", receivedBody)
	}
}

func TestHttpPostJSON(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"result": "ok"})
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm)

	code := fmt.Sprintf(`JSON.stringify($http.postJSON("%s", {msg: "deploy started"}))`, server.URL)
	v, err := vm.RunString(code)
	if err != nil {
		t.Fatal(err)
	}

	if receivedBody["msg"] != "deploy started" {
		t.Fatalf("expected msg in body, got %v", receivedBody)
	}

	var resp map[string]interface{}
	json.Unmarshal([]byte(v.String()), &resp)
	if resp["result"] != "ok" {
		t.Fatalf("expected result=ok, got %v", resp)
	}
}

func TestHttpGetWithHeaders(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm)

	code := fmt.Sprintf(`$http.get("%s", { headers: { "Authorization": "Bearer token123" } })`, server.URL)
	_, err := vm.RunString(code)
	if err != nil {
		t.Fatal(err)
	}

	if receivedAuth != "Bearer token123" {
		t.Fatalf("expected auth header, got %q", receivedAuth)
	}
}

func TestHttpGetUnreachable(t *testing.T) {
	vm := goja.New()
	injectHttp(vm)

	_, err := vm.RunString(`$http.get("http://127.0.0.1:1")`)
	if err == nil {
		t.Fatal("expected error for unreachable host")
	}
}

// --- Pipeline Tests ---

func TestPipelineBase64InSecret(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)

	code := `export default () => ({
		namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
		components: [{
			apiVersion: "v1", kind: "Secret", metadata: { name: "tls" },
			data: { "tls.crt": $base64.encode("CERT-CONTENT") }
		}]
	})`

	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(code), 0644)

	jsCode, err := Load(tsFile, false)
	if err != nil {
		t.Fatal(err)
	}

	export, err := Run(jsCode, tsFile)
	if err != nil {
		t.Fatal(err)
	}

	data := export.Components[0]["data"].(map[string]interface{})
	if data["tls.crt"] != "Q0VSVC1DT05URU5U" {
		t.Fatalf("expected base64 encoded cert, got %v", data["tls.crt"])
	}
}

func TestPipelineHashAsAnnotation(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)

	code := `export default () => ({
		namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
		components: [{
			apiVersion: "apps/v1", kind: "Deployment",
			metadata: { name: "app", annotations: { "c8x/config-hash": $hash.sha256("config-data") } },
			spec: {}
		}]
	})`

	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(code), 0644)

	jsCode, _ := Load(tsFile, false)
	export, err := Run(jsCode, tsFile)
	if err != nil {
		t.Fatal(err)
	}

	meta := export.Components[0].GetMetadata()
	annotations := meta["annotations"].(map[string]interface{})
	hash := annotations["c8x/config-hash"].(string)

	if len(hash) != 64 {
		t.Fatalf("expected 64-char sha256 hash, got %q", hash)
	}
}

func TestPipelineAssertFails(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)

	code := `
		$assert(false, "DB_PASSWORD must be set");
		export default () => ({ components: [] })
	`

	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(code), 0644)

	jsCode, _ := Load(tsFile, false)
	_, err := Run(jsCode, tsFile)
	if err == nil {
		t.Fatal("expected error from $assert")
	}
	if !strings.Contains(err.Error(), "DB_PASSWORD must be set") {
		t.Fatalf("expected assertion message in error, got %v", err)
	}
}

func TestPipelineFileReadInConfigMap(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)
	os.WriteFile(filepath.Join(dir, "nginx.conf"), []byte("server { listen 80; }"), 0644)

	code := `export default () => ({
		namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
		components: [{
			apiVersion: "v1", kind: "ConfigMap", metadata: { name: "nginx" },
			data: { "nginx.conf": $file.read("nginx.conf") }
		}]
	})`

	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(code), 0644)

	jsCode, _ := Load(tsFile, false)
	export, err := Run(jsCode, tsFile)
	if err != nil {
		t.Fatal(err)
	}

	data := export.Components[0]["data"].(map[string]interface{})
	if data["nginx.conf"] != "server { listen 80; }" {
		t.Fatalf("expected nginx config, got %v", data["nginx.conf"])
	}
}

func TestPipelineHttpInChart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"replicas": 3})
	}))
	defer server.Close()

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)

	code := fmt.Sprintf(`
		var config = $http.getJSON("%s");
		export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [{
				apiVersion: "apps/v1", kind: "Deployment",
				metadata: { name: "app" },
				spec: { replicas: config.replicas }
			}]
		})
	`, server.URL)

	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(code), 0644)

	jsCode, _ := Load(tsFile, false)
	export, err := Run(jsCode, tsFile)
	if err != nil {
		t.Fatal(err)
	}

	spec := export.Components[0]["spec"].(map[string]interface{})
	replicas := fmt.Sprintf("%v", spec["replicas"])
	if replicas != "3" {
		t.Fatalf("expected replicas=3 from HTTP, got %v (%T)", spec["replicas"], spec["replicas"])
	}
}
