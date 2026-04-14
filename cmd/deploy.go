package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wtchronos/wt-cli/internal/config"
	"github.com/wtchronos/wt-cli/internal/operator"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [target]",
	Short: "Deploy the current project to VPS",
	Long: `Run the project's deploy script and emit a deploy event to the operator surface.

The deploy script is read from [scripts] deploy key in .wt.toml.
If no target is specified, uses the default deploy script.
If a target is specified, looks for deploy:<target> in scripts.

Examples:
  wt deploy           # runs [scripts] deploy
  wt deploy staging   # runs [scripts] deploy:staging`,
	Args: cobra.MaximumNArgs(1),
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

		projectRoot := filepath.Dir(cfgPath)

		// Determine deploy script
		scriptKey := "deploy"
		if len(args) > 0 {
			scriptKey = "deploy:" + args[0]
		}

		deployScript, ok := cfg.Scripts[scriptKey]
		if !ok {
			fmt.Fprintf(os.Stderr, "No '%s' script defined in .wt.toml [scripts]\n", scriptKey)
			os.Exit(1)
		}

		// Get current git info for the deploy event
		branch := strings.TrimSpace(runQuietDeploy(projectRoot, "git", "rev-parse", "--abbrev-ref", "HEAD"))
		commitHash := strings.TrimSpace(runQuietDeploy(projectRoot, "git", "rev-parse", "--short", "HEAD"))

		// Emit pre-deploy event
		emitter := makeEmitter(cfg)
		emitter.Emit("deploy:start", fmt.Sprintf("deploying %s@%s", branch, commitHash), map[string]string{
			"branch": branch,
			"commit": commitHash,
			"target": scriptKey,
		})

		fmt.Printf("Deploying %s (%s@%s)...\n", cfg.Project.Name, branch, commitHash)

		// Inject project env
		environ := os.Environ()
		environ = append(environ, "WT_PROJECT_NAME="+cfg.Project.Name)
		environ = append(environ, "WT_PROJECT_ROOT="+projectRoot)
		environ = append(environ, "WT_DEPLOY_BRANCH="+branch)
		environ = append(environ, "WT_DEPLOY_COMMIT="+commitHash)
		for k, v := range cfg.Env {
			environ = append(environ, k+"="+v)
		}

		start := time.Now()
		c := exec.Command("sh", "-c", deployScript)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin
		c.Dir = projectRoot
		c.Env = environ

		err = c.Run()
		duration := time.Since(start)

		if err != nil {
			emitter.Emit("deploy:fail", fmt.Sprintf("deploy failed after %s: %v", duration.Round(time.Second), err), map[string]string{
				"branch":   branch,
				"commit":   commitHash,
				"duration": duration.Round(time.Second).String(),
				"error":    err.Error(),
			})
			fmt.Fprintf(os.Stderr, "\nDeploy FAILED after %s\n", duration.Round(time.Second))
			os.Exit(1)
		}

		emitter.Emit("deploy:success", fmt.Sprintf("deployed %s@%s in %s", branch, commitHash, duration.Round(time.Second)), map[string]string{
			"branch":   branch,
			"commit":   commitHash,
			"duration": duration.Round(time.Second).String(),
		})

		fmt.Printf("\nDeploy OK (%s)\n", duration.Round(time.Second))
	},
}

func makeEmitter(cfg *config.Config) *operator.Emitter {
	return &operator.Emitter{
		CortixURL: cfg.Operator.CortixURL,
		APIKey:    cfg.Operator.APIKey,
		Project:   cfg.Project.Name,
		Tags:      cfg.Operator.Tags,
		Verbose:   verbose,
	}
}

func runQuietDeploy(dir string, name string, a ...string) string {
	cmd := exec.Command(name, a...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
