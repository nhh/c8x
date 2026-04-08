package cmd

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path"
	"text/template"
)

func init() {
	rootCmd.AddCommand(newCmd)
}

var chartTsx = `import {Chart} from "c8x"

// TODO: fix the compiler issues to have a working deployment
export default (): Chart => {}
`

var packageJson = `{
  "name": "{{.packageName}}",
  "private": true,
  "version": "0.0.1",
  "dependencies": {
    "c8x": "0.0.17"
  },
  "appVersion": "1.0.0",
  "kubeVersion": "1.31",
  "type": "application",
  "keywords": [
    "cms",
    "wordpress",
    "author"
  ],
  "home": "https://github.com/kubernetix/charts/wordpress",
  "maintainers": [
    "Niklas Hanft"
  ],
  "icon": null,
  "deprecated": false,
  "annotations": []
}
`

func renderPackageJson(name string) ([]byte, error) {
	t, err := template.New("text").Parse(packageJson)
	if err != nil {
		return nil, fmt.Errorf("parsing package.json template: %w", err)
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, map[string]interface{}{"packageName": name})
	if err != nil {
		return nil, fmt.Errorf("executing package.json template: %w", err)
	}

	return buf.Bytes(), nil
}

func initChart(dir string, name string) error {
	pkgJson, err := renderPackageJson(name)
	if err != nil {
		return err
	}

	chartPath := path.Join(dir, name)

	err = os.Mkdir(chartPath, 0777)
	if err != nil {
		return fmt.Errorf("creating chart directory: %w", err)
	}

	err = os.WriteFile(path.Join(chartPath, "package.json"), pkgJson, 0666)
	if err != nil {
		return fmt.Errorf("writing package.json: %w", err)
	}

	err = os.WriteFile(path.Join(chartPath, "chart.ts"), []byte(chartTsx), 0666)
	if err != nil {
		return fmt.Errorf("writing chart.ts: %w", err)
	}

	return nil
}

var newCmd = &cobra.Command{
	Use:     "init",
	Short:   "Initialize a c8x chart. (index.ts, package.json)",
	Example: "c8x init wordpress",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			os.Exit(-1)
		}

		chartName := args[0]

		err := initChart(".", chartName)
		if err != nil {
			panic(err)
		}

		fmt.Println("Initialized chart....")
	},
}
