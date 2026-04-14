package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/wtchronos/wt-cli/internal/config"
)

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Print prompt segment for current context",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, err := config.Find(".")
		if err != nil {
			// No config found, no prompt to show
			return
		}
		
		cfg, err := config.Load(configPath)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		
		if cfg.Prompt.Segment == "" {
			return
		}
		
		// Create a safe template
		tmpl, err := template.New("prompt").Parse(cfg.Prompt.Segment)
		if err != nil {
			fmt.Printf("Error parsing prompt template: %v\n", err)
			return
		}
		
		// Execute template
		var output strings.Builder
		err = tmpl.Execute(&output, cfg)
		if err != nil {
			fmt.Printf("Error executing prompt template: %v\n", err)
			return
		}
		
		fmt.Print(output.String())
	},
}

func init() {
	rootCmd.AddCommand(promptCmd)
}
