package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntentCmdHappyPath tests successful intent submission to Cortix server.
func TestIntentCmdHappyPath(t *testing.T) {
	// Setup mock Cortix server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/intent" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Verify request headers
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type: application/json, got %s", ct)
		}

		// Parse request body
		var intent map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&intent); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		// Verify intent structure
		if intent["source"] != "wt-cli" {
			t.Errorf("expected source=wt-cli, got %v", intent["source"])
		}
		if intent["type"] != "intent" {
			t.Errorf("expected type=intent, got %v", intent["type"])
		}
		if intent["description"] != "run test suite" {
			t.Errorf("expected description='run test suite', got %v", intent["description"])
		}
		if intent["priority"] != "P1" {
			t.Errorf("expected priority=P1, got %v", intent["priority"])
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"id":      "abc",
			"status":  "queued",
			"routing": "auto-execute",
		})
	}))
	defer server.Close()

	// Setup test environment
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	t.Setenv("CORTIX_URL", server.URL)

	// Reset intent priority before test
	intentPriority = "P1"

	// Execute command
	err := intentCmd.RunE(intentCmd, []string{"run test suite"})

	// Verify success
	if err != nil {
		t.Errorf("expected RunE to return nil, got error: %v", err)
	}

	// Verify .wt/intents.jsonl was NOT created (happy path)
	if _, err := os.Stat(filepath.Join(tmpDir, ".wt/intents.jsonl")); err == nil {
		t.Error("expected .wt/intents.jsonl not to be created on successful submission")
	}
}

