package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wtchronos/wt-cli/internal/config"
	"github.com/wtchronos/wt-cli/internal/hooks"
)

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Manage and run project hooks",
}

var hookRunCmd = &cobra.Command{
	Use:   "run <event> [git-args...]",
	Short: "Run hooks for the specified event",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		event := args[0]

		if os.Getenv("WT_IN_HOOK") == "1" {
			return
		}

		cfgPath, err := config.Find(".")
		if err != nil {
			return
		}

		cfg, err := config.Load(cfgPath)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "[wt] config error: %v\n", err)
			}
			return
		}

		projectRoot := filepath.Dir(cfgPath)
		runner := &hooks.Runner{
			ProjectName: cfg.Project.Name,
			ProjectRoot: projectRoot,
			Verbose:     verbose,
		}

		var commands []string
		switch {
		case event == "enter":
			commands = cfg.Hooks.Enter.Commands
		case event == "leave":
			commands = cfg.Hooks.Leave.Commands
		default:
			// Git hook events (pre-commit, post-checkout, etc.)
			if cfg.Hooks.Git != nil {
				commands = cfg.Hooks.Git[event]
			}
		}

		if len(commands) == 0 {
			return
		}

		// Pass git hook args as WT_HOOK_ARGS env var
		if len(args) > 1 {
			os.Setenv("WT_HOOK_ARGS", strings.Join(args[1:], " "))
		}

		if err := runner.Run(event, commands); err != nil {
			fmt.Fprintf(os.Stderr, "[wt] hook error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	hookCmd.AddCommand(hookRunCmd)
	rootCmd.AddCommand(hookCmd)
}
