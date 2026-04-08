package ts

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dop251/goja"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/kubernetix/c8x/internal/k8s"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Loads and transpiles tsx files
func Load(path string, debug bool) (string, error) {
	options := api.BuildOptions{
		Loader: map[string]api.Loader{
			".ts": api.LoaderTS,
			".js": api.LoaderJS,
		},
		EntryPoints: []string{path},
		Bundle:      true,
		Write:       false,
		GlobalName:  "c8x",
		Format:      api.FormatIIFE,
	}

	result := api.Build(options)

	if len(result.Errors) > 0 {
		msgs := make([]string, len(result.Errors))
		for i, msg := range result.Errors {
			msgs[i] = msg.Text
		}
		return "", fmt.Errorf("esbuild: %s", strings.Join(msgs, "; "))
	}

	for _, message := range result.Warnings {
		fmt.Println(message)
	}

	if len(result.OutputFiles) == 0 {
		return "", fmt.Errorf("esbuild produced no output for %s", path)
	}

	code := string(result.OutputFiles[0].Contents)

	if debug {
		fmt.Print(code)
	}

	return code, nil
}

// Can return number or string
func __jsEnvGet(name string) interface{} {
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)

		if !strings.HasPrefix(pair[0], "C8X_") {
			continue
		}

		if name != strings.Replace(pair[0], "C8X_", "", 1) {
			continue
		}

		if pair[1] == "true" {
			return true
		}

		if pair[1] == "false" {
			return false
		}

		i, err := strconv.Atoi(strings.TrimSpace(pair[1]))
		if err != nil {
			return strings.TrimSpace(pair[1])
		}
		return i
	}

	return nil
}

// Can return number or string
func __jsEnvGetAsObject(name string) interface{} {
	m := make(map[string]interface{})

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)

		if !strings.HasPrefix(pair[0], "C8X_") {
			continue
		}

		if !strings.HasPrefix(strings.Replace(pair[0], "C8X_", "", 1), name) {
			continue
		}

		if !strings.Contains(pair[0], "KEY") {
			continue
		}

		key := os.Getenv(pair[0])
		value := strings.TrimSpace(os.Getenv(strings.Replace(pair[0], "KEY", "VALUE", 1)))

		i, err := strconv.Atoi(value)
		if err != nil {
			if value == "true" {
				m[key] = true
				continue
			}

			if value == "false" {
				m[key] = false
				continue
			}

			m[key] = value
		} else {
			m[key] = i
		}
	}

	return m
}

func injectEnv(vm *goja.Runtime) error {
	obj := vm.NewObject()

	if err := obj.ToObject(vm).Set("get", __jsEnvGet); err != nil {
		return fmt.Errorf("injecting $env.get: %w", err)
	}

	if err := obj.ToObject(vm).Set("getAsObject", __jsEnvGetAsObject); err != nil {
		return fmt.Errorf("injecting $env.getAsObject: %w", err)
	}

	if err := vm.Set("$env", obj); err != nil {
		return fmt.Errorf("setting $env on vm: %w", err)
	}

	return nil
}

func injectChartInfo(vm *goja.Runtime, path string) error {
	dir, _ := filepath.Split(path)
	packageJson := filepath.Join(dir, "package.json")

	if _, err := os.Stat(packageJson); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	fileOutput, err := os.ReadFile(packageJson)
	if err != nil {
		return fmt.Errorf("reading package.json: %w", err)
	}

	var pjson any
	if err := json.Unmarshal(fileOutput, &pjson); err != nil {
		return fmt.Errorf("parsing package.json: %w", err)
	}

	if err := vm.Set("$chart", pjson); err != nil {
		return fmt.Errorf("setting $chart on vm: %w", err)
	}

	return nil
}

// parseChartExport converts the raw Goja output into a typed ChartExport.
func parseChartExport(raw map[string]interface{}) (k8s.ChartExport, error) {
	export := k8s.ChartExport{}

	// Parse namespace (optional)
	if ns, ok := raw["namespace"].(map[string]interface{}); ok {
		export.Namespace = k8s.K8sResource(ns)
	}

	// Parse components
	rawComponents, ok := raw["components"].([]interface{})
	if !ok && raw["components"] != nil {
		return export, fmt.Errorf("components must be an array")
	}

	export.Components = make([]k8s.K8sResource, 0, len(rawComponents))
	for _, c := range rawComponents {
		if c == nil {
			export.Components = append(export.Components, nil)
			continue
		}
		comp, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		export.Components = append(export.Components, k8s.K8sResource(comp))
	}

	return export, nil
}

// RunOptions configures the chart execution environment.
type RunOptions struct {
	Permissions Permissions
	Namespace   string // for $release lookup (empty = skip $release)
	ReleaseName string // for $release lookup (empty = skip $release)
}

// Executes tsx and returns its result
func Run(code string, path string, perms ...Permissions) (k8s.ChartExport, error) {
	opts := RunOptions{}
	if len(perms) > 0 {
		opts.Permissions = perms[0]
	}

	vm := goja.New()

	if err := injectEnv(vm); err != nil {
		return k8s.ChartExport{}, err
	}

	if err := injectChartInfo(vm, path); err != nil {
		return k8s.ChartExport{}, err
	}

	chartDir, _ := filepath.Split(path)

	if err := injectBase64(vm); err != nil {
		return k8s.ChartExport{}, err
	}

	if err := injectHash(vm); err != nil {
		return k8s.ChartExport{}, err
	}

	if err := injectLog(vm); err != nil {
		return k8s.ChartExport{}, err
	}

	if err := injectAssert(vm); err != nil {
		return k8s.ChartExport{}, err
	}

	if err := injectFile(vm, chartDir, opts.Permissions); err != nil {
		return k8s.ChartExport{}, err
	}

	if err := injectYaml(vm); err != nil {
		return k8s.ChartExport{}, err
	}

	if err := injectHttp(vm, opts.Permissions); err != nil {
		return k8s.ChartExport{}, err
	}

	if err := injectCluster(vm, opts.Permissions); err != nil {
		return k8s.ChartExport{}, err
	}

	if opts.Namespace != "" && opts.ReleaseName != "" {
		if err := injectRelease(vm, opts.Permissions, opts.Namespace, opts.ReleaseName); err != nil {
			return k8s.ChartExport{}, err
		}
	}

	if _, err := vm.RunString(code); err != nil {
		return k8s.ChartExport{}, fmt.Errorf("executing chart code: %w", err)
	}

	c8x, ok := goja.AssertFunction(vm.Get("c8x").ToObject(vm).Get("default"))
	if !ok {
		return k8s.ChartExport{}, fmt.Errorf("chart must export a default function: export default () => ({})")
	}

	obj, err := c8x(goja.Undefined())
	if err != nil {
		return k8s.ChartExport{}, fmt.Errorf("calling chart default function: %w", err)
	}

	raw, ok := obj.Export().(map[string]interface{})
	if !ok {
		return k8s.ChartExport{}, fmt.Errorf("chart default function must return an object")
	}

	export, err := parseChartExport(raw)
	if err != nil {
		return k8s.ChartExport{}, err
	}

	// Patch namespace into components
	nsName := export.NamespaceName()
	if nsName != "" {
		for _, component := range export.Components {
			if component == nil {
				continue
			}
			component.SetNamespace(nsName)
		}
	}

	return export, nil
}
