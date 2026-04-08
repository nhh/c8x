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
		fmt.Printf("Chart:      %s@%s\n", release.ChartName, release.ChartVersion)
		fmt.Printf("Deployed:   %s\n", release.DeployedAt.Format("2006-01-02 15:04:05"))
	},
}
