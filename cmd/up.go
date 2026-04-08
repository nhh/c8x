package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/kubernetix/c8x/internal/k8s"
	"github.com/spf13/cobra"
)

func init() {
	upCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Enable debug output")
	upCmd.PersistentFlags().BoolVar(&AllowFile, "allow-file", false, "Allow $file access")
	upCmd.PersistentFlags().BoolVar(&AllowHttp, "allow-http", false, "Allow $http access")
	upCmd.PersistentFlags().BoolVar(&AllowCluster, "allow-cluster", false, "Allow $cluster access")
	upCmd.PersistentFlags().BoolVarP(&AllowAll, "allow-all", "A", false, "Allow all permissions")
	upCmd.PersistentFlags().StringVar(&ReleaseName, "name", "", "Release name (defaults to namespace)")
	upCmd.PersistentFlags().IntVar(&HistoryMax, "history-max", 10, "Maximum number of revisions to keep")
	rootCmd.AddCommand(upCmd)
}

var upCmd = &cobra.Command{
	Use:   "up <file>",
	Short: "Deploy a chart (install or upgrade automatically)",
	Long: `Deploy a chart to your cluster. If no release exists, it creates one.
If a release already exists, it upgrades to the new version.
Like "docker compose up" - just run it and it does the right thing.`,
	Example: "c8x up chart.ts",
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
		current, err := client.GetCurrentRelease(namespace, name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking release: %v\n", err)
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

		if current == nil {
			// Install (new release)
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
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}

			fmt.Printf("Installed %s (revision 1) in namespace %s [%s, %d resources]\n", name, namespace, release.Duration, release.ResourceCount)
		} else {
			// Upgrade (existing release)
			if err := client.UpdateReleaseStatus(current, k8s.StatusSuperseded); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}

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
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}

			if err := client.DeleteOldRevisions(namespace, name, HistoryMax); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}

			fmt.Printf("Upgraded %s to revision %d in namespace %s [%s, %d resources]\n", name, newRevision, namespace, release.Duration, release.ResourceCount)
		}
	},
}
