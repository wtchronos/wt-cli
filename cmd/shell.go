package cmd

import (
	"fmt"
	"os"
	"text/template"

	"github.com/spf13/cobra"
)

type shellTemplateData struct {
	WtPath string
}

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Shell integration commands",
}

var shellInitCmd = &cobra.Command{
	Use:   "init [bash|zsh|fish]",
	Short: "Emit eval-able shell integration script",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		shell := args[0]

		wtPath, err := os.Executable()
		if err != nil {
			wtPath = "wt"
		}

		data := shellTemplateData{WtPath: wtPath}

		var tmplStr string
		switch shell {
		case "bash":
			tmplStr = bashTemplate
		case "zsh":
			tmplStr = zshTemplate
		case "fish":
			tmplStr = fishTemplate
		default:
			fmt.Printf("Unsupported shell: %s\n", shell)
			os.Exit(1)
		}

		tmpl, err := template.New("shell").Parse(tmplStr)
		if err != nil {
			fmt.Printf("Error parsing template: %v\n", err)
			os.Exit(1)
		}

		if err := tmpl.Execute(os.Stdout, data); err != nil {
			fmt.Printf("Error executing template: %v\n", err)
			os.Exit(1)
		}
	},
}

const bashTemplate = `# wt shell integration for bash
__wt_chpwd() {
    local current="$PWD"
    if [[ "$current" != "$__WT_LAST_DIR" ]]; then
        # Unload previous aliases
        if [[ -n "$__WT_LOADED_ALIASES" ]]; then
            eval "$("{{.WtPath}}" aliases --unload)"
        fi
        
        # Run leave hooks if we were in a wt project
        if [[ -n "$__WT_LAST_DIR" ]]; then
            (cd "$__WT_LAST_DIR" && "{{.WtPath}}" hook run leave 2>/dev/null || true)
        fi
        
        # Load new aliases, env, and run enter hooks
        local aliases_output env_output
        if aliases_output="$("{{.WtPath}}" aliases --load 2>/dev/null)"; then
            eval "$aliases_output"
            __WT_LOADED_ALIASES=1
            # Inject project env vars
            if env_output="$("{{.WtPath}}" env export 2>/dev/null)"; then
                eval "$env_output"
                __WT_LOADED_ENV=1
            fi
            "{{.WtPath}}" hook run enter 2>/dev/null || true
        else
            unset __WT_LOADED_ALIASES
            unset __WT_LOADED_ENV
        fi

        __WT_LAST_DIR="$current"
    fi
}

# Add to PROMPT_COMMAND
if [[ -z "$PROMPT_COMMAND" ]]; then
    PROMPT_COMMAND="__wt_chpwd"
else
    PROMPT_COMMAND="__wt_chpwd;$PROMPT_COMMAND"
fi

# Prompt function
wt_prompt() {
    "{{.WtPath}}" prompt 2>/dev/null || true
}

# Convenience: wt run shortcut
wtr() {
    "{{.WtPath}}" run "$@"
}

# Add to PS1 if not already there
if [[ "$PS1" != *"wt_prompt"* ]]; then
    PS1='$(wt_prompt)'"$PS1"
fi
`

const zshTemplate = `# wt shell integration for zsh
__wt_chpwd() {
    local current="$PWD"
    if [[ "$current" != "$__WT_LAST_DIR" ]]; then
        # Unload previous aliases
        if [[ -n "$__WT_LOADED_ALIASES" ]]; then
            eval "$("{{.WtPath}}" aliases --unload)"
        fi
        
        # Run leave hooks if we were in a wt project
        if [[ -n "$__WT_LAST_DIR" ]]; then
            (cd "$__WT_LAST_DIR" && "{{.WtPath}}" hook run leave 2>/dev/null || true)
        fi
        
        # Load new aliases, env, and run enter hooks
        local aliases_output env_output
        if aliases_output="$("{{.WtPath}}" aliases --load 2>/dev/null)"; then
            eval "$aliases_output"
            __WT_LOADED_ALIASES=1
            if env_output="$("{{.WtPath}}" env export 2>/dev/null)"; then
                eval "$env_output"
                __WT_LOADED_ENV=1
            fi
            "{{.WtPath}}" hook run enter 2>/dev/null || true
        else
            unset __WT_LOADED_ALIASES
            unset __WT_LOADED_ENV
        fi

        __WT_LAST_DIR="$current"
    fi
}

# Add to chpwd_functions
chpwd_functions+=(__wt_chpwd)

# Prompt function
wt_prompt() {
    "{{.WtPath}}" prompt 2>/dev/null || true
}

# Convenience: wt run shortcut
wtr() {
    "{{.WtPath}}" run "$@"
}

# Add to PROMPT
if [[ "$PROMPT" != *"wt_prompt"* ]]; then
    PROMPT='$(wt_prompt)'"$PROMPT"
fi
`

const fishTemplate = `# wt shell integration for fish
function __wt_on_pwd --on-variable PWD
    set -l current "$PWD"
    if test "$current" != "$__WT_LAST_DIR"
        # Unload previous aliases
        if set -q __WT_LOADED_ALIASES
            "{{.WtPath}}" aliases --unload | source
        end
        
        # Run leave hooks if we were in a wt project
        if set -q __WT_LAST_DIR
            cd "$__WT_LAST_DIR" && "{{.WtPath}}" hook run leave 2>/dev/null
        end
        
        # Load new aliases and run enter hooks
        if "{{.WtPath}}" aliases --load 2>/dev/null | source
            set -g __WT_LOADED_ALIASES 1
            "{{.WtPath}}" hook run enter 2>/dev/null
        else
            set -e __WT_LOADED_ALIASES
        end
        
        set -g __WT_LAST_DIR "$current"
    end
end

# Prompt integration
function fish_prompt
    "{{.WtPath}}" prompt 2>/dev/null
    echo -n (fish_default_prompt)
end
`

func init() {
	shellCmd.AddCommand(shellInitCmd)
	rootCmd.AddCommand(shellCmd)
}
