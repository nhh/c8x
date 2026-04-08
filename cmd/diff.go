package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/kubernetix/c8x/internal/k8s"
	"github.com/spf13/cobra"
)

func init() {
	diffCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Enable debug output")
	diffCmd.PersistentFlags().BoolVar(&AllowFile, "allow-file", false, "Allow $file access")
	diffCmd.PersistentFlags().BoolVar(&AllowHttp, "allow-http", false, "Allow $http access")
	diffCmd.PersistentFlags().BoolVar(&AllowCluster, "allow-cluster", false, "Allow $cluster access")
	diffCmd.PersistentFlags().BoolVarP(&AllowAll, "allow-all", "A", false, "Allow all permissions")
	diffCmd.PersistentFlags().StringVar(&ReleaseName, "name", "", "Release name")
	rootCmd.AddCommand(diffCmd)
}

var diffCmd = &cobra.Command{
	Use:     "diff <file>",
	Short:   "Show what would change on upgrade",
	Example: "c8x diff chart.ts",
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

		current, err := client.GetCurrentRelease(namespace, name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		newManifest := chart.Combined()

		if current == nil {
			fmt.Println("No existing release found. Showing full manifest:")
			fmt.Println(newManifest)
			return
		}

		oldLines := strings.Split(current.Manifest, "\n")
		newLines := strings.Split(newManifest, "\n")

		if current.Manifest == newManifest {
			fmt.Println("No changes detected.")
			return
		}

		// Simple line-based diff
		fmt.Printf("--- %s revision %d\n", name, current.Revision)
		fmt.Printf("+++ %s (pending)\n", name)

		oldSet := make(map[string]bool, len(oldLines))
		for _, l := range oldLines {
			oldSet[l] = true
		}
		newSet := make(map[string]bool, len(newLines))
		for _, l := range newLines {
			newSet[l] = true
		}

		for _, l := range oldLines {
			if !newSet[l] {
				fmt.Printf("- %s\n", l)
			}
		}
		for _, l := range newLines {
			if !oldSet[l] {
				fmt.Printf("+ %s\n", l)
			}
		}
	},
}
