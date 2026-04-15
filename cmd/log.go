package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var (
	logLines    int
	logFailures bool
	logSource   string
)

func init() {
	logCmd.Flags().IntVarP(&logLines, "lines", "n", 10, "number of entries to show")
	logCmd.Flags().BoolVarP(&logFailures, "failures", "f", false, "show only failures/errors")
	logCmd.Flags().StringVarP(&logSource, "source", "s", "", "filter by log source (ops|events|audit)")
	rootCmd.AddCommand(logCmd)
}

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Tail the unified ops log across all Warren Command systems",
	Long: `Shows recent entries from ops-log.jsonl, events.jsonl, and audit.jsonl.
Use --source to filter, --failures for errors only, --lines to control count.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()

		// Log sources — local first, then well-known paths
		sources := map[string][]string{
			"events": {
				".wt/events.jsonl",
				filepath.Join(home, "nightagent-sync", "ops", "events.jsonl"),
			},
			"ops": {
				filepath.Join(home, "nightagent-sync", "ops", "ops-log.jsonl"),
			},
			"audit": {
				filepath.Join(home, "nightagent-sync", "ops", "audits", "audit.jsonl"),
			},
		}

		type logEntry struct {
			source   string
			line     string
			parsed   map[string]interface{}
		}

		var entries []logEntry

		for name, paths := range sources {
			if logSource != "" && logSource != name {
				continue
			}
			for _, p := range paths {
				f, err := os.Open(p)
				if err != nil {
					continue
				}
				var lines []string
				scanner := bufio.NewScanner(f)
				// Buffer up to 1MB lines
				scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
				for scanner.Scan() {
					lines = append(lines, scanner.Text())
				}
				f.Close()

				// Take last N lines
				start := len(lines) - logLines
				if start < 0 {
					start = 0
				}
				for _, line := range lines[start:] {
					var parsed map[string]interface{}
					json.Unmarshal([]byte(line), &parsed)

					if logFailures {
						status, _ := parsed["status"].(string)
						result, _ := parsed["result"].(string)
						errMsg, _ := parsed["error"].(string)
						isFailure := status == "fail" || status == "error" || status == "degraded" ||
							result == "fail" || result == "error" || errMsg != ""
						if !isFailure {
							continue
						}
					}

					entries = append(entries, logEntry{
						source: name,
						line:   line,
						parsed: parsed,
					})
				}
				break // Use first found path for this source
			}
		}

		if len(entries) == 0 {
			if logFailures {
				fmt.Println("\033[32m✓ No failures found\033[0m")
			} else {
				fmt.Println("No log entries found.")
				if logSource == "" {
					fmt.Println("  Searched: .wt/events.jsonl, ~/nightagent-sync/ops/")
				}
			}
			return nil
		}

		// Display
		sourceColors := map[string]string{
			"events": "\033[36m",
			"ops":    "\033[34m",
			"audit":  "\033[33m",
		}
		reset := "\033[0m"

		for _, e := range entries {
			color := sourceColors[e.source]
			if color == "" {
				color = reset
			}

			// Format: [source] timestamp — type: message
			ts, _ := e.parsed["ts"].(string)
			if ts == "" {
				ts, _ = e.parsed["timestamp"].(string)
			}
			if len(ts) > 19 {
				ts = ts[:19]
			}

			etype, _ := e.parsed["type"].(string)
			if etype == "" {
				etype, _ = e.parsed["action"].(string)
			}

			msg, _ := e.parsed["message"].(string)
			if msg == "" {
				// For ops-log, build a summary from data
				if svc, ok := e.parsed["services"].(map[string]interface{}); ok {
					active, _ := svc["active"].(float64)
					total, _ := svc["total"].(float64)
					msg = fmt.Sprintf("services %s/%s", strconv.Itoa(int(active)), strconv.Itoa(int(total)))
				}
			}

			// Truncate message
			if len(msg) > 80 {
				msg = msg[:77] + "..."
			}

			status, _ := e.parsed["status"].(string)
			statusIcon := ""
			if status == "fail" || status == "error" || status == "degraded" {
				statusIcon = "\033[31m✗\033[0m "
			}

			tag := fmt.Sprintf("%s[%s]%s", color, e.source, reset)
			parts := []string{statusIcon + tag}
			if ts != "" {
				parts = append(parts, ts)
			}
			if etype != "" {
				parts = append(parts, etype)
			}
			if msg != "" {
				parts = append(parts, "—", msg)
			}

			fmt.Println(strings.Join(parts, " "))
		}

		return nil
	},
}
