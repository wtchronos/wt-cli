package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wtchronos/wt-cli/internal/config"
	"github.com/wtchronos/wt-cli/internal/operator"
)

var emitCmd = &cobra.Command{
	Use:   "emit <type> <message>",
	Short: "Emit an event to the operator surface",
	Long: `Send a project event to Cortix (or log locally if unreachable).
Events are typed strings with optional key=value metadata.

Examples:
  wt emit deploy "deployed to production"
  wt emit build "tests passing" --meta status=green,tests=142`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		eventType := args[0]
		message := strings.Join(args[1:], " ")

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

		metadata := parseMetadata(metaFlag)

		emitter := &operator.Emitter{
			CortixURL: cfg.Operator.CortixURL,
			APIKey:    cfg.Operator.APIKey,
			Project:   cfg.Project.Name,
			Tags:      cfg.Operator.Tags,
			Verbose:   verbose,
		}

		if err := emitter.Emit(eventType, message, metadata); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "Event emitted: %s\n", eventType)
		}
	},
}

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Show local event log",
	Run: func(cmd *cobra.Command, args []string) {
		cfgPath, err := config.Find(".")
		if err != nil {
			fmt.Fprintln(os.Stderr, "No .wt.toml found")
			os.Exit(1)
		}

		projectRoot := filepath.Dir(cfgPath)
		logPath := filepath.Join(projectRoot, ".wt", "events.jsonl")
		data, err := os.ReadFile(logPath)
		if err != nil {
			fmt.Println("No events logged yet.")
			return
		}

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		for _, line := range lines {
			var ev operator.Event
			if err := json.Unmarshal([]byte(line), &ev); err != nil {
				continue
			}
			fmt.Printf("[%s] %s: %s\n", ev.Ts[:19], ev.Type, ev.Message)
		}
	},
}

var metaFlag string

func parseMetadata(raw string) map[string]string {
	if raw == "" {
		return nil
	}
	meta := make(map[string]string)
	for _, pair := range strings.Split(raw, ",") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			meta[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return meta
}

func init() {
	emitCmd.Flags().StringVar(&metaFlag, "meta", "", "key=value metadata pairs (comma-separated)")
	rootCmd.AddCommand(emitCmd)
	rootCmd.AddCommand(eventsCmd)
}
