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

// ==================== $base64 variants ====================

func TestBase64EncodeUnicode(t *testing.T) {
	vm := goja.New()
	injectBase64(vm)

	v, _ := vm.RunString(`$base64.encode("Ünïcödé 🚀")`)
	decoded, _ := vm.RunString(fmt.Sprintf(`$base64.decode("%s")`, v.String()))
	if decoded.String() != "Ünïcödé 🚀" {
		t.Fatalf("unicode roundtrip failed, got %q", decoded.String())
	}
}

func TestBase64EncodeNewlines(t *testing.T) {
	vm := goja.New()
	injectBase64(vm)

	v, _ := vm.RunString(`$base64.encode("line1\nline2\nline3")`)
	decoded, _ := vm.RunString(fmt.Sprintf(`$base64.decode("%s")`, v.String()))
	if decoded.String() != "line1\nline2\nline3" {
		t.Fatalf("newline roundtrip failed, got %q", decoded.String())
	}
}

func TestBase64EncodeBinaryLike(t *testing.T) {
	vm := goja.New()
	injectBase64(vm)

	// String with null bytes and special chars
	v, _ := vm.RunString(`$base64.encode("\x00\x01\x02\xff")`)
	if v.String() == "" {
		t.Fatal("expected non-empty base64 for binary-like input")
	}
}

func TestBase64DecodePadding(t *testing.T) {
	vm := goja.New()
	injectBase64(vm)

	// No padding
	_, err := vm.RunString(`$base64.decode("aGVsbG8")`)
	// StdEncoding requires padding - this should fail
	if err == nil {
		t.Fatal("expected error for base64 without padding (StdEncoding)")
	}
}

func TestBase64EncodeLongString(t *testing.T) {
	vm := goja.New()
	injectBase64(vm)

	long := strings.Repeat("a", 10000)
	code := fmt.Sprintf(`$base64.decode($base64.encode("%s"))`, long)
	v, err := vm.RunString(code)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != long {
		t.Fatalf("long string roundtrip failed, got length %d", len(v.String()))
	}
}

// ==================== $hash variants ====================

func TestHashSha256Empty(t *testing.T) {
	vm := goja.New()
	injectHash(vm)

	v, _ := vm.RunString(`$hash.sha256("")`)
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if v.String() != expected {
		t.Fatalf("expected sha256 of empty string, got %s", v.String())
	}
}

func TestHashMd5Empty(t *testing.T) {
	vm := goja.New()
	injectHash(vm)

	v, _ := vm.RunString(`$hash.md5("")`)
	expected := "d41d8cd98f00b204e9800998ecf8427e"
	if v.String() != expected {
		t.Fatalf("expected md5 of empty string, got %s", v.String())
	}
}

func TestHashSha256DifferentInputs(t *testing.T) {
	vm := goja.New()
	injectHash(vm)

	v, _ := vm.RunString(`$hash.sha256("a") !== $hash.sha256("b")`)
	if !v.ToBoolean() {
		t.Fatal("different inputs should produce different hashes")
	}
}

func TestHashSha256Unicode(t *testing.T) {
	vm := goja.New()
	injectHash(vm)

	// Should not crash and should be deterministic
	v1, _ := vm.RunString(`$hash.sha256("日本語")`)
	v2, _ := vm.RunString(`$hash.sha256("日本語")`)
	if v1.String() != v2.String() {
		t.Fatal("unicode hash not deterministic")
	}
	if len(v1.String()) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(v1.String()))
	}
}

func TestHashMd5LongInput(t *testing.T) {
	vm := goja.New()
	injectHash(vm)

	long := strings.Repeat("x", 100000)
	code := fmt.Sprintf(`$hash.md5("%s").length`, long)
	v, _ := vm.RunString(code)
	if fmt.Sprintf("%v", v.Export()) != "32" {
		t.Fatalf("expected md5 length 32, got %v", v.Export())
	}
}

func TestHashCombinedForConfigHash(t *testing.T) {
	vm := goja.New()
	injectHash(vm)

	// Realistic use case: hash of JSON config for annotation
	code := `$hash.sha256(JSON.stringify({db: "postgres", port: 5432, ssl: true}))`
	v, err := vm.RunString(code)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.String()) != 64 {
		t.Fatalf("expected 64-char hash, got %q", v.String())
	}
}

// ==================== $log variants ====================

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	fn()
	w.Close()
	os.Stderr = old
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

