package cmd

import (
	"fmt"
	"os"

	"github.com/kubernetix/c8x/internal/k8s"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:     "status <name>",
	Short:   "Show current release status",
	Example: "c8x status wordpress",
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

		release, err := client.GetCurrentRelease(namespace, name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if release == nil {
			fmt.Fprintf(os.Stderr, "No active release found for %q\n", name)
			os.Exit(1)
		}

		fmt.Printf("Name:       %s\n", release.Name)
		fmt.Printf("Namespace:  %s\n", release.Namespace)
		fmt.Printf("Revision:   %d\n", release.Revision)
		fmt.Printf("Status:     %s\n", release.Status)
		fmt.Printf("Trigger:    %s\n", release.Trigger)
		fmt.Printf("Deployed:   %s\n", release.DeployedAt.Format("2006-01-02 15:04:05"))
		if release.Duration != "" {
			fmt.Printf("Duration:   %s\n", release.Duration)
		}
		fmt.Printf("Resources:  %d\n", release.ResourceCount)
		if len(release.Resources) > 0 {
			for _, r := range release.Resources {
				fmt.Printf("            - %s\n", r)
			}
		}
		if release.Source != nil {
			fmt.Printf("Source:     %s\n", release.Source.File)
			if release.Source.Checksum != "" {
				fmt.Printf("Checksum:   %s\n", release.Source.Checksum)
			}
		}
		if release.Deployer != nil {
			fmt.Printf("Deployer:   %s@%s\n", release.Deployer.User, release.Deployer.Hostname)
		}
		if release.Runtime != nil {
			fmt.Printf("Runtime:    c8x %s (%s/%s)\n", release.Runtime.C8xVersion, release.Runtime.OS, release.Runtime.Arch)
		}
		if release.CI != nil {
			fmt.Printf("CI:         %s (run %s by %s)\n", release.CI.Provider, release.CI.RunID, release.CI.Actor)
		}
		if release.PreviousRevision != nil {
			fmt.Printf("Rollback:   from revision %d\n", *release.PreviousRevision)
		}
	},
}
