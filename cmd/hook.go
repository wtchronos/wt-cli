package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wtchronos/wt-cli/internal/config"
)

var hookCmd = &cobra.Command{
	Use:   "hook run <event>",
	Short: "Run hooks for the specified event",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		event := args[0]
		
		// Prevent recursion
		if os.Getenv("WT_IN_HOOK") == "1" {
			return
		}
		
		// Find and load config
		configPath, err := config.Find(".")
		if err != nil {
			// No config found, nothing to do
			return
		}
		
		cfg, err := config.Load(configPath)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		
		// Set environment variables
		os.Setenv("WT_IN_HOOK", "1")
		os.Setenv("WT_PROJECT_ROOT", configPath)
		os.Setenv("WT_EVENT", event)
		
		var commands []string
		
		// Get commands based on event type
		switch {
		case strings.HasPrefix(event, "post-"):
			// Git hook
			if cfg.Hooks.Git != nil {
				commands = cfg.Hooks.Git[event]
			}
		case event == "enter":
			commands = cfg.Hooks.Enter.Commands
		case event == "leave":
			commands = cfg.Hooks.Leave.Commands
		}
		
		if commands == nil {
			return
		}
		
		// Execute commands
		for _, cmdStr := range commands {
			if cmdStr == "" {
				continue
			}
			
			// Expand template variables
			cmdStr = strings.ReplaceAll(cmdStr, "{{.Project.Name}}", cfg.Project.Name)
			
			fmt.Printf("Running hook: %s\n", cmdStr)
			cmd := exec.Command("sh", "-c", cmdStr)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Dir = "."
			
			if err := cmd.Run(); err != nil {
				fmt.Printf("Hook failed: %v\n", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(hookCmd)
}
