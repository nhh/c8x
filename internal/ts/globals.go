package ts

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
	"gopkg.in/yaml.v3"
)

// --- $base64 ---

func injectBase64(vm *goja.Runtime) error {
	obj := vm.NewObject()

	obj.Set("encode", func(s string) string {
		return base64.StdEncoding.EncodeToString([]byte(s))
	})

	obj.Set("decode", func(s string) (string, error) {
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return "", fmt.Errorf("base64 decode: %w", err)
		}
		return string(b), nil
	})

	return vm.Set("$base64", obj)
}

// --- $hash ---

func injectHash(vm *goja.Runtime) error {
	obj := vm.NewObject()

	obj.Set("sha256", func(s string) string {
		h := sha256.Sum256([]byte(s))
		return fmt.Sprintf("%x", h)
	})

	obj.Set("md5", func(s string) string {
		h := md5.Sum([]byte(s))
		return fmt.Sprintf("%x", h)
	})

	return vm.Set("$hash", obj)
}

// --- $log ---

func injectLog(vm *goja.Runtime) error {
	obj := vm.NewObject()

	obj.Set("info", func(msg string) {
		fmt.Fprintf(os.Stderr, "[c8x] INFO  %s\n", msg)
	})

	obj.Set("warn", func(msg string) {
		fmt.Fprintf(os.Stderr, "[c8x] WARN  %s\n", msg)
	})

	obj.Set("error", func(msg string) {
		fmt.Fprintf(os.Stderr, "[c8x] ERROR %s\n", msg)
	})

	return vm.Set("$log", obj)
}

// --- $assert ---

func injectAssert(vm *goja.Runtime) error {
	return vm.Set("$assert", func(condition interface{}, message string) error {
		truthy := true
		switch v := condition.(type) {
		case nil:
			truthy = false
		case bool:
			truthy = v
		case int64:
			truthy = v != 0
		case float64:
			truthy = v != 0
		case string:
			truthy = v != ""
		case *goja.Object:
			truthy = v != nil
		}

		if !truthy {
			return fmt.Errorf("assertion failed: %s", message)
		}
		return nil
	})
}

// --- $file ---

func injectFile(vm *goja.Runtime, chartDir string) error {
	obj := vm.NewObject()

	resolve := func(path string) string {
		if filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(chartDir, path)
	}

	obj.Set("read", func(path string) (string, error) {
		data, err := os.ReadFile(resolve(path))
		if err != nil {
			return "", fmt.Errorf("$file.read: %w", err)
		}
		return string(data), nil
	})

	obj.Set("exists", func(path string) bool {
		_, err := os.Stat(resolve(path))
		return err == nil
	})

	return vm.Set("$file", obj)
}

// --- $yaml ---

func injectYaml(vm *goja.Runtime) error {
	obj := vm.NewObject()

	obj.Set("parse", func(s string) (interface{}, error) {
		var result interface{}
		if err := yaml.Unmarshal([]byte(s), &result); err != nil {
			return nil, fmt.Errorf("$yaml.parse: %w", err)
		}
		return normalizeYaml(result), nil
	})

	obj.Set("stringify", func(v interface{}) (string, error) {
		b, err := yaml.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("$yaml.stringify: %w", err)
		}
		return string(b), nil
	})

	return vm.Set("$yaml", obj)
}

// normalizeYaml converts yaml.Unmarshal output (map[string]interface{} with
// possible map[interface{}]interface{}) into JSON-compatible types that Goja can handle.
func normalizeYaml(v interface{}) interface{} {
	switch val := v.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(val))
		for k, v := range val {
			m[fmt.Sprintf("%v", k)] = normalizeYaml(v)
		}
		return m
	case map[string]interface{}:
		for k, v := range val {
			val[k] = normalizeYaml(v)
		}
		return val
	case []interface{}:
		for i, v := range val {
			val[i] = normalizeYaml(v)
		}
		return val
	default:
		return v
	}
}

