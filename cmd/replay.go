package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	replayLast     int
	replayFailures bool
)

func init() {
	replayCmd.Flags().IntVarP(&replayLast, "last", "n", 20, "number of entries to show")
	replayCmd.Flags().BoolVarP(&replayFailures, "failures", "f", false, "show only failures/errors")
	rootCmd.AddCommand(replayCmd)
}

type replayEntry struct {
	ts      time.Time
	source  string
	intent  string
	route   string
	result  string
	status  string
	raw     map[string]interface{}
}

var replayCmd = &cobra.Command{
	Use:   "replay",
	Short: "Show intent → route → result chains from ops logs",
	Long: `Walks ops-log.jsonl, events.jsonl, and audit.jsonl to display
intent → route → result chains. Use --failures for errors only,
--last N to control the number of entries shown.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()

		sources := []struct {
			name  string
			paths []string
		}{
			{
				name: "ops",
				paths: []string{
					filepath.Join(home, "nightagent-sync", "ops", "ops-log.jsonl"),
					".wt/ops-log.jsonl",
				},
			},
			{
				name: "events",
				paths: []string{
					filepath.Join(home, "nightagent-sync", "ops", "events.jsonl"),
					".wt/events.jsonl",
				},
			},
			{
				name: "audit",
				paths: []string{
					filepath.Join(home, "nightagent-sync", "ops", "audits", "audit.jsonl"),
					".wt/audit.jsonl",
				},
			},
		}

		var entries []replayEntry

		for _, src := range sources {
			for _, p := range src.paths {
				f, err := os.Open(p)
				if err != nil {
					continue
				}

				var lines []string
				scanner := bufio.NewScanner(f)
				scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
				for scanner.Scan() {
					lines = append(lines, scanner.Text())
				}
				f.Close()

				for _, line := range lines {
					if strings.TrimSpace(line) == "" {
						continue
					}
					var parsed map[string]interface{}
					if err := json.Unmarshal([]byte(line), &parsed); err != nil {
						continue
					}

					entry := replayEntry{source: src.name, raw: parsed}
					entry.status = strField(parsed, "status", "result")
					entry.intent = strField(parsed, "intent", "description", "action", "type")
					entry.route = strField(parsed, "route", "routing", "handler", "processor")
					entry.result = strField(parsed, "result_summary", "message", "result")

					// Parse timestamp
					tsStr := strField(parsed, "ts", "timestamp", "created_at")
					if tsStr != "" {
						for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
							if t, err := time.Parse(layout, tsStr); err == nil {
								entry.ts = t
								break
							}
						}
					}
					if entry.ts.IsZero() {
						entry.ts = time.Now() // fallback so it sorts to end
					}

					entries = append(entries, entry)
				}
				break // use first found path
			}
		}

		if len(entries) == 0 {
			fmt.Println("No log entries found.")
			fmt.Println("  Searched: ~/nightagent-sync/ops/, .wt/")
			return nil
		}

		// Sort by timestamp descending (newest first)
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].ts.After(entries[j].ts)
		})

		// Filter failures if requested
		if replayFailures {
			var filtered []replayEntry
			for _, e := range entries {
				if isFailure(e.status) || isFailure(e.result) {
					filtered = append(filtered, e)
				}
			}
			entries = filtered
		}

		// Cap at --last N
		if replayLast > 0 && len(entries) > replayLast {
			entries = entries[:replayLast]
		}

		if len(entries) == 0 {
			if replayFailures {
				fmt.Println("\033[32m✓ No failures found\033[0m")
			} else {
				fmt.Println("No entries to display.")
			}
			return nil
		}

		// Reverse so oldest is at top (chronological order in output)
		for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
			entries[i], entries[j] = entries[j], entries[i]
		}

		fmt.Printf("\033[1;34m⬡ Replay\033[0m  %d entries", len(entries))
		if replayFailures {
			fmt.Printf("  \033[31m(failures only)\033[0m")
		}
		fmt.Println()
		fmt.Println()

		sourceColors := map[string]string{
			"ops":    "\033[34m",
			"events": "\033[36m",
			"audit":  "\033[33m",
		}
		reset := "\033[0m"

		for _, e := range entries {
			color := sourceColors[e.source]
			if color == "" {
				color = reset
			}

			// Status icon
			icon := "\033[32m●\033[0m"
			if isFailure(e.status) || isFailure(e.result) {
				icon = "\033[31m✗\033[0m"
			} else if e.status == "pending" || e.status == "queued" {
				icon = "\033[33m○\033[0m"
			}

			ts := e.ts.Format("01-02 15:04:05")
			tag := fmt.Sprintf("%s[%s]%s", color, e.source, reset)

			// Build chain: intent → route → result
			chain := buildChain(e.intent, e.route, e.result)

			fmt.Printf("  %s %s %s  %s\n", icon, ts, tag, chain)
		}

		fmt.Println()
		return nil
	},
}

// strField returns the first non-empty string value found for any of the given keys.
func strField(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func isFailure(s string) bool {
	switch strings.ToLower(s) {
	case "fail", "failed", "error", "degraded", "timeout":
		return true
	}
	return false
}

func buildChain(intent, route, result string) string {
	parts := []string{}
	if intent != "" {
		if len(intent) > 40 {
			intent = intent[:37] + "..."
		}
		parts = append(parts, intent)
	}
	if route != "" {
		parts = append(parts, "\033[90m→\033[0m "+route)
	}
	if result != "" {
		if len(result) > 50 {
			result = result[:47] + "..."
		}
		parts = append(parts, "\033[90m→\033[0m "+result)
	}
	return strings.Join(parts, " ")
}
