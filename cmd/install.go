package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/kubernetix/c8x/internal/k8s"
	"github.com/kubernetix/c8x/internal/ts"
	"github.com/spf13/cobra"
)

var (
	Verbose      bool
	AllowFile    bool
	AllowHttp    bool
	AllowCluster bool
	AllowAll     bool
	ReleaseName  string
	HistoryMax   int
)

func init() {
	install.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Enable debug output")
	install.PersistentFlags().BoolVar(&AllowFile, "allow-file", false, "Allow $file access (read files from chart directory)")
	install.PersistentFlags().BoolVar(&AllowHttp, "allow-http", false, "Allow $http access (make HTTP requests)")
	install.PersistentFlags().BoolVar(&AllowCluster, "allow-cluster", false, "Allow $cluster access (query Kubernetes API)")
	install.PersistentFlags().BoolVarP(&AllowAll, "allow-all", "A", false, "Allow all permissions")
	install.PersistentFlags().StringVar(&ReleaseName, "name", "", "Release name (defaults to namespace)")
	install.PersistentFlags().IntVar(&HistoryMax, "history-max", 10, "Maximum number of revisions to keep")
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

func compileChart(path string, perms ts.Permissions) (k8s.Chart, k8s.ChartExport, error) {
	code, err := ts.Load(path, Verbose)
	if err != nil {
		return k8s.Chart{}, k8s.ChartExport{}, fmt.Errorf("loading chart: %w", err)
	}

	export, err := ts.Run(code, path, perms)
	if err != nil {
		return k8s.Chart{}, k8s.ChartExport{}, fmt.Errorf("running chart: %w", err)
	}

	return k8s.PatchAndTransform(export), export, nil
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

		chart, export, err := compileChart(args[0], perms)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		namespace := export.NamespaceName()
		if namespace == "" {
			namespace = "default"
		}

		name := ReleaseName
		if name == "" {
			name = namespace
		}

		client, err := k8s.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to cluster: %v\n", err)
			os.Exit(1)
		}

		// Check if already installed
		existing, err := client.GetCurrentRelease(namespace, name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking existing release: %v\n", err)
			os.Exit(1)
		}
		if existing != nil {
			fmt.Fprintf(os.Stderr, "Error: release %q already installed (revision %d). Use 'c8x upgrade' instead.\n", name, existing.Revision)
			os.Exit(1)
		}

		// Apply
		start := time.Now()
		if err := k8s.ApplyChart(client, chart); err != nil {
			fmt.Fprintf(os.Stderr, "Error applying chart: %v\n", err)
			os.Exit(1)
		}
		duration := time.Since(start)

		// Collect metadata
		manifest := chart.Combined()
		resources := k8s.ExtractResources(manifest)
		permsUsed := buildPermissions()

		// Save release state
		release := &k8s.Release{
			Name:          name,
			Revision:      1,
			Status:        k8s.StatusDeployed,
			Namespace:     namespace,
			Manifest:      manifest,
			DeployedAt:    time.Now(),
			Permissions:   &k8s.ReleasePermissions{File: permsUsed.File, Http: permsUsed.Http, Cluster: permsUsed.Cluster},
			Resources:     resources,
			ResourceCount: len(resources),
			Duration:      duration.Round(time.Millisecond).String(),
			Trigger:       k8s.TriggerManual,
			Source:        k8s.CollectSource(args[0]),
			Runtime:       k8s.CollectRuntime(),
			Deployer:      k8s.CollectDeployer(),
			CI:            k8s.DetectCI(),
		}
		if release.CI != nil {
			release.Trigger = k8s.TriggerCI
		}

		if err := client.SaveRelease(release); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: chart applied but failed to save release state: %v\n", err)
		}

		fmt.Printf("Installed %s (revision 1) in namespace %s [%s, %d resources]\n", name, namespace, release.Duration, release.ResourceCount)
	},
}
