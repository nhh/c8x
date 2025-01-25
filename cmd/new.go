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

var chartTsx = `
import {Chart} from "c8x"

// TODO: fix the compiler issues to have a working deployment
export default (): Chart => {}
`

var packageJson = `{
  "name": "{{.packageName}}",
  "private": true,
  "version": "0.0.1",
  "dependencies": {
    "c8x": "0.0.1"
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

var newCmd = &cobra.Command{
	Use:     "init",
	Short:   "Initialize a c8x chart. (index.ts, package.json)",
	Example: "c8x init wordpress",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			os.Exit(-1)
		}

		chartPath := args[0]

		t, _ := template.New("text").Parse(packageJson)

		pkgjson := bytes.Buffer{}
		err := t.Execute(&pkgjson, map[string]interface{}{"packageName": chartPath})

		if err != nil {
			panic(err)
		}

		err = os.Mkdir(chartPath, 0666)

		if err != nil {
			panic(err)
		}

		err = os.WriteFile(path.Join(chartPath, "package.json"), pkgjson.Bytes(), 0666)

		if err != nil {
			panic(err)
		}

		err = os.WriteFile(path.Join(chartPath, "chart.tsx"), []byte(chartTsx), 0666)

		if err != nil {
			panic(err)
		}

		fmt.Println("Initialized chart....")
	},
}
