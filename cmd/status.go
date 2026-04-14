package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wtchronos/wt-cli/internal/config"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show project status overview",
	Run: func(cmd *cobra.Command, args []string) {
		cfgPath, err := config.Find(".")
		if err != nil {
			fmt.Println("Not in a wt project (no .wt.toml found)")
			os.Exit(1)
		}

		cfg, err := config.Load(cfgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
			os.Exit(1)
		}

		projectRoot := filepath.Dir(cfgPath)
		fmt.Printf("Project: %s\n", cfg.Project.Name)
		fmt.Printf("Root:    %s\n", projectRoot)
		fmt.Printf("Config:  %s\n", cfgPath)

		// Git status
		gitStatus := runQuiet(projectRoot, "git", "status", "--porcelain")
		if gitStatus != "" {
			lines := strings.Split(strings.TrimSpace(gitStatus), "\n")
			fmt.Printf("Git:     %d uncommitted changes\n", len(lines))
		} else {
			fmt.Println("Git:     clean")
		}

		branch := strings.TrimSpace(runQuiet(projectRoot, "git", "rev-parse", "--abbrev-ref", "HEAD"))
		if branch != "" {
			fmt.Printf("Branch:  %s\n", branch)
		}

		// Hooks
		gitHookCount := 0
		if cfg.Hooks.Git != nil {
			for _, cmds := range cfg.Hooks.Git {
				gitHookCount += len(cmds)
			}
		}
		enterCount := len(cfg.Hooks.Enter.Commands)
		leaveCount := len(cfg.Hooks.Leave.Commands)
		fmt.Printf("Hooks:   %d git, %d enter, %d leave\n", gitHookCount, enterCount, leaveCount)

		// Aliases
		if len(cfg.Aliases) > 0 {
			names := make([]string, 0, len(cfg.Aliases))
			for k := range cfg.Aliases {
				names = append(names, k)
			}
			sort.Strings(names)
			fmt.Printf("Aliases: %s\n", strings.Join(names, ", "))
		}

		// Scripts
		if len(cfg.Scripts) > 0 {
			names := make([]string, 0, len(cfg.Scripts))
			for k := range cfg.Scripts {
				names = append(names, k)
			}
			sort.Strings(names)
			fmt.Printf("Scripts: %s\n", strings.Join(names, ", "))
		}

		// Env vars
		if len(cfg.Env) > 0 {
			fmt.Printf("Env:     %d project variables\n", len(cfg.Env))
		}

		// Operator config
		if cfg.Operator.CortixURL != "" {
			fmt.Printf("Cortix:  %s\n", cfg.Operator.CortixURL)
			if len(cfg.Operator.Tags) > 0 {
				fmt.Printf("Tags:    %s\n", strings.Join(cfg.Operator.Tags, ", "))
			}
		}
	},
}

func runQuiet(dir string, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
