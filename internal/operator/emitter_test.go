package operator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEmitLocalFallback(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	e := &Emitter{
		Project: "test-project",
		Tags:    []string{"test"},
	}

	err := e.Emit("build", "test message", map[string]string{"key": "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check local log was written
	data, err := os.ReadFile(filepath.Join(dir, ".wt", "events.jsonl"))
	if err != nil {
		t.Fatalf("expected event log: %v", err)
	}

	var ev Event
	if err := json.Unmarshal(data[:len(data)-1], &ev); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if ev.Source != "wt-cli" {
		t.Fatalf("expected source 'wt-cli', got '%s'", ev.Source)
	}
	if ev.Type != "build" {
		t.Fatalf("expected type 'build', got '%s'", ev.Type)
	}
	if ev.Project != "test-project" {
		t.Fatalf("expected project 'test-project', got '%s'", ev.Project)
	}
	if ev.Metadata["key"] != "val" {
		t.Fatalf("metadata not preserved")
	}
}

func TestEmitUnreachableURL(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	e := &Emitter{
		CortixURL: "http://127.0.0.1:1", // port 1 = unreachable
		Project:   "test",
	}

	// Should fall back to local log, not error
	err := e.Emit("test", "unreachable", nil)
	if err != nil {
		t.Fatalf("should fallback, not error: %v", err)
	}

	// Verify local fallback worked
	_, err = os.ReadFile(filepath.Join(dir, ".wt", "events.jsonl"))
	if err != nil {
		t.Fatal("expected local fallback log")
	}
}

func TestEmitMultipleAppends(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	e := &Emitter{Project: "multi"}

	e.Emit("a", "first", nil)
	e.Emit("b", "second", nil)

	data, _ := os.ReadFile(filepath.Join(dir, ".wt", "events.jsonl"))
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if lines != 2 {
		t.Fatalf("expected 2 lines, got %d", lines)
	}
}
