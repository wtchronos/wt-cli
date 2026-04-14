package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/wtchronos/wt-cli/internal/config"
	"github.com/wtchronos/wt-cli/internal/operator"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Flush queued local events to the operator surface",
	Long: `When wt emit can't reach Cortix, events are queued to .wt/events.jsonl.
This command replays them to the operator surface and clears the local queue on success.`,
	Run: func(cmd *cobra.Command, args []string) {
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

		if cfg.Operator.CortixURL == "" {
			fmt.Fprintln(os.Stderr, "No [operator] cortix_url configured in .wt.toml")
			os.Exit(1)
		}

		projectRoot := filepath.Dir(cfgPath)
		logPath := filepath.Join(projectRoot, ".wt", "events.jsonl")

		f, err := os.Open(logPath)
		if err != nil {
			fmt.Println("No queued events to sync.")
			return
		}
		defer f.Close()

		emitter := &operator.Emitter{
			CortixURL: cfg.Operator.CortixURL,
			APIKey:    cfg.Operator.APIKey,
			Project:   cfg.Project.Name,
			Tags:      cfg.Operator.Tags,
			Verbose:   verbose,
		}

		var sent, failed int
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var ev operator.Event
			if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
				failed++
				continue
			}

			if err := emitter.Emit(ev.Type, ev.Message, ev.Metadata); err != nil {
				failed++
				if verbose {
					fmt.Fprintf(os.Stderr, "[wt sync] failed: %v\n", err)
				}
			} else {
				sent++
			}
		}

		if failed > 0 {
			fmt.Printf("Synced %d events, %d failed (kept in queue)\n", sent, failed)
			return
		}

		// All sent — clear the log
		os.Remove(logPath)
		fmt.Printf("Synced %d events. Queue cleared.\n", sent)
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
