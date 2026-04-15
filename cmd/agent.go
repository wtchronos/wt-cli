package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/wtchronos/wt-cli/internal/config"
)

func init() {
	rootCmd.AddCommand(agentCmd)
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Query agent and service status from Cortix",
	Long:  `Queries the Cortix kernel for active services, agent status, and recent execution packets.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		cfgPath, _ := config.Find(cwd)
		var cortixURL string

		if cfgPath != "" {
			cfg, err := config.Load(cfgPath)
			if err == nil && cfg.Operator.CortixURL != "" {
				cortixURL = cfg.Operator.CortixURL
			}
		}
		if cortixURL == "" {
			cortixURL = os.Getenv("CORTIX_URL")
		}
		if cortixURL == "" {
			cortixURL = "https://command.warrencommand.dev"
		}

		// Query Cortix health endpoint
		client := &http.Client{Timeout: 5 * time.Second}

		fmt.Println("\033[1;34m⬡ Agent Status\033[0m")
		fmt.Println()

		// Health check
		healthURL := cortixURL + "/api/health"
		resp, err := client.Get(healthURL)
		if err != nil {
			fmt.Printf("\033[31m✗ Cortix unreachable\033[0m at %s\n", cortixURL)
			fmt.Printf("  Error: %v\n", err)
			return nil
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var health map[string]interface{}
		if err := json.Unmarshal(body, &health); err == nil {
			status, _ := health["status"].(string)
			if status == "ok" || status == "healthy" {
				fmt.Printf("\033[32m✓ Cortix kernel\033[0m  %s\n", cortixURL)
			} else {
				fmt.Printf("\033[33m⚠ Cortix kernel\033[0m  %s (status: %s)\n", cortixURL, status)
			}

			if services, ok := health["services"].(map[string]interface{}); ok {
				fmt.Println()
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "SERVICE\tSTATUS")
				fmt.Fprintln(w, "───────\t──────")
				for svc, st := range services {
					s, _ := st.(string)
					icon := "\033[32m●\033[0m"
					if s != "active" && s != "ok" && s != "running" {
						icon = "\033[31m●\033[0m"
					}
					fmt.Fprintf(w, "%s %s\t%s\n", icon, svc, s)
				}
				w.Flush()
			}

			// Show additional fields if present
			if uptime, ok := health["uptime"].(string); ok {
				fmt.Printf("\n  Uptime: %s\n", uptime)
			}
			if version, ok := health["version"].(string); ok {
				fmt.Printf("  Version: %s\n", version)
			}
		} else {
			fmt.Printf("  Raw: %s\n", string(body))
		}

		// Try to get recent packets/intents
		packetsURL := cortixURL + "/api/packets?limit=5"
		req, _ := http.NewRequest("GET", packetsURL, nil)
		if apiKey := os.Getenv("CORTIX_API_KEY"); apiKey != "" {
			req.Header.Set("X-API-Key", apiKey)
		}
		presp, err := client.Do(req)
		if err == nil && presp.StatusCode == 200 {
			defer presp.Body.Close()
			pbody, _ := io.ReadAll(presp.Body)
			var packets []map[string]interface{}
			if json.Unmarshal(pbody, &packets) == nil && len(packets) > 0 {
				fmt.Println()
				fmt.Println("\033[1;34mRecent Packets\033[0m")
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "ID\tTYPE\tSTATUS\tTIME")
				for _, p := range packets {
					id, _ := p["id"].(string)
					ptype, _ := p["type"].(string)
					pstatus, _ := p["status"].(string)
					pts, _ := p["created_at"].(string)
					if len(id) > 8 {
						id = id[:8]
					}
					if len(pts) > 16 {
						pts = pts[:16]
					}
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, ptype, pstatus, pts)
				}
				w.Flush()
			}
		} else if presp != nil {
			presp.Body.Close()
		}

		return nil
	},
}
