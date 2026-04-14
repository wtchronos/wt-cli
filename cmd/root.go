package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	verbose bool
	configPath string
)

var rootCmd = &cobra.Command{
	Use:   "wt",
	Short: "Warren's personal CLI tool for project automation",
	Long:  `wt provides git hooks, shell integration, and project-specific automation`,
}

func Execute() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "path to config file (default looks for .wt.toml in ancestors)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
