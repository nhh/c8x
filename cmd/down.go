package cmd

import (
	"fmt"
	"os"

	"github.com/kubernetix/c8x/internal/k8s"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(downCmd)
}

var downCmd = &cobra.Command{
	Use:   "down <name>",
	Short: "Remove a release and all its resources",
	Long: `Remove a deployed release and delete all its Kubernetes resources.
Like "docker compose down" - tears everything down cleanly.`,
	Example: "c8x down wordpress",
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

		current, err := client.GetCurrentRelease(namespace, name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if current == nil {
			fmt.Fprintf(os.Stderr, "Error: release %q not found\n", name)
			os.Exit(1)
		}

		// Delete all resources from the manifest
		output, err := client.Delete([]byte(current.Manifest))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting resources: %v\n", err)
			os.Exit(1)
		}
		if output != "" {
			fmt.Println(output)
		}

		// Delete all release ConfigMaps
		if err := client.DeleteReleases(namespace, name); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to delete release history: %v\n", err)
		}

		fmt.Printf("Removed %s from namespace %s\n", name, namespace)
	},
}
