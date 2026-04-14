package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindInCurrentDir(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".wt.toml")
	os.WriteFile(configPath, []byte("[project]\nname = \"test\"\n"), 0644)

	found, err := Find(dir)
	if err != nil {
		t.Fatalf("expected to find config, got error: %v", err)
	}
	if found != configPath {
		t.Fatalf("expected %s, got %s", configPath, found)
	}
}

func TestFindInAncestorDir(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, ".wt.toml"), []byte("[project]\nname = \"root\"\n"), 0644)

	child := filepath.Join(root, "sub", "deep")
	os.MkdirAll(child, 0755)

	found, err := Find(child)
	if err != nil {
		t.Fatalf("expected to find config in ancestor, got error: %v", err)
	}
	if found != filepath.Join(root, ".wt.toml") {
		t.Fatalf("expected root config, got %s", found)
	}
}

func TestFindNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := Find(dir)
	if err == nil {
		t.Fatal("expected error when no config exists")
	}
}

func TestLoadValid(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".wt.toml")
	content := `[project]
name = "testproj"

[prompt]
segment = "[{{.Project.Name}}]"

[aliases]
test = "go test ./..."

[env]
GO_ENV = "test"

[scripts]
build = "go build ."
`
	os.WriteFile(configPath, []byte(content), 0644)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Project.Name != "testproj" {
		t.Fatalf("expected name 'testproj', got '%s'", cfg.Project.Name)
	}
	if cfg.Aliases["test"] != "go test ./..." {
		t.Fatalf("alias not parsed")
	}
	if cfg.Env["GO_ENV"] != "test" {
		t.Fatalf("env not parsed")
	}
	if cfg.Scripts["build"] != "go build ." {
		t.Fatalf("scripts not parsed")
	}
}

func TestLoadInvalid(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".wt.toml")
	os.WriteFile(configPath, []byte("not valid toml {{{{"), 0644)

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("expected error for invalid toml")
	}
}

func TestLoadWithOperator(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".wt.toml")
	content := `[project]
name = "ops"

[operator]
cortix_url = "https://command.warrencommand.dev"
api_key = "test-key"
tags = ["active", "kairos"]
`
	os.WriteFile(configPath, []byte(content), 0644)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Operator.CortixURL != "https://command.warrencommand.dev" {
		t.Fatalf("operator url not parsed")
	}
	if cfg.Operator.APIKey != "test-key" {
		t.Fatalf("operator api key not parsed")
	}
	if len(cfg.Operator.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(cfg.Operator.Tags))
	}
}
