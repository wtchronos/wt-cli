package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wtchronos/wt-cli/internal/config"
)

func init() {
	rootCmd.AddCommand(topCmd)
}

var topCmd = &cobra.Command{
	Use:   "top",
	Short: "Compact status: services + last tick results",
	Long:  `Shows a compact terminal status view: Cortix services, last tick results, and recent intent bridge activity.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		cfgPath, _ := config.Find(cwd)
		var cortixURL, apiKey string

		if cfgPath != "" {
			cfg, err := config.Load(cfgPath)
			if err == nil {
				if cfg.Operator.CortixURL != "" {
					cortixURL = cfg.Operator.CortixURL
				}
				if cfg.Operator.APIKey != "" {
					apiKey = cfg.Operator.APIKey
				}
			}
		}
		if cortixURL == "" {
			cortixURL = os.Getenv("CORTIX_URL")
		}
		if cortixURL == "" {
			cortixURL = "https://command.warrencommand.dev"
		}
		if apiKey == "" {
			apiKey = os.Getenv("CORTIX_API_KEY")
		}

		now := time.Now().Format("15:04:05")
		fmt.Printf("\033[1;34m⬡ wt top\033[0m  %s\n\n", now)

		// --- Services section ---
		fmt.Printf("\033[1mSERVICES\033[0m\n")
		printServices(cortixURL, apiKey)

		fmt.Println()

		// --- Last tick results from local logs ---
		fmt.Printf("\033[1mLAST TICKS\033[0m\n")
		printLastTicks()

		fmt.Println()

		// --- Recent intent bridge activity ---
		fmt.Printf("\033[1mINTENT BRIDGE\033[0m\n")
		printIntentBridge()

		fmt.Println()
		return nil
	},
}

func printServices(cortixURL, apiKey string) {
	client := &http.Client{Timeout: 4 * time.Second}

	req, err := http.NewRequest("GET", cortixURL+"/api/health", nil)
	if err != nil {
		fmt.Printf("  \033[31m✗ Cortix unreachable\033[0m (%s)\n", cortixURL)
		return
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  \033[31m✗ Cortix unreachable\033[0m — %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var health map[string]interface{}
	if err := json.Unmarshal(body, &health); err != nil {
		fmt.Printf("  \033[31m✗ Cortix\033[0m parse error\n")
		return
	}

	kernelStatus := "unknown"
	if s, ok := health["status"].(string); ok {
		kernelStatus = s
	}

	kernelIcon := serviceIcon(kernelStatus)
	fmt.Printf("  %s cortix-kernel   %s\n", kernelIcon, kernelStatus)

	if services, ok := health["services"].(map[string]interface{}); ok {
		// Sort keys for deterministic output
		names := make([]string, 0, len(services))
		for k := range services {
			names = append(names, k)
		}
		sortStrings(names)
		for _, name := range names {
			st, _ := services[name].(string)
			icon := serviceIcon(st)
			padded := name
			for len(padded) < 15 {
				padded += " "
			}
			fmt.Printf("  %s %s %s\n", icon, padded, st)
		}
	} else {
		// No services breakdown — just show kernel status
	}
}

func serviceIcon(status string) string {
	switch strings.ToLower(status) {
	case "active", "ok", "running", "healthy":
		return "\033[32m●\033[0m"
	case "degraded", "warn", "warning":
		return "\033[33m●\033[0m"
	default:
		return "\033[31m●\033[0m"
	}
}

func sortStrings(ss []string) {
	// Simple insertion sort (avoids importing sort for a tiny list)
	for i := 1; i < len(ss); i++ {
		key := ss[i]
		j := i - 1
		for j >= 0 && ss[j] > key {
			ss[j+1] = ss[j]
			j--
		}
		ss[j+1] = key
	}
}

func printLastTicks() {
	home, _ := os.UserHomeDir()
	paths := []string{
		filepath.Join(home, "nightagent-sync", "ops", "ops-log.jsonl"),
		".wt/ops-log.jsonl",
	}

	type tickEntry struct {
		name   string
		status string
		ts     string
	}

	var ticks []tickEntry
	seen := map[string]bool{}

	for _, p := range paths {
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

		// Walk from end to get most recent entries per tick name
		for i := len(lines) - 1; i >= 0 && len(ticks) < 8; i-- {
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(lines[i]), &parsed); err != nil {
				continue
			}

			name := strField(parsed, "tick", "action", "type", "processor")
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true

			status := strField(parsed, "status", "result")
			ts := strField(parsed, "ts", "timestamp", "created_at")
			if len(ts) > 16 {
				ts = ts[:16]
			}

			ticks = append(ticks, tickEntry{name: name, status: status, ts: ts})
		}
		break
	}

	if len(ticks) == 0 {
		fmt.Printf("  \033[90mno tick data\033[0m  (~/nightagent-sync/ops/ops-log.jsonl)\n")
		return
	}

	for _, t := range ticks {
		icon := "\033[32m●\033[0m"
		if isFailure(t.status) {
			icon = "\033[31m✗\033[0m"
		}
		name := t.name
		for len(name) < 28 {
			name += " "
		}
		tsDisplay := t.ts
		if tsDisplay == "" {
			tsDisplay = "—"
		}
		fmt.Printf("  %s %s %s  %s\n", icon, name, tsDisplay, t.status)
	}
}

func printIntentBridge() {
	home, _ := os.UserHomeDir()
	paths := []string{
		filepath.Join(home, "nightagent-sync", "ops", "events.jsonl"),
		".wt/events.jsonl",
		".wt/intents.jsonl",
	}

	type intentEntry struct {
		intent string
		status string
		ts     string
	}

	var intents []intentEntry

	for _, p := range paths {
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

		// Last 5 entries
		start := len(lines) - 5
		if start < 0 {
			start = 0
		}
		for _, line := range lines[start:] {
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(line), &parsed); err != nil {
				continue
			}
			intent := strField(parsed, "description", "intent", "action", "message", "type")
			status := strField(parsed, "status", "result")
			ts := strField(parsed, "ts", "timestamp", "created_at")
			if len(ts) > 16 {
				ts = ts[:16]
			}
			if len(intent) > 45 {
				intent = intent[:42] + "..."
			}
			intents = append(intents, intentEntry{intent: intent, status: status, ts: ts})
		}
		break
	}

	if len(intents) == 0 {
		fmt.Printf("  \033[90mno intent data\033[0m  (.wt/intents.jsonl or nightagent-sync)\n")
		return
	}

	for _, e := range intents {
		icon := "\033[36m○\033[0m"
		if isFailure(e.status) {
			icon = "\033[31m✗\033[0m"
		} else if e.status == "ok" || e.status == "done" || e.status == "complete" {
			icon = "\033[32m✓\033[0m"
		}
		tsDisplay := e.ts
		if tsDisplay == "" {
			tsDisplay = "—"
		}
		fmt.Printf("  %s %s  %s\n", icon, tsDisplay, e.intent)
	}
}
