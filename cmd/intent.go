package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wtchronos/wt-cli/internal/config"
)

var intentPriority string

func init() {
	intentCmd.Flags().StringVarP(&intentPriority, "priority", "p", "P1", "priority level (P0/P1/P2)")
	rootCmd.AddCommand(intentCmd)
}

var intentCmd = &cobra.Command{
	Use:   "intent <description>",
	Short: "Submit an intent to the Cortix intent bridge",
	Long: `Submits a structured intent to the Cortix kernel for routing, approval, and execution.
Intents flow through the trust ladder — P0 requires human approval, P1/P2 may auto-execute
for graduated categories.

Examples:
  wt intent "run test suite and report coverage"
  wt intent -p P0 "deploy latest to VPS"
  wt intent "clean up dead code in alerter.py"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		description := strings.Join(args, " ")

		cwd, _ := os.Getwd()
		cfgPath, _ := config.Find(cwd)
		var cortixURL, apiKey, project string

		if cfgPath != "" {
			cfg, err := config.Load(cfgPath)
			if err == nil {
				if cfg.Operator.CortixURL != "" {
					cortixURL = cfg.Operator.CortixURL
				}
				if cfg.Operator.APIKey != "" {
					apiKey = cfg.Operator.APIKey
				}
				project = cfg.Project.Name
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
		if project == "" {
			project = "wt-cli"
		}

		// Build intent payload
		intent := map[string]interface{}{
			"source":      "wt-cli",
			"type":        "intent",
			"description": description,
			"priority":    intentPriority,
			"project":     project,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
			"metadata": map[string]string{
				"cwd":    cwd,
				"origin": "terminal",
			},
		}

		body, err := json.Marshal(intent)
		if err != nil {
			return fmt.Errorf("marshal intent: %w", err)
		}

		fmt.Printf("\033[34m⬡ Submitting intent\033[0m\n")
		fmt.Printf("  %s\n", description)
		fmt.Printf("  Priority: %s | Project: %s\n", intentPriority, project)

		// Submit to Cortix
		url := cortixURL + "/api/intent"
		req, err := http.NewRequest("POST", url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			req.Header.Set("X-API-Key", apiKey)
		}

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("\n\033[33m⚠ Cortix unreachable\033[0m — logging intent locally\n")
			return logIntentLocally(intent)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)

		if resp.StatusCode >= 400 {
			fmt.Printf("\n\033[33m⚠ Cortix returned %d\033[0m — logging intent locally\n", resp.StatusCode)
			if verbose {
				fmt.Printf("  Response: %s\n", string(respBody))
			}
			return logIntentLocally(intent)
		}

		// Parse response
		var result map[string]interface{}
		if json.Unmarshal(respBody, &result) == nil {
			fmt.Printf("\n\033[32m✓ Intent submitted\033[0m\n")
			if id, ok := result["id"].(string); ok {
				fmt.Printf("  ID: %s\n", id)
			}
			if status, ok := result["status"].(string); ok {
				fmt.Printf("  Status: %s\n", status)
			}
			if routing, ok := result["routing"].(string); ok {
				fmt.Printf("  Routing: %s\n", routing)
			}
		} else {
			fmt.Printf("\n\033[32m✓ Intent submitted\033[0m (HTTP %d)\n", resp.StatusCode)
		}

		return nil
	},
}

func logIntentLocally(intent map[string]interface{}) error {
	dir := ".wt"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(dir+"/intents.jsonl", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	line, _ := json.Marshal(intent)
	fmt.Fprintf(f, "%s\n", line)
	fmt.Printf("  Saved to .wt/intents.jsonl\n")
	return nil
}
