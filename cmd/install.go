package cmd

import (
	"github.com/kubernetix/k8x/v1/internal/k8s"
	"github.com/kubernetix/k8x/v1/internal/ts"
	"github.com/spf13/cobra"
	"os"
)

var Verbose bool

func init() {
	install.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Enable debug output")
	rootCmd.AddCommand(install)
}

var install = &cobra.Command{
	Use:   "install",
	Short: "Install a chart.tsx file into your k8s cluster",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			_ = cmd.Help()
			os.Exit(-1)
		}

		path := args[0]

		code := ts.Load(path, Verbose)
		export := ts.Run(code, path)
		chart := k8s.PatchAndTransform(export)
		k8s.ApplyChart(chart)
	},
}
