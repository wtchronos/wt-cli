package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"github.com/wtchronos/wt-cli/internal/config"
)

var runCmd = &cobra.Command{
	Use:   "run <script> [args...]",
	Short: "Run a project-defined script",
	Long:  `Run a named script defined in .wt.toml under [scripts]. Pass additional arguments after the script name.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scriptName := args[0]

		cfgPath, err := config.Find(".")
		if err != nil {
			fmt.Fprintln(os.Stderr, "No .wt.toml found")
			os.Exit(1)
		}

		cfg, err := config.Load(cfgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
			os.Exit(1)
		}

		scriptCmd, ok := cfg.Scripts[scriptName]
		if !ok {
			fmt.Fprintf(os.Stderr, "Unknown script: %s\n", scriptName)
			if len(cfg.Scripts) > 0 {
				fmt.Fprintln(os.Stderr, "\nAvailable scripts:")
				names := make([]string, 0, len(cfg.Scripts))
				for k := range cfg.Scripts {
					names = append(names, k)
				}
				sort.Strings(names)
				for _, n := range names {
					fmt.Fprintf(os.Stderr, "  %s\t%s\n", n, cfg.Scripts[n])
				}
			}
			os.Exit(1)
		}

		// Append extra args
		if len(args) > 1 {
			scriptCmd = scriptCmd + " " + argsToString(args[1:])
		}

		projectRoot := filepath.Dir(cfgPath)

		// Inject project env vars
		environ := os.Environ()
		environ = append(environ, "WT_PROJECT_NAME="+cfg.Project.Name)
		environ = append(environ, "WT_PROJECT_ROOT="+projectRoot)
		for k, v := range cfg.Env {
			environ = append(environ, k+"="+v)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "[wt run/%s] %s\n", scriptName, scriptCmd)
		}

		c := exec.Command("sh", "-c", scriptCmd)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin
		c.Dir = projectRoot
		c.Env = environ

		if err := c.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func argsToString(args []string) string {
	result := ""
	for i, a := range args {
		if i > 0 {
			result += " "
		}
		result += "'" + a + "'"
	}
	return result
}

func init() {
	rootCmd.AddCommand(runCmd)
}