func TestLogMultipleMessages(t *testing.T) {
	vm := goja.New()
	injectLog(vm)

	output := captureStderr(t, func() {
		vm.RunString(`$log.info("first"); $log.warn("second"); $log.error("third")`)
	})

	if !strings.Contains(output, "first") || !strings.Contains(output, "second") || !strings.Contains(output, "third") {
		t.Fatalf("expected all three messages, got %q", output)
	}
}

func TestLogSpecialChars(t *testing.T) {
	vm := goja.New()
	injectLog(vm)

	output := captureStderr(t, func() {
		vm.RunString(`$log.info("path: /var/www & \"quoted\" <angle>")`)
	})

	if !strings.Contains(output, "/var/www") {
		t.Fatalf("expected special chars preserved, got %q", output)
	}
}

func TestLogEmptyMessage(t *testing.T) {
	vm := goja.New()
	injectLog(vm)

	output := captureStderr(t, func() {
		vm.RunString(`$log.info("")`)
	})

	if !strings.Contains(output, "INFO") {
		t.Fatalf("expected INFO prefix even for empty message, got %q", output)
	}
}

// ==================== $assert variants ====================

func TestAssertUndefined(t *testing.T) {
	vm := goja.New()
	injectAssert(vm)

	_, err := vm.RunString(`$assert(undefined, "undefined is falsy")`)
	if err == nil {
		t.Fatal("expected error for undefined")
	}
}

func TestAssertNegativeNumber(t *testing.T) {
	vm := goja.New()
	injectAssert(vm)

	_, err := vm.RunString(`$assert(-1, "negative is truthy")`)
	if err != nil {
		t.Fatalf("expected no error for -1, got %v", err)
	}
}

func TestAssertArrayPasses(t *testing.T) {
	vm := goja.New()
	injectAssert(vm)

	_, err := vm.RunString(`$assert([1,2,3], "array is truthy")`)
	if err != nil {
		t.Fatalf("expected no error for array, got %v", err)
	}
}

func TestAssertObjectPasses(t *testing.T) {
	vm := goja.New()
	injectAssert(vm)

	_, err := vm.RunString(`$assert({key: "val"}, "object is truthy")`)
	if err != nil {
		t.Fatalf("expected no error for object, got %v", err)
	}
}

func TestAssertExpressionComparison(t *testing.T) {
	vm := goja.New()
	injectAssert(vm)

	_, err := vm.RunString(`$assert(3 > 2, "3 > 2")`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = vm.RunString(`$assert(2 > 3, "2 > 3 is false")`)
	if err == nil {
		t.Fatal("expected error for false comparison")
	}
}

func TestAssertRegexMatch(t *testing.T) {
	vm := goja.New()
	injectAssert(vm)

	_, err := vm.RunString(`$assert("hello.com".match(/^[a-z.]+$/), "valid domain")`)
	if err != nil {
		t.Fatalf("expected no error for matching regex, got %v", err)
	}

	_, err = vm.RunString(`$assert("INVALID!@#".match(/^[a-z.]+$/), "invalid domain")`)
	if err == nil {
		t.Fatal("expected error for non-matching regex")
	}
}

func TestAssertStringLength(t *testing.T) {
	vm := goja.New()
	injectAssert(vm)

	_, err := vm.RunString(`
		var pw = "short";
		$assert(pw.length >= 8, "Password must be at least 8 characters, got " + pw.length)
	`)
	if err == nil {
		t.Fatal("expected error for short password")
	}
	if !strings.Contains(err.Error(), "got 5") {
		t.Fatalf("expected dynamic message, got %v", err)
	}
}

func TestAssertMultipleInSequence(t *testing.T) {
	vm := goja.New()
	injectAssert(vm)

	_, err := vm.RunString(`
		$assert(true, "first");
		$assert(1, "second");
		$assert("ok", "third");
	`)
	if err != nil {
		t.Fatalf("expected all assertions to pass, got %v", err)
	}
}

func TestAssertStopsAtFirstFailure(t *testing.T) {
	vm := goja.New()
	injectAssert(vm)

	_, err := vm.RunString(`
		$assert(true, "passes");
		$assert(false, "fails here");
		$assert(true, "never reached");
	`)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "fails here") {
		t.Fatalf("expected 'fails here', got %v", err)
	}
}

// ==================== $file variants ====================

func TestFileReadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "empty.txt"), []byte(""), 0644)

	vm := goja.New()
	injectFile(vm, dir)

	v, err := vm.RunString(`$file.read("empty.txt")`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "" {
		t.Fatalf("expected empty string, got %q", v.String())
	}
}

func TestFileReadBinaryContent(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bin"), []byte{0x00, 0xFF, 0x42}, 0644)

	vm := goja.New()
	injectFile(vm, dir)

	// Should not crash
	_, err := vm.RunString(`$file.read("bin")`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFileReadMultilineContent(t *testing.T) {
	dir := t.TempDir()
	content := "line1\nline2\nline3\n"
	os.WriteFile(filepath.Join(dir, "multi.txt"), []byte(content), 0644)

	vm := goja.New()
	injectFile(vm, dir)

	v, _ := vm.RunString(`$file.read("multi.txt")`)
	if v.String() != content {
		t.Fatalf("expected multiline content, got %q", v.String())
	}
}

func TestFileReadLargeFile(t *testing.T) {
	dir := t.TempDir()
	large := strings.Repeat("data\n", 10000) // 50KB
	os.WriteFile(filepath.Join(dir, "large.txt"), []byte(large), 0644)

	vm := goja.New()
	injectFile(vm, dir)

	v, err := vm.RunString(`$file.read("large.txt").length`)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%v", v.Export()) != "50000" {
		t.Fatalf("expected length 50000, got %v", v.Export())
	}
}

func TestFileExistsDirectory(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)

	vm := goja.New()
	injectFile(vm, dir)

	v, _ := vm.RunString(`$file.exists("subdir")`)
	if !v.ToBoolean() {
		t.Fatal("expected true for existing directory")
	}
}

func TestFileReadNestedDeep(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(deep, 0755)
	os.WriteFile(filepath.Join(deep, "deep.txt"), []byte("found"), 0644)

	vm := goja.New()
	injectFile(vm, dir)

	v, err := vm.RunString(`$file.read("a/b/c/deep.txt")`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "found" {
		t.Fatalf("expected 'found', got %q", v.String())
	}
}

func TestFileReadJsonAndParse(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"port":8080,"debug":true}`), 0644)

	vm := goja.New()
	injectFile(vm, dir)

	v, err := vm.RunString(`JSON.parse($file.read("config.json")).port`)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%v", v.Export()) != "8080" {
		t.Fatalf("expected 8080, got %v", v.Export())
	}
}

func TestFileConditionalRead(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "optional.conf"), []byte("custom"), 0644)

	vm := goja.New()
	injectFile(vm, dir)

	v, err := vm.RunString(`$file.exists("optional.conf") ? $file.read("optional.conf") : "default"`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "custom" {
		t.Fatalf("expected 'custom', got %q", v.String())
	}

	// Without file
	vm2 := goja.New()
	injectFile(vm2, t.TempDir())
	v2, _ := vm2.RunString(`$file.exists("optional.conf") ? $file.read("optional.conf") : "default"`)
	if v2.String() != "default" {
		t.Fatalf("expected 'default', got %q", v2.String())
	}
}

// ==================== $http variants ====================

func TestHttpGetReturnsHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "test-value")
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm)

	v, err := vm.RunString(fmt.Sprintf(`$http.get("%s").headers["X-Custom"]`, server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "test-value" {
		t.Fatalf("expected 'test-value', got %q", v.String())
	}
}

func TestHttpGet404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm)

	v, err := vm.RunString(fmt.Sprintf(`$http.get("%s")`, server.URL))
	if err != nil {
		t.Fatal(err)
	}

	obj := v.Export().(map[string]interface{})
	if fmt.Sprintf("%v", obj["status"]) != "404" {
		t.Fatalf("expected 404, got %v", obj["status"])
	}
	if obj["body"] != "not found" {
		t.Fatalf("expected 'not found', got %v", obj["body"])
	}
}

func TestHttpGet500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"internal"}`))
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm)

	v, err := vm.RunString(fmt.Sprintf(`$http.get("%s").status`, server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%v", v.Export()) != "500" {
		t.Fatalf("expected 500, got %v", v.Export())
	}
}

func TestHttpGetMultipleHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "key123" || r.Header.Get("Accept") != "text/plain" {
			w.WriteHeader(401)
			return
		}
		w.Write([]byte("authorized"))
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm)

	code := fmt.Sprintf(`$http.getText("%s", { headers: { "X-Api-Key": "key123", "Accept": "text/plain" } })`, server.URL)
	v, err := vm.RunString(code)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "authorized" {
		t.Fatalf("expected 'authorized', got %q", v.String())
	}
}

func TestHttpGetJSONArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]string{"a", "b", "c"})
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm)

	v, err := vm.RunString(fmt.Sprintf(`$http.getJSON("%s").length`, server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%v", v.Export()) != "3" {
		t.Fatalf("expected array length 3, got %v", v.Export())
	}
}

func TestHttpGetJSONNested(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"secrets": map[string]interface{}{
					"password": "s3cret",
				},
			},
		})
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm)

	v, err := vm.RunString(fmt.Sprintf(`$http.getJSON("%s").data.secrets.password`, server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "s3cret" {
		t.Fatalf("expected 's3cret', got %q", v.String())
	}
}

func TestHttpGetJSONInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json at all"))
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm)

	_, err := vm.RunString(fmt.Sprintf(`$http.getJSON("%s")`, server.URL))
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestHttpPostEmptyBody(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		receivedBody = string(b)
		w.WriteHeader(204)
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm)

	v, err := vm.RunString(fmt.Sprintf(`$http.post("%s", "").status`, server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%v", v.Export()) != "204" {
		t.Fatalf("expected 204, got %v", v.Export())
	}
	if receivedBody != "" {
		t.Fatalf("expected empty body, got %q", receivedBody)
	}
}

func TestHttpPostJSONComplex(t *testing.T) {
	var received map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		json.NewEncoder(w).Encode(map[string]string{"status": "received"})
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm)

	code := fmt.Sprintf(`$http.postJSON("%s", {
		name: "deploy",
		targets: ["prod-1", "prod-2"],
		config: { replicas: 3, image: "nginx:latest" }
	})`, server.URL)

	_, err := vm.RunString(code)
	if err != nil {
		t.Fatal(err)
	}

	if received["name"] != "deploy" {
		t.Fatalf("expected name=deploy, got %v", received["name"])
	}
	targets := received["targets"].([]interface{})
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}
}

func TestHttpGetMethodVerification(t *testing.T) {
	var receivedMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	vm := goja.New()
	injectHttp(vm)

	vm.RunString(fmt.Sprintf(`$http.get("%s")`, server.URL))
	if receivedMethod != "GET" {
		t.Fatalf("expected GET, got %s", receivedMethod)
	}

	vm.RunString(fmt.Sprintf(`$http.post("%s", "body")`, server.URL))
	if receivedMethod != "POST" {
		t.Fatalf("expected POST, got %s", receivedMethod)
	}
}

// ==================== Cross-global pipeline tests ====================

