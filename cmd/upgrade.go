package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/kubernetix/c8x/internal/k8s"
	"github.com/spf13/cobra"
)

func init() {
	upgradeCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Enable debug output")
	upgradeCmd.PersistentFlags().BoolVar(&AllowFile, "allow-file", false, "Allow $file access")
	upgradeCmd.PersistentFlags().BoolVar(&AllowHttp, "allow-http", false, "Allow $http access")
	upgradeCmd.PersistentFlags().BoolVar(&AllowCluster, "allow-cluster", false, "Allow $cluster access")
	upgradeCmd.PersistentFlags().BoolVarP(&AllowAll, "allow-all", "A", false, "Allow all permissions")
	upgradeCmd.PersistentFlags().StringVar(&ReleaseName, "name", "", "Release name (defaults to namespace)")
	upgradeCmd.PersistentFlags().IntVar(&HistoryMax, "history-max", 10, "Maximum number of revisions to keep")
	rootCmd.AddCommand(upgradeCmd)
}

var upgradeCmd = &cobra.Command{
	Use:     "upgrade <file>",
	Short:   "Upgrade an existing release",
	Example: "c8x upgrade chart.ts",
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
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Get current release
		current, err := client.GetCurrentRelease(namespace, name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if current == nil {
			fmt.Fprintf(os.Stderr, "Error: release %q not found. Use 'c8x install' first.\n", name)
			os.Exit(1)
		}

		// Apply
		start := time.Now()
		if err := k8s.ApplyChart(client, chart); err != nil {
			fmt.Fprintf(os.Stderr, "Error applying chart: %v\n", err)
			os.Exit(1)
		}
		duration := time.Since(start)

		// Mark old revision as superseded
		if err := client.UpdateReleaseStatus(current, k8s.StatusSuperseded); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update old release status: %v\n", err)
		}

		// Collect metadata
		manifest := chart.Combined()
		resources := k8s.ExtractResources(manifest)
		permsUsed := buildPermissions()

		newRevision := current.Revision + 1
		release := &k8s.Release{
			Name:          name,
			Revision:      newRevision,
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

		if err := client.DeleteOldRevisions(namespace, name, HistoryMax); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean old revisions: %v\n", err)
		}

		fmt.Printf("Upgraded %s to revision %d in namespace %s [%s, %d resources]\n", name, newRevision, namespace, release.Duration, release.ResourceCount)
	},
}
