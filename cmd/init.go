package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wtchronos/wt-cli/internal/githooks"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a project with wt configuration",
	Run: func(cmd *cobra.Command, args []string) {
		// Create .wt.toml if it doesn't exist
		configFile := ".wt.toml"
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			template := `[project]
name = "myproject"

[prompt]
segment = "[{{.Project.Name}}]"

[hooks.git]
post-checkout = []
post-merge = []
post-commit = []

[hooks.enter]
commands = []

[hooks.leave]
commands = []

[aliases]
# example = "command"
`
			if err := os.WriteFile(configFile, []byte(template), 0644); err != nil {
				fmt.Printf("Error creating %s: %v\n", configFile, err)
				os.Exit(1)
			}
			fmt.Printf("Created %s\n", configFile)
		} else {
			fmt.Printf("%s already exists\n", configFile)
		}

		// Install git hooks
		if err := githooks.Install(); err != nil {
			fmt.Printf("Error installing git hooks: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
