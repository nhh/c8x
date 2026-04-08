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
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/kubernetix/c8x/internal/k8s"
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

func injectCluster(vm *goja.Runtime) error {
	var client *k8s.Client
	var clientErr error
	var once sync.Once

	getClient := func() (*k8s.Client, error) {
		once.Do(func() { client, clientErr = k8s.NewClient() })
		return client, clientErr
	}

	obj := vm.NewObject()

	// $cluster.version() → "1.31"
	obj.Set("version", func() (string, error) {
		c, err := getClient()
		if err != nil {
			return "", err
		}
		return c.ServerVersion()
	})

	// $cluster.versionAtLeast("1.25") → true/false
	obj.Set("versionAtLeast", func(minVersion string) (bool, error) {
		c, err := getClient()
		if err != nil {
			return false, err
		}
		version, err := c.ServerVersion()
		if err != nil {
			return false, err
		}
		return compareVersions(version, minVersion), nil
	})

	// $cluster.nodeCount() → number
	obj.Set("nodeCount", func() (int, error) {
		c, err := getClient()
		if err != nil {
			return 0, err
		}
		return c.NodeCount()
	})

	// $cluster.apiAvailable("gateway.networking.k8s.io/v1") → true/false
	obj.Set("apiAvailable", func(apiVersion string) bool {
		c, err := getClient()
		if err != nil {
			return false
		}
		return c.APIAvailable(apiVersion)
	})

	// $cluster.crdExists("certificates.cert-manager.io") → true/false
	obj.Set("crdExists", func(name string) bool {
		c, err := getClient()
		if err != nil {
			return false
		}
		return c.CRDExists(name)
	})

	// $cluster.exists(apiVersion, kind, namespace, name) → true/false
	obj.Set("exists", func(apiVersion, kind, namespace, name string) bool {
		c, err := getClient()
		if err != nil {
			return false
		}
		return c.ResourceExists(kind, namespace, name)
	})

	// $cluster.list(kind, namespace?) → []object
	obj.Set("list", func(kind string, args ...string) (interface{}, error) {
		c, err := getClient()
		if err != nil {
			return nil, err
		}
		ns := ""
		if len(args) > 0 {
			ns = args[0]
		}
		return c.ListResources(kind, ns)
	})

	return vm.Set("$cluster", obj)
}

// compareVersions returns true if actual >= required.
func compareVersions(actual, required string) bool {
	actualParts := strings.SplitN(actual, ".", 2)
	requiredParts := strings.SplitN(required, ".", 2)

	actualMajor, _ := strconv.Atoi(actualParts[0])
	actualMinor := 0
	if len(actualParts) > 1 {
		actualMinor, _ = strconv.Atoi(strings.TrimRight(actualParts[1], "+"))
	}

	reqMajor, _ := strconv.Atoi(requiredParts[0])
	reqMinor := 0
	if len(requiredParts) > 1 {
		reqMinor, _ = strconv.Atoi(requiredParts[1])
	}

	if actualMajor > reqMajor {
		return true
	}
	return actualMajor == reqMajor && actualMinor >= reqMinor
}
