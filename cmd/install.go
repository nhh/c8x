package cmd

import (
	"fmt"
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

func compileChart(path string) (k8s.Chart, error) {
	code, err := ts.Load(path, Verbose)
	if err != nil {
		return k8s.Chart{}, fmt.Errorf("loading chart: %w", err)
	}

	export, err := ts.Run(code, path)
	if err != nil {
		return k8s.Chart{}, fmt.Errorf("running chart: %w", err)
	}

	return k8s.PatchAndTransform(export), nil
}

var install = &cobra.Command{
	Use:   "install",
	Short: "Install a chart file into your k8s cluster",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			_ = cmd.Help()
			os.Exit(-1)
		}

		chart, err := compileChart(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := k8s.ApplyChart(chart); err != nil {
			fmt.Fprintf(os.Stderr, "Error applying chart: %v\n", err)
			os.Exit(1)
		}
	},
}