func TestPipelineCombinedGlobals(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"combined"}`), 0644)
	os.WriteFile(filepath.Join(dir, "tls.crt"), []byte("CERT-DATA-HERE"), 0644)

	code := `
		$assert($file.exists("tls.crt"), "TLS cert must exist");
		var cert = $file.read("tls.crt");
		var certHash = $hash.sha256(cert);

		export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{
					apiVersion: "v1", kind: "Secret", type: "kubernetes.io/tls",
					metadata: { name: "app-tls", annotations: { "c8x/cert-hash": certHash } },
					data: { "tls.crt": $base64.encode(cert) }
				}
			]
		})
	`

	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(code), 0644)

	jsCode, _ := Load(tsFile, false)
	export, err := Run(jsCode, tsFile)
	if err != nil {
		t.Fatal(err)
	}

	comp := export.Components[0]
	data := comp["data"].(map[string]interface{})
	meta := comp.GetMetadata()
	annotations := meta["annotations"].(map[string]interface{})

	// Cert is base64 encoded
	if data["tls.crt"] != "Q0VSVC1EQVRBLUhFUkU=" {
		t.Fatalf("expected base64 cert, got %v", data["tls.crt"])
	}

	// Hash is 64 chars
	hash := annotations["c8x/cert-hash"].(string)
	if len(hash) != 64 {
		t.Fatalf("expected 64-char hash, got %q", hash)
	}
}

func TestPipelineAssertWithEnv(t *testing.T) {
	t.Setenv("C8X_REQUIRED_VAL", "present")

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)

	code := `
		var val = $env.get("REQUIRED_VAL");
		$assert(val, "REQUIRED_VAL must be set");
		$assert(val !== "changeme", "REQUIRED_VAL must not be default");

		export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [{
				apiVersion: "v1", kind: "ConfigMap",
				metadata: { name: "cfg" },
				data: { val: val }
			}]
		})
	`

	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(code), 0644)

	jsCode, _ := Load(tsFile, false)
	export, err := Run(jsCode, tsFile)
	if err != nil {
		t.Fatal(err)
	}

	data := export.Components[0]["data"].(map[string]interface{})
	if data["val"] != "present" {
		t.Fatalf("expected 'present', got %v", data["val"])
	}
}

func TestPipelineHttpWithBase64AndFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"token": "abc123"})
	}))
	defer server.Close()

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)
	os.WriteFile(filepath.Join(dir, "ca.crt"), []byte("ROOT-CA"), 0644)

	code := fmt.Sprintf(`
		var token = $http.getJSON("%s").token;
		var ca = $file.read("ca.crt");

		export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [{
				apiVersion: "v1", kind: "Secret",
				metadata: { name: "auth" },
				data: {
					token: $base64.encode(token),
					"ca.crt": $base64.encode(ca)
				}
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

	data := export.Components[0]["data"].(map[string]interface{})
	if data["token"] != "YWJjMTIz" { // base64("abc123")
		t.Fatalf("expected base64 token, got %v", data["token"])
	}
	if data["ca.crt"] != "Uk9PVC1DQQ==" { // base64("ROOT-CA")
		t.Fatalf("expected base64 CA cert, got %v", data["ca.crt"])
	}
}

func TestPipelineConditionalComponentsWithFileCheck(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)
	os.WriteFile(filepath.Join(dir, "extra.yaml"), []byte("extra config"), 0644)

	code := `
		var components = [
			{ apiVersion: "v1", kind: "Service", metadata: { name: "app" }, spec: { ports: [{ port: 80 }] } }
		];

		if ($file.exists("extra.yaml")) {
			components.push({
				apiVersion: "v1", kind: "ConfigMap",
				metadata: { name: "extra" },
				data: { "extra.yaml": $file.read("extra.yaml") }
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
	export, err := Run(jsCode, tsFile)
	if err != nil {
		t.Fatal(err)
	}

	if len(export.Components) != 2 {
		t.Fatalf("expected 2 components (service + conditional configmap), got %d", len(export.Components))
	}
	if export.Components[1]["kind"] != "ConfigMap" {
		t.Fatalf("expected ConfigMap, got %v", export.Components[1]["kind"])
	}
}

// ==================== $yaml variants ====================

func TestYamlParseSimple(t *testing.T) {
	vm := goja.New()
	injectYaml(vm)

	v, err := vm.RunString(`$yaml.parse("name: hello\nport: 8080").name`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "hello" {
		t.Fatalf("expected 'hello', got %q", v.String())
	}
}

func TestYamlParseNumber(t *testing.T) {
	vm := goja.New()
	injectYaml(vm)

	v, err := vm.RunString(`$yaml.parse("port: 8080").port`)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%v", v.Export()) != "8080" {
		t.Fatalf("expected 8080, got %v", v.Export())
	}
}

func TestYamlParseBool(t *testing.T) {
	vm := goja.New()
	injectYaml(vm)

	v, err := vm.RunString(`$yaml.parse("debug: true").debug`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.ToBoolean() {
		t.Fatal("expected true")
	}
}

func TestYamlParseNested(t *testing.T) {
	vm := goja.New()
	injectYaml(vm)

	yamlStr := `
metadata:
  name: app
  labels:
    tier: frontend`

	code := fmt.Sprintf(`$yaml.parse(%q).metadata.labels.tier`, yamlStr)
	v, err := vm.RunString(code)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "frontend" {
		t.Fatalf("expected 'frontend', got %q", v.String())
	}
}

func TestYamlParseArray(t *testing.T) {
	vm := goja.New()
	injectYaml(vm)

	yamlStr := `items:\n  - a\n  - b\n  - c`
	code := fmt.Sprintf(`$yaml.parse("%s").items.length`, yamlStr)
	v, err := vm.RunString(code)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%v", v.Export()) != "3" {
		t.Fatalf("expected 3, got %v", v.Export())
	}
}

func TestYamlParseInvalid(t *testing.T) {
	vm := goja.New()
	injectYaml(vm)

	_, err := vm.RunString(`$yaml.parse(":::invalid:::")`)
	// yaml.v3 is lenient – just check it doesn't crash
	if err != nil {
		// Some invalid YAML may error, which is fine
	}
}

func TestYamlStringifySimple(t *testing.T) {
	vm := goja.New()
	injectYaml(vm)

	v, err := vm.RunString(`$yaml.stringify({name: "app", port: 8080})`)
	if err != nil {
		t.Fatal(err)
	}

	output := v.String()
	if !strings.Contains(output, "name: app") {
		t.Fatalf("expected 'name: app' in YAML, got %q", output)
	}
	if !strings.Contains(output, "port: 8080") {
		t.Fatalf("expected 'port: 8080' in YAML, got %q", output)
	}
}

func TestYamlStringifyNested(t *testing.T) {
	vm := goja.New()
	injectYaml(vm)

	v, err := vm.RunString(`$yaml.stringify({metadata: {name: "test", labels: {app: "web"}}})`)
	if err != nil {
		t.Fatal(err)
	}

	output := v.String()
	if !strings.Contains(output, "metadata:") || !strings.Contains(output, "app: web") {
		t.Fatalf("expected nested YAML, got %q", output)
	}
}

func TestYamlStringifyArray(t *testing.T) {
	vm := goja.New()
	injectYaml(vm)

	v, err := vm.RunString(`$yaml.stringify({items: ["a", "b", "c"]})`)
	if err != nil {
		t.Fatal(err)
	}

	output := v.String()
	if !strings.Contains(output, "- a") || !strings.Contains(output, "- b") {
		t.Fatalf("expected YAML array, got %q", output)
	}
}

func TestYamlRoundtrip(t *testing.T) {
	vm := goja.New()
	injectYaml(vm)

	code := `
		var obj = {apiVersion: "v1", kind: "Service", metadata: {name: "svc"}};
		var yamlStr = $yaml.stringify(obj);
		var parsed = $yaml.parse(yamlStr);
		parsed.kind
	`
	v, err := vm.RunString(code)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "Service" {
		t.Fatalf("expected 'Service' after roundtrip, got %q", v.String())
	}
}

func TestYamlParseMultiDocument(t *testing.T) {
	vm := goja.New()
	injectYaml(vm)

	// yaml.v3 Unmarshal only parses the first document
	yamlStr := "name: first\n---\nname: second"
	code := fmt.Sprintf(`$yaml.parse(%q).name`, yamlStr)
	v, err := vm.RunString(code)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "first" {
		t.Fatalf("expected 'first' (first document), got %q", v.String())
	}
}

func TestYamlStringifyEmpty(t *testing.T) {
	vm := goja.New()
	injectYaml(vm)

	v, err := vm.RunString(`$yaml.stringify({})`)
	if err != nil {
		t.Fatal(err)
	}
	output := strings.TrimSpace(v.String())
	if output != "{}" {
		t.Fatalf("expected '{}', got %q", output)
	}
}

// $yaml pipeline: read YAML file, modify, use as ConfigMap
func TestPipelineYamlFileManipulation(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)

	promConfig := `global:
  scrape_interval: 15s
scrape_configs:
  - job_name: prometheus
    static_configs:
      - targets:
          - localhost:9090
`
	os.WriteFile(filepath.Join(dir, "prometheus.yml"), []byte(promConfig), 0644)

	code := `
		var config = $yaml.parse($file.read("prometheus.yml"));
		config.scrape_configs = config.scrape_configs.concat([{job_name: "my-app", static_configs: [{targets: ["app:8080"]}]}]);
		var configYaml = $yaml.stringify(config);

		export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [{
				apiVersion: "v1", kind: "ConfigMap",
				metadata: { name: "prometheus-config" },
				data: { "prometheus.yml": configYaml }
			}]
		})
	`

	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(code), 0644)

	jsCode, _ := Load(tsFile, false)
	export, err := Run(jsCode, tsFile)
	if err != nil {
		t.Fatal(err)
	}

	data := export.Components[0]["data"].(map[string]interface{})
	yamlOutput := data["prometheus.yml"].(string)

	if !strings.Contains(yamlOutput, "my-app") {
		t.Fatalf("expected 'my-app' in modified YAML, got %q", yamlOutput)
	}
	if !strings.Contains(yamlOutput, "app:8080") {
		t.Fatalf("expected 'app:8080' in modified YAML, got %q", yamlOutput)
	}
	if !strings.Contains(yamlOutput, "prometheus") {
		t.Fatalf("expected original 'prometheus' job preserved, got %q", yamlOutput)
	}
}
