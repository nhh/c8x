package ts

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dop251/goja"
	"github.com/evanw/esbuild/pkg/api"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Loads and transpiles tsx files
func Load(path string, debug bool) string {

	options := api.BuildOptions{
		Loader: map[string]api.Loader{
			".ts": api.LoaderTS,
			".js": api.LoaderJS,
		},
		EntryPoints: []string{path},
		Bundle:      true,
		Write:       false,
		GlobalName:  "k8x",
		Format:      api.FormatIIFE,
	}

	result := api.Build(options)

	for _, message := range result.Errors {
		fmt.Println(message)
	}

	for _, message := range result.Warnings {
		fmt.Println(message)
	}

	if len(result.Errors) > 0 {
		fmt.Println("Cannot transform js")
	}

	if debug {
		fmt.Print(string(result.OutputFiles[0].Contents))
	}

	return string(result.OutputFiles[0].Contents)
}

// Can return number or string
func __jsEnvGet(name string) interface{} {
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")

		if strings.Index(pair[0], "K8X_") == -1 {
			continue
		}

		if name != strings.Replace(pair[0], "K8X_", "", 1) {
			continue
		}

		if "true" == pair[1] {
			return true
		}

		if "false" == pair[1] {
			return false
		}

		// Try to parse stuff as number, might break stuff
		// Dont know if https://1231412.de gets converted to 1231412
		// Allows ts to write k8x.$env["SCALE"] instead of having to parse it: Number(k8x.$env["SCALE"])
		i, err := strconv.Atoi(strings.TrimSpace(pair[1]))
		if err != nil {
			return strings.TrimSpace(pair[1])
		} else {
			return i
		}
	}

	return nil
}

// Can return number or string
func __jsEnvGetAsObject(name string) interface{} {
	m := make(map[string]interface{})

	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		if strings.Index(pair[0], "K8X_") == -1 {
			continue
		}

		if !strings.HasPrefix(strings.Replace(pair[0], "K8X_", "", 1), name) {
			continue
		}

		if !strings.Contains(pair[0], "KEY") {
			continue
		}

		// K8X_INGRESS_CLASS_ANNOTATIONS_KEY_1=nginx.ingress.kubernetes.io/app-root
		// K8X_INGRESS_CLASS_ANNOTATIONS_VALUE_1=/var/www/html

		// K8X_INGRESS_CLASS_ANNOTATIONS_KEY_2=nginx.ingress.kubernetes.io/enable-cors
		// K8X_INGRESS_CLASS_ANNOTATIONS_VALUE_2=true
		key := os.Getenv(pair[0])
		value := strings.TrimSpace(os.Getenv(strings.Replace(pair[0], "KEY", "VALUE", 1)))

		// Try to parse stuff as number, might break stuff
		// Dont know if https://1231412.de gets converted to 1231412
		// Allows ts to write k8x.$env["SCALE"] instead of having to parse it: Number(k8x.$env["SCALE"])
		i, err := strconv.Atoi(value)
		if err != nil {
			if "true" == value {
				m[key] = true
				continue
			}

			if "false" == value {
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

func injectEnv(vm *goja.Runtime) {
	obj := vm.NewObject()

	err := obj.ToObject(vm).Set("get", __jsEnvGet)
	if err != nil {
		panic("Cannot inject $env.get")
	}

	err = obj.ToObject(vm).Set("getAsObject", __jsEnvGetAsObject)
	if err != nil {
		panic("Cannot inject $env.getAsObject")
	}

	err = vm.Set("$env", obj)

	if err != nil {
		panic("Cannot set $env obj to vm")
	}

}

func injectChartInfo(vm *goja.Runtime, path string) {
	dir, _ := filepath.Split(path)

	packageJson := filepath.Join(dir, "./package.json")

	if _, err := os.Stat(packageJson); errors.Is(err, os.ErrNotExist) {
		fmt.Println("INFO: No package.json detected, ignoring chart information")
		return
	}

	fileOutput, _ := os.ReadFile(packageJson)

	var pjson any
	_ = json.Unmarshal(fileOutput, &pjson)

	err := vm.Set("$chart", pjson)

	if err != nil {
		panic("Cannot set $env obj to vm")
	}
}

// Executes tsx and returns its result
func Run(code string, path string) map[string]interface{} {
	vm := goja.New()

	// Todo handle this better because one creates k8x the other expects it to exist
	injectEnv(vm)
	injectChartInfo(vm, path)

	_, err := vm.RunString(code)

	if err != nil {
		panic(err)
	}

	k8x, ok := goja.AssertFunction(vm.Get("k8x").ToObject(vm).Get("default"))

	if !ok {
		panic("Please make sure you are exporting a function: export default () => ({})")
	}

	obj, err := k8x(goja.Undefined())

	if err != nil {
		panic(err)
	}

	k8sExport, ok := obj.Export().(map[string]interface{})

	if !ok {
		panic("Cant cast to object")
	}

	// Well thats actually wild
	name, ok := k8sExport["namespace"].(map[string]interface{})["metadata"].(map[string]interface{})["name"]

	// Patching namespaces
	if name != nil && ok {
		for _, component := range k8sExport["components"].([]interface{}) {
			if component == nil {
				continue
			}

			comp, _ := component.(map[string]interface{})

			if comp == nil {
				comp = make(map[string]interface{})
			}

			metadata, _ := comp["metadata"].(map[string]interface{})

			if metadata == nil {
				m := make(map[string]interface{})
				comp["metadata"] = m
				continue
			}

			metadata["namespace"] = name
		}
	}

	return k8sExport
}
