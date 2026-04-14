package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wtchronos/wt-cli/internal/config"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check project and operator health",
	Run: func(cmd *cobra.Command, args []string) {
		cfgPath, err := config.Find(".")
		if err != nil {
			fmt.Println("Not in a wt project")
			os.Exit(1)
		}

		cfg, err := config.Load(cfgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
			os.Exit(1)
		}

		projectRoot := filepath.Dir(cfgPath)
		allOK := true

		// 1. Config valid
		fmt.Printf("Config:    %s\n", cfgPath)

		// 2. Git state
		gitOK := checkGitHealth(projectRoot)
		if !gitOK {
			allOK = false
		}

		// 3. Hook installation
		hookOK := checkHookInstallation(projectRoot)
		if !hookOK {
			allOK = false
		}

		// 4. Operator surface (if configured)
		if cfg.Operator.CortixURL != "" {
			operatorOK := checkOperatorHealth(cfg.Operator.CortixURL, cfg.Operator.APIKey)
			if !operatorOK {
				allOK = false
			}
		}

		// Final verdict
		fmt.Println()
		if allOK {
			fmt.Println("Health: OK")
		} else {
			fmt.Println("Health: DEGRADED")
			os.Exit(1)
		}
	},
}

func checkGitHealth(root string) bool {
	ok := true

	// Check if git repo
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		fmt.Println("Git:       not a repository")
		return false
	}

	// Uncommitted changes
	out, _ := exec.Command("git", "-C", root, "status", "--porcelain").Output()
	status := strings.TrimSpace(string(out))
	if status != "" {
		lines := strings.Split(status, "\n")
		fmt.Printf("Git:       %d uncommitted changes\n", len(lines))
	} else {
		fmt.Println("Git:       clean")
	}

	// Unpushed commits
	out, err := exec.Command("git", "-C", root, "rev-list", "--count", "@{u}..HEAD").Output()
	if err == nil {
		count := strings.TrimSpace(string(out))
		if count != "0" {
			fmt.Printf("Unpushed:  %s commits ahead\n", count)
			ok = false
		} else {
			fmt.Println("Unpushed:  up to date")
		}
	}

	return ok
}

func checkHookInstallation(root string) bool {
	hooksDir := filepath.Join(root, ".git", "hooks")
	required := []string{"pre-commit", "post-checkout", "post-merge"}
	missing := []string{}

	for _, hook := range required {
		path := filepath.Join(hooksDir, hook)
		data, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(data), "Managed by wt") {
			missing = append(missing, hook)
		}
	}

	if len(missing) > 0 {
		fmt.Printf("Hooks:     missing: %s (run wt init)\n", strings.Join(missing, ", "))
		return false
	}
	fmt.Println("Hooks:     installed")
	return true
}

func checkOperatorHealth(url, apiKey string) bool {
	healthURL := url + "/health"
	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		fmt.Printf("Operator:  error: %v\n", err)
		return false
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Operator:  unreachable (%s)\n", url)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		fmt.Printf("Operator:  unhealthy (HTTP %d)\n", resp.StatusCode)
		return false
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err == nil {
		if status, ok := body["status"].(string); ok {
			fmt.Printf("Operator:  %s (%s)\n", status, url)
			return status == "ok" || status == "healthy"
		}
	}

	fmt.Printf("Operator:  reachable (%s)\n", url)
	return true
}

func init() {
	rootCmd.AddCommand(healthCmd)
}
