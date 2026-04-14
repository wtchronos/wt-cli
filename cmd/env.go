package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/wtchronos/wt-cli/internal/config"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Show or export project environment variables",
	Long: `Print project-defined environment variables from .wt.toml [env] section.
Use with eval to inject into your shell: eval "$(wt env --export)"`,
}

var envShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display project environment variables",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := mustLoadConfig()
		if len(cfg.Env) == 0 {
			fmt.Println("No project environment variables defined.")
			return
		}

		keys := sortedKeys(cfg.Env)
		for _, k := range keys {
			fmt.Printf("%s=%s\n", k, cfg.Env[k])
		}
	},
}

var envExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Print eval-able export statements",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := mustLoadConfig()
		keys := sortedKeys(cfg.Env)
		for _, k := range keys {
			fmt.Printf("export %s='%s'\n", k, cfg.Env[k])
		}
	},
}

func mustLoadConfig() *config.Config {
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
	return cfg
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func init() {
	envCmd.AddCommand(envShowCmd)
	envCmd.AddCommand(envExportCmd)
	rootCmd.AddCommand(envCmd)
}