// --- $http ---

func injectHttp(vm *goja.Runtime) error {
	client := &http.Client{Timeout: 10 * time.Second}

	obj := vm.NewObject()

	doRequest := func(method, url string, body io.Reader, options map[string]interface{}) (map[string]interface{}, error) {
		req, err := http.NewRequest(method, url, body)
		if err != nil {
			return nil, fmt.Errorf("$http: %w", err)
		}

		if headers, ok := options["headers"].(map[string]interface{}); ok {
			for k, v := range headers {
				if s, ok := v.(string); ok {
					req.Header.Set(k, s)
				}
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("$http: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("$http: reading response: %w", err)
		}

		respHeaders := make(map[string]interface{})
		for k, v := range resp.Header {
			if len(v) == 1 {
				respHeaders[k] = v[0]
			} else {
				respHeaders[k] = strings.Join(v, ", ")
			}
		}

		return map[string]interface{}{
			"status":  resp.StatusCode,
			"body":    string(respBody),
			"headers": respHeaders,
		}, nil
	}

	// $http.get(url, options?)
	obj.Set("get", func(url string, args ...map[string]interface{}) (map[string]interface{}, error) {
		opts := map[string]interface{}{}
		if len(args) > 0 {
			opts = args[0]
		}
		return doRequest("GET", url, nil, opts)
	})

	// $http.getText(url, options?)
	obj.Set("getText", func(url string, args ...map[string]interface{}) (string, error) {
		opts := map[string]interface{}{}
		if len(args) > 0 {
			opts = args[0]
		}
		resp, err := doRequest("GET", url, nil, opts)
		if err != nil {
			return "", err
		}
		return resp["body"].(string), nil
	})

	// $http.getJSON(url, options?)
	obj.Set("getJSON", func(url string, args ...map[string]interface{}) (interface{}, error) {
		opts := map[string]interface{}{}
		if len(args) > 0 {
			opts = args[0]
		}
		resp, err := doRequest("GET", url, nil, opts)
		if err != nil {
			return nil, err
		}
		var parsed interface{}
		if err := json.Unmarshal([]byte(resp["body"].(string)), &parsed); err != nil {
			return nil, fmt.Errorf("$http.getJSON: %w", err)
		}
		return parsed, nil
	})

	// $http.post(url, body, options?)
	obj.Set("post", func(url string, bodyStr string, args ...map[string]interface{}) (map[string]interface{}, error) {
		opts := map[string]interface{}{}
		if len(args) > 0 {
			opts = args[0]
		}
		return doRequest("POST", url, strings.NewReader(bodyStr), opts)
	})

	// $http.postJSON(url, body, options?)
	obj.Set("postJSON", func(url string, body interface{}, args ...map[string]interface{}) (interface{}, error) {
		opts := map[string]interface{}{}
		if len(args) > 0 {
			opts = args[0]
		}

		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("$http.postJSON: marshaling body: %w", err)
		}

		if _, ok := opts["headers"]; !ok {
			opts["headers"] = map[string]interface{}{}
		}
		opts["headers"].(map[string]interface{})["Content-Type"] = "application/json"

		resp, err := doRequest("POST", url, strings.NewReader(string(jsonBytes)), opts)
		if err != nil {
			return nil, err
		}

		var parsed interface{}
		if err := json.Unmarshal([]byte(resp["body"].(string)), &parsed); err != nil {
			return nil, fmt.Errorf("$http.postJSON: %w", err)
		}
		return parsed, nil
	})

	return vm.Set("$http", obj)
}

// --- $cluster ---

// kubectl runs a kubectl command and returns stdout. If kubectl is not available
// or the command fails, it returns an empty string and the error.
func kubectl(args ...string) (string, error) {
	cmd := exec.Command("kubectl", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("$cluster: kubectl %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output)), nil
}

func injectCluster(vm *goja.Runtime) error {
	obj := vm.NewObject()

	// $cluster.version() → "1.31.2"
	obj.Set("version", func() (string, error) {
		out, err := kubectl("version", "--output=json")
		if err != nil {
			return "", err
		}
		var v map[string]interface{}
		if err := json.Unmarshal([]byte(out), &v); err != nil {
			return "", fmt.Errorf("$cluster.version: %w", err)
		}
		if sv, ok := v["serverVersion"].(map[string]interface{}); ok {
			major, _ := sv["major"].(string)
			minor, _ := sv["minor"].(string)
			return major + "." + minor, nil
		}
		return "", fmt.Errorf("$cluster.version: cannot parse server version")
	})

	// $cluster.versionAtLeast("1.25") → true/false
	obj.Set("versionAtLeast", func(minVersion string) (bool, error) {
		out, err := kubectl("version", "--output=json")
		if err != nil {
			return false, err
		}
		var v map[string]interface{}
		if err := json.Unmarshal([]byte(out), &v); err != nil {
			return false, err
		}
		sv, ok := v["serverVersion"].(map[string]interface{})
		if !ok {
			return false, fmt.Errorf("$cluster.versionAtLeast: cannot parse server version")
		}

		majorStr, _ := sv["major"].(string)
		minorStr, _ := sv["minor"].(string)
		// minor may contain "+" suffix like "31+"
		minorStr = strings.TrimRight(minorStr, "+")

		major, _ := strconv.Atoi(majorStr)
		minor, _ := strconv.Atoi(minorStr)

		parts := strings.SplitN(minVersion, ".", 2)
		reqMajor, _ := strconv.Atoi(parts[0])
		reqMinor := 0
		if len(parts) > 1 {
			reqMinor, _ = strconv.Atoi(parts[1])
		}

		if major > reqMajor {
			return true, nil
		}
		if major == reqMajor && minor >= reqMinor {
			return true, nil
		}
		return false, nil
	})

	// $cluster.nodeCount() → number
	obj.Set("nodeCount", func() (int, error) {
		out, err := kubectl("get", "nodes", "--no-headers", "-o", "name")
		if err != nil {
			return 0, err
		}
		if out == "" {
			return 0, nil
		}
		return len(strings.Split(out, "\n")), nil
	})

	// $cluster.apiAvailable("gateway.networking.k8s.io/v1") → true/false
	obj.Set("apiAvailable", func(apiVersion string) bool {
		_, err := kubectl("api-resources", "--api-group="+extractGroup(apiVersion), "--no-headers")
		return err == nil
	})

	// $cluster.crdExists("certificates.cert-manager.io") → true/false
	obj.Set("crdExists", func(name string) bool {
		_, err := kubectl("get", "crd", name, "--no-headers")
		return err == nil
	})

	// $cluster.exists(apiVersion, kind, namespace, name) → true/false
	obj.Set("exists", func(apiVersion, kind, namespace, name string) bool {
		args := []string{"get", kind, name, "--no-headers"}
		if namespace != "" {
			args = append(args, "-n", namespace)
		}
		_, err := kubectl(args...)
		return err == nil
	})

	// $cluster.list(apiVersion, kind, namespace?) → []object
	obj.Set("list", func(kind string, args ...string) (interface{}, error) {
		cmdArgs := []string{"get", kind, "-o", "json"}
		if len(args) > 0 && args[0] != "" {
			cmdArgs = append(cmdArgs, "-n", args[0])
		}
		out, err := kubectl(cmdArgs...)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			return nil, fmt.Errorf("$cluster.list: %w", err)
		}
		items, _ := result["items"].([]interface{})
		return items, nil
	})

	return vm.Set("$cluster", obj)
}

// extractGroup extracts the API group from an apiVersion string.
// "networking.k8s.io/v1" → "networking.k8s.io"
// "v1" → "" (core group)
func extractGroup(apiVersion string) string {
	parts := strings.SplitN(apiVersion, "/", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return ""
}
