package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/wtchronos/wt-cli/internal/config"
)

// ANSI color helpers — no external deps
var ansiColors = map[string]string{
	"reset":   "\033[0m",
	"bold":    "\033[1m",
	"black":   "\033[30m",
	"red":     "\033[31m",
	"green":   "\033[32m",
	"yellow":  "\033[33m",
	"blue":    "\033[34m",
	"magenta": "\033[35m",
	"cyan":    "\033[36m",
	"white":   "\033[37m",
}

func colorFunc(name, s string) string {
	code, ok := ansiColors[name]
	if !ok {
		return s
	}
	return code + s + ansiColors["reset"]
}

// promptFuncMap exposes color helpers inside segment templates.
// Usage: {{cyan .Project.Name}} or {{color "green" .Project.Name}}
var promptFuncMap = template.FuncMap{
	"color":   colorFunc,
	"black":   func(s string) string { return colorFunc("black", s) },
	"red":     func(s string) string { return colorFunc("red", s) },
	"green":   func(s string) string { return colorFunc("green", s) },
	"yellow":  func(s string) string { return colorFunc("yellow", s) },
	"blue":    func(s string) string { return colorFunc("blue", s) },
	"magenta": func(s string) string { return colorFunc("magenta", s) },
	"cyan":    func(s string) string { return colorFunc("cyan", s) },
	"white":   func(s string) string { return colorFunc("white", s) },
	"bold":    func(s string) string { return colorFunc("bold", s) },
}

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Print prompt segment for current context",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, err := config.Find(".")
		if err != nil {
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

		tmpl, err := template.New("prompt").Funcs(promptFuncMap).Parse(cfg.Prompt.Segment)
		if err != nil {
			fmt.Printf("Error parsing prompt template: %v\n", err)
			return
		}

		var output strings.Builder
		if err := tmpl.Execute(&output, cfg); err != nil {
			fmt.Printf("Error executing prompt template: %v\n", err)
			return
		}

		fmt.Print(output.String())
	},
}

func init() {
	rootCmd.AddCommand(promptCmd)
}