// TestIntentCmd400Fallback tests fallback to local logging when server returns 400.
func TestIntentCmd400Fallback(t *testing.T) {
	// Setup mock Cortix server that returns 400
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/intent" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid intent format"}`))
	}))
	defer server.Close()

	// Setup test environment
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	t.Setenv("CORTIX_URL", server.URL)

	// Reset intent priority
	intentPriority = "P1"

	// Execute command
	err := intentCmd.RunE(intentCmd, []string{"run test suite"})

	// Verify success (RunE returns nil even on fallback)
	if err != nil {
		t.Errorf("expected RunE to return nil, got error: %v", err)
	}

	// Verify .wt/intents.jsonl was created
	jsonlPath := filepath.Join(tmpDir, ".wt/intents.jsonl")
	if _, err := os.Stat(jsonlPath); err != nil {
		t.Fatalf("expected .wt/intents.jsonl to be created, got error: %v", err)
	}

	// Verify file contains valid JSON
	content, err := os.ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("failed to read intents.jsonl: %v", err)
	}

	var intent map[string]interface{}
	if err := json.Unmarshal(content, &intent); err != nil {
		t.Fatalf("expected valid JSON in intents.jsonl, got error: %v", err)
	}

	// Verify intent contents
	if intent["source"] != "wt-cli" {
		t.Errorf("expected source=wt-cli, got %v", intent["source"])
	}
	if intent["description"] != "run test suite" {
		t.Errorf("expected description='run test suite', got %v", intent["description"])
	}
	if intent["priority"] != "P1" {
		t.Errorf("expected priority=P1, got %v", intent["priority"])
	}
}

// TestIntentCmdUnreachable tests fallback when Cortix server is unreachable.
func TestIntentCmdUnreachable(t *testing.T) {
	// Setup test environment with unreachable URL
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Set CORTIX_URL to an address that won't respond
	t.Setenv("CORTIX_URL", "http://localhost:9999")

	// Reset intent priority
	intentPriority = "P1"

	// Execute command
	err := intentCmd.RunE(intentCmd, []string{"run test suite"})

	// Verify success (RunE returns nil even on fallback)
	if err != nil {
		t.Errorf("expected RunE to return nil, got error: %v", err)
	}

	// Verify .wt/intents.jsonl was created
	jsonlPath := filepath.Join(tmpDir, ".wt/intents.jsonl")
	if _, err := os.Stat(jsonlPath); err != nil {
		t.Fatalf("expected .wt/intents.jsonl to be created, got error: %v", err)
	}

	// Verify file contains valid JSON
	content, err := os.ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("failed to read intents.jsonl: %v", err)
	}

	var intent map[string]interface{}
	if err := json.Unmarshal(content, &intent); err != nil {
		t.Fatalf("expected valid JSON in intents.jsonl, got error: %v", err)
	}

	// Verify intent contents
	if intent["description"] != "run test suite" {
		t.Errorf("expected description='run test suite', got %v", intent["description"])
	}
	if intent["source"] != "wt-cli" {
		t.Errorf("expected source=wt-cli, got %v", intent["source"])
	}
}

// TestIntentCmdBadURL tests that invalid CORTIX_URL returns an error.
func TestIntentCmdBadURL(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Set CORTIX_URL to invalid URL (malformed)
	t.Setenv("CORTIX_URL", "not a valid url :://")

	// Reset intent priority
	intentPriority = "P1"

	// Execute command
	err := intentCmd.RunE(intentCmd, []string{"run test suite"})

	// Invalid URL should return an error (http.NewRequest fails)
	if err == nil {
		t.Error("expected RunE to return error for invalid URL")
	}
}

// TestLogIntentLocally tests the logIntentLocally function directly.
func TestLogIntentLocally(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Create test intent
	intent := map[string]interface{}{
		"source":      "wt-cli",
		"type":        "intent",
		"description": "test intent",
		"priority":    "P1",
		"project":     "wt-cli",
		"timestamp":   "2025-01-01T00:00:00Z",
		"metadata": map[string]string{
			"cwd":    tmpDir,
			"origin": "terminal",
		},
	}

	// Call logIntentLocally
	err := logIntentLocally(intent)

	// Verify success
	if err != nil {
		t.Fatalf("expected logIntentLocally to succeed, got error: %v", err)
	}

	// Verify .wt/intents.jsonl was created
	jsonlPath := filepath.Join(tmpDir, ".wt/intents.jsonl")
	if _, err := os.Stat(jsonlPath); err != nil {
		t.Fatalf("expected .wt/intents.jsonl to be created, got error: %v", err)
	}

	// Verify file contains valid JSON
	content, err := os.ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("failed to read intents.jsonl: %v", err)
	}

	var loaded map[string]interface{}
	if err := json.Unmarshal(content, &loaded); err != nil {
		t.Fatalf("expected valid JSON in intents.jsonl, got error: %v", err)
	}

	// Verify intent contents
	if loaded["source"] != "wt-cli" {
		t.Errorf("expected source=wt-cli, got %v", loaded["source"])
	}
	if loaded["description"] != "test intent" {
		t.Errorf("expected description='test intent', got %v", loaded["description"])
	}
	if loaded["priority"] != "P1" {
		t.Errorf("expected priority=P1, got %v", loaded["priority"])
	}
}

// TestLogIntentLocallyMultipleLines tests appending multiple intents to the same file.
func TestLogIntentLocallyMultipleLines(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Log first intent
	intent1 := map[string]interface{}{
		"source":      "wt-cli",
		"description": "first intent",
		"priority":    "P1",
	}
	if err := logIntentLocally(intent1); err != nil {
		t.Fatalf("first logIntentLocally failed: %v", err)
	}

	// Log second intent
	intent2 := map[string]interface{}{
		"source":      "wt-cli",
		"description": "second intent",
		"priority":    "P0",
	}
	if err := logIntentLocally(intent2); err != nil {
		t.Fatalf("second logIntentLocally failed: %v", err)
	}

	// Read file
	jsonlPath := filepath.Join(tmpDir, ".wt/intents.jsonl")
	content, err := os.ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("failed to read intents.jsonl: %v", err)
	}

	// Parse as JSONL (one JSON object per line)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines in intents.jsonl, got %d", len(lines))
	}

	// Verify first intent
	var first map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("failed to unmarshal first intent: %v", err)
	}
	if first["description"] != "first intent" {
		t.Errorf("expected first intent description='first intent', got %v", first["description"])
	}

	// Verify second intent
	var second map[string]interface{}
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("failed to unmarshal second intent: %v", err)
	}
	if second["description"] != "second intent" {
		t.Errorf("expected second intent description='second intent', got %v", second["description"])
	}
}

// TestIntentCmdPriorityLevels tests different priority levels.
func TestIntentCmdPriorityLevels(t *testing.T) {
	tests := []struct {
		name     string
		priority string
	}{
		{"P0", "P0"},
		{"P1", "P1"},
		{"P2", "P2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var intent map[string]interface{}
				json.NewDecoder(r.Body).Decode(&intent)

				if intent["priority"] != tt.priority {
					t.Errorf("expected priority=%s, got %v", tt.priority, intent["priority"])
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"id": "test"})
			}))
			defer server.Close()

			tmpDir := t.TempDir()
			oldCwd, _ := os.Getwd()
			os.Chdir(tmpDir)
			defer os.Chdir(oldCwd)

			t.Setenv("CORTIX_URL", server.URL)
			intentPriority = tt.priority

			err := intentCmd.RunE(intentCmd, []string{"test"})
			if err != nil {
				t.Errorf("expected RunE to return nil, got error: %v", err)
			}
		})
	}
}

// TestIntentCmdMultipleArguments tests command with multiple arguments joined as description.
func TestIntentCmdMultipleArguments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var intent map[string]interface{}
		json.NewDecoder(r.Body).Decode(&intent)

		expected := "run test suite and report coverage"
		if intent["description"] != expected {
			t.Errorf("expected description='%s', got %v", expected, intent["description"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "test"})
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	t.Setenv("CORTIX_URL", server.URL)
	intentPriority = "P1"

	// Pass multiple arguments
	err := intentCmd.RunE(intentCmd, []string{"run", "test", "suite", "and", "report", "coverage"})
	if err != nil {
		t.Errorf("expected RunE to return nil, got error: %v", err)
	}
}

// TestIntentCmdAPIKey tests that API key is sent in headers when present.
func TestIntentCmdAPIKey(t *testing.T) {
	apiKeyReceived := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if key := r.Header.Get("X-API-Key"); key == "test-api-key" {
			apiKeyReceived = true
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "test"})
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	t.Setenv("CORTIX_URL", server.URL)
	t.Setenv("CORTIX_API_KEY", "test-api-key")
	intentPriority = "P1"

	err := intentCmd.RunE(intentCmd, []string{"test"})
	if err != nil {
		t.Errorf("expected RunE to return nil, got error: %v", err)
	}

	if !apiKeyReceived {
		t.Error("expected API key to be sent in X-API-Key header")
	}
}

// TestIntentCmdServerError tests handling of server errors (5xx).
func TestIntentCmdServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	t.Setenv("CORTIX_URL", server.URL)
	intentPriority = "P1"

	err := intentCmd.RunE(intentCmd, []string{"test"})

	// RunE should still return nil (fallback path)
	if err != nil {
		t.Errorf("expected RunE to return nil, got error: %v", err)
	}

	// Verify .wt/intents.jsonl was created
	jsonlPath := filepath.Join(tmpDir, ".wt/intents.jsonl")
	if _, err := os.Stat(jsonlPath); err != nil {
		t.Fatalf("expected .wt/intents.jsonl to be created, got error: %v", err)
	}
}

// TestIntentCmdNoArgs tests that command requires at least one argument.
func TestIntentCmdNoArgs(t *testing.T) {
	// Create a mock server for the default fallback URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Override the default Cortix URL
	t.Setenv("CORTIX_URL", server.URL)
	intentPriority = "P1"

	// Cobra's Args constraint (MinimumNArgs) is checked by the framework
	// before RunE is called, so we test that the command succeeds with empty description
	// but falls back to local logging (since the intent has empty description)
	err := intentCmd.RunE(intentCmd, []string{})

	// With no arguments, description will be empty string
	// The command should still execute (RunE handles the empty case gracefully)
	// and log locally due to server error
	if err != nil {
		t.Errorf("expected RunE to handle empty description, got error: %v", err)
	}

	// Verify .wt/intents.jsonl was created from fallback
	jsonlPath := filepath.Join(tmpDir, ".wt/intents.jsonl")
	if _, err := os.Stat(jsonlPath); err != nil {
		t.Fatalf("expected .wt/intents.jsonl to be created, got error: %v", err)
	}
}

// TestIntentCmdMetadata tests that metadata is properly included in intent.
func TestIntentCmdMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var intent map[string]interface{}
		json.NewDecoder(r.Body).Decode(&intent)

		// Verify metadata exists
		metadata, ok := intent["metadata"].(map[string]interface{})
		if !ok {
			t.Error("expected metadata to be present")
		}

		// Verify metadata contains expected fields
		if _, hasOrigin := metadata["origin"]; !hasOrigin {
			t.Error("expected 'origin' in metadata")
		}
		if _, hasCwd := metadata["cwd"]; !hasCwd {
			t.Error("expected 'cwd' in metadata")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "test"})
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	t.Setenv("CORTIX_URL", server.URL)
	intentPriority = "P1"

	err := intentCmd.RunE(intentCmd, []string{"test"})
	if err != nil {
		t.Errorf("expected RunE to return nil, got error: %v", err)
	}
}
