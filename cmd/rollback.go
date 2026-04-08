package cmd

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kubernetix/c8x/internal/k8s"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(rollbackCmd)
}

var rollbackCmd = &cobra.Command{
	Use:     "rollback <name> [revision]",
	Short:   "Rollback to a previous revision",
	Example: "c8x rollback wordpress 2",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			_ = cmd.Help()
			os.Exit(-1)
		}

		name := args[0]
		namespace := name

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
			fmt.Fprintf(os.Stderr, "Error: release %q not found\n", name)
			os.Exit(1)
		}

		// Determine target revision
		targetRevision := current.Revision - 1
		if len(args) > 1 {
			targetRevision, err = strconv.Atoi(args[1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid revision %q\n", args[1])
				os.Exit(1)
			}
		}

		if targetRevision < 1 {
			fmt.Fprintf(os.Stderr, "Error: no previous revision to rollback to\n")
			os.Exit(1)
		}

		// Get target release
		target, err := client.GetRelease(namespace, name, targetRevision)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Apply the old manifest
		output, err := client.Apply([]byte(target.Manifest))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error applying rollback: %v\n", err)
			os.Exit(1)
		}
		if output != "" {
			fmt.Println(output)
		}

		// Mark current as superseded
		if err := client.UpdateReleaseStatus(current, k8s.StatusSuperseded); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}

		// Save new revision with the old manifest
		newRevision := current.Revision + 1
		prevRev := targetRevision
		release := &k8s.Release{
			Name:             name,
			Revision:         newRevision,
			Status:           k8s.StatusDeployed,
			ChartName:        target.ChartName,
			ChartVersion:     target.ChartVersion,
			Namespace:        namespace,
			Manifest:         target.Manifest,
			DeployedAt:       time.Now(),
			Resources:        k8s.ExtractResources(target.Manifest),
			ResourceCount:    len(k8s.ExtractResources(target.Manifest)),
			Trigger:          k8s.TriggerRollback,
			PreviousRevision: &prevRev,
			Runtime:          k8s.CollectRuntime(),
			Deployer:         k8s.CollectDeployer(),
			CI:               k8s.DetectCI(),
		}

		if err := client.SaveRelease(release); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}

		fmt.Printf("Rolled back %s to revision %d (new revision %d)\n", name, targetRevision, newRevision)
	},
}
