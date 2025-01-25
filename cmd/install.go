package cmd

import (
	"github.com/kubernetix/c8x/internal/k8s"
	"github.com/kubernetix/c8x/internal/ts"
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
	Short: "Install a chart file into your k8s cluster",
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
