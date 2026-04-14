package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wtchronos/wt-cli/internal/config"
)

var validAliasName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var aliasesCmd = &cobra.Command{
	Use:   "aliases",
	Short: "Manage project-specific aliases",
	Run: func(cmd *cobra.Command, args []string) {
		loadFlag, _ := cmd.Flags().GetBool("load")
		unloadFlag, _ := cmd.Flags().GetBool("unload")

		if !loadFlag && !unloadFlag {
			fmt.Println("Use --load or --unload")
			os.Exit(1)
		}

		configPath, err := config.Find(".")
		if err != nil {
			os.Exit(0) // No config, no aliases
			return
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if cfg.Aliases == nil || len(cfg.Aliases) == 0 {
			return
		}

		if unloadFlag {
			// Emit unalias statements
			for name := range cfg.Aliases {
				fmt.Printf("unalias %s\n", name)
			}
		} else if loadFlag {
			// First unload any existing, then load new
			for name := range cfg.Aliases {
				fmt.Printf("unalias %s 2>/dev/null; alias %s='%s'\n", 
					name, name, shellQuote(cfg.Aliases[name]))
			}
		}
	},
}

func shellQuote(s string) string {
	// Simple shell quoting - replace single quotes
	s = strings.ReplaceAll(s, "'", "'\\''")
	return s
}

func init() {
	aliasesCmd.Flags().Bool("load", false, "Load aliases")
	aliasesCmd.Flags().Bool("unload", false, "Unload aliases")
	rootCmd.AddCommand(aliasesCmd)
}
