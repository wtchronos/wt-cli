package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print wt version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("wt %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
		fmt.Printf("  os:     %s\n", runtime.GOOS)
		fmt.Printf("  arch:   %s\n", runtime.GOARCH)
		fmt.Printf("  go:     %s\n", runtime.Version())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
