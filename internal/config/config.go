package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Project ProjectConfig `toml:"project"`
	Prompt  PromptConfig  `toml:"prompt"`
	Hooks   HooksConfig   `toml:"hooks"`
	Aliases map[string]string `toml:"aliases"`
}

type ProjectConfig struct {
	Name string `toml:"name"`
}

type PromptConfig struct {
	Segment string `toml:"segment"`
}

type HooksConfig struct {
	Git   map[string][]string `toml:"git"`
	Enter struct {
		Commands []string `toml:"commands"`
	} `toml:"enter"`
	Leave struct {
		Commands []string `toml:"commands"`
	} `toml:"leave"`
}

func Find(startDir string) (string, error) {
	current := startDir
	for {
		configPath := filepath.Join(current, ".wt.toml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", fmt.Errorf("no .wt.toml found in %s or its ancestors", startDir)
}

func Load(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &cfg, nil
}
