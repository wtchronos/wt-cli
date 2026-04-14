package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Runner struct {
	ProjectName string
	ProjectRoot string
	Verbose     bool
}

func (r *Runner) Run(event string, commands []string) error {
	if os.Getenv("WT_IN_HOOK") == "1" {
		return nil
	}

	os.Setenv("WT_IN_HOOK", "1")
	defer os.Unsetenv("WT_IN_HOOK")

	os.Setenv("WT_PROJECT_ROOT", r.ProjectRoot)
	os.Setenv("WT_PROJECT_NAME", r.ProjectName)
	os.Setenv("WT_EVENT", event)

	for _, cmdStr := range commands {
		if cmdStr == "" {
			continue
		}

		// Expand template variables
		cmdStr = strings.ReplaceAll(cmdStr, "{{.Project.Name}}", r.ProjectName)
		cmdStr = strings.ReplaceAll(cmdStr, "{{.Project.Root}}", r.ProjectRoot)

		if r.Verbose {
			fmt.Fprintf(os.Stderr, "[wt hook/%s] %s\n", event, cmdStr)
		}

		cmd := exec.Command("sh", "-c", cmdStr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = r.ProjectRoot
		cmd.Env = os.Environ()

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("hook %s command failed: %w", event, err)
		}
	}

	return nil
}
