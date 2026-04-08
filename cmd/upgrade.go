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
		if err := k8s.ApplyChart(client, chart); err != nil {
			fmt.Fprintf(os.Stderr, "Error applying chart: %v\n", err)
			os.Exit(1)
		}

		// Mark old revision as superseded
		if err := client.UpdateReleaseStatus(current, k8s.StatusSuperseded); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update old release status: %v\n", err)
		}

		// Save new revision
		newRevision := current.Revision + 1
		release := &k8s.Release{
			Name:       name,
			Revision:   newRevision,
			Status:     k8s.StatusDeployed,
			Namespace:  namespace,
			Manifest:   chart.Combined(),
			Env:        collectEnv(),
			DeployedAt: time.Now(),
		}

		if err := client.SaveRelease(release); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: chart applied but failed to save release state: %v\n", err)
		}

		// Garbage collect old revisions
		if err := client.DeleteOldRevisions(namespace, name, HistoryMax); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean old revisions: %v\n", err)
		}

		fmt.Printf("Upgraded %s to revision %d in namespace %s\n", name, newRevision, namespace)
	},
}
