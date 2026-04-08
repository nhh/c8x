package cmd

import (
	"fmt"
	"github.com/kubernetix/c8x/internal/k8s"
	"github.com/kubernetix/c8x/internal/ts"
	"github.com/spf13/cobra"
	"os"
)

var (
	Verbose      bool
	AllowFile    bool
	AllowHttp    bool
	AllowCluster bool
	AllowAll     bool
)

func init() {
	install.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Enable debug output")
	install.PersistentFlags().BoolVar(&AllowFile, "allow-file", false, "Allow $file access (read files from chart directory)")
	install.PersistentFlags().BoolVar(&AllowHttp, "allow-http", false, "Allow $http access (make HTTP requests)")
	install.PersistentFlags().BoolVar(&AllowCluster, "allow-cluster", false, "Allow $cluster access (query Kubernetes API)")
	install.PersistentFlags().BoolVarP(&AllowAll, "allow-all", "A", false, "Allow all permissions")
	rootCmd.AddCommand(install)
}

func buildPermissions() ts.Permissions {
	if AllowAll {
		return ts.AllPermissions()
	}
	return ts.Permissions{
		File:    AllowFile,
		Http:    AllowHttp,
		Cluster: AllowCluster,
	}
}

func compileChart(path string, perms ts.Permissions) (k8s.Chart, error) {
	code, err := ts.Load(path, Verbose)
	if err != nil {
		return k8s.Chart{}, fmt.Errorf("loading chart: %w", err)
	}

	export, err := ts.Run(code, path, perms)
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

		perms := buildPermissions()

		chart, err := compileChart(args[0], perms)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client, err := k8s.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to cluster: %v\n", err)
			os.Exit(1)
		}

		if err := k8s.ApplyChart(client, chart); err != nil {
			fmt.Fprintf(os.Stderr, "Error applying chart: %v\n", err)
			os.Exit(1)
		}
	},
}
