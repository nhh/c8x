package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"runtime/debug"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version control information",
	Run: func(cmd *cobra.Command, args []string) {
		buildInfo, _ := debug.ReadBuildInfo()
		fmt.Println(buildInfo.String())
	},
}
