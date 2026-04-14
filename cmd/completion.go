package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `To load completions:

Bash:

  $ source <(wt completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ wt completion bash > /etc/bash_completion.d/wt
  # macOS:
  $ wt completion bash > $(brew --prefix)/etc/bash_completion.d/wt

Zsh:

  $ wt completion zsh > "${fpath[1]}/_wt"

Fish:

  $ wt completion fish | source

`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		switch args[0] {
		case "bash":
			err = rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			err = rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			err = rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			err = rootCmd.GenPowerShellCompletion(os.Stdout)
		default:
			fmt.Fprintf(os.Stderr, "Unsupported shell: %s\n", args[0])
			os.Exit(1)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating completion: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
