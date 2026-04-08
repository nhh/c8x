package cmd

import (
	"fmt"
	"os"

	"github.com/kubernetix/c8x/internal/k8s"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(historyCmd)
}

var historyCmd = &cobra.Command{
	Use:     "history <name>",
	Short:   "Show release history",
	Example: "c8x history wordpress",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			_ = cmd.Help()
			os.Exit(-1)
		}

		name := args[0]
		namespace := name
		if ReleaseName != "" {
			namespace = ReleaseName
		}

		client, err := k8s.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		releases, err := client.ListReleases(namespace, name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(releases) == 0 {
			fmt.Fprintf(os.Stderr, "No releases found for %q in namespace %q\n", name, namespace)
			os.Exit(1)
		}

		fmt.Printf("%-10s %-12s %-25s %s\n", "REVISION", "STATUS", "DEPLOYED", "CHART")
		for _, r := range releases {
			fmt.Printf("%-10d %-12s %-25s %s@%s\n",
				r.Revision,
				r.Status,
				r.DeployedAt.Format("2006-01-02 15:04:05"),
				r.ChartName,
				r.ChartVersion,
			)
		}
	},
}
