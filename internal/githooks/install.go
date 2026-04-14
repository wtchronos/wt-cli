package githooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Install() error {
	gitDir := ".git"
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository")
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	supportedHooks := []string{
		"pre-commit",
		"pre-push",
		"post-checkout",
		"post-commit",
		"post-merge",
	}

	for _, hook := range supportedHooks {
		hookPath := filepath.Join(hooksDir, hook)
		content := "#!/bin/sh\n" +
			"# Managed by wt - do not edit directly\n" +
			"exec wt hook run " + hook + " \"$@\"\n"

		// Check if hook already exists and isn't managed by wt
		if existing, err := os.ReadFile(hookPath); err == nil {
			if !isManagedByWT(string(existing)) {
				backupPath := hookPath + ".pre-wt"
				if err := os.WriteFile(backupPath, existing, 0755); err != nil {
					return fmt.Errorf("failed to backup existing hook: %w", err)
				}
			}
		}

		if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
			return fmt.Errorf("failed to write hook: %w", err)
		}
	}

	return nil
}

func isManagedByWT(content string) bool {
	return strings.Contains(content, "# Managed by wt")
}
