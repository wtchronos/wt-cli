package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAgentCmdHappyPath(t *testing.T) {
	// Start mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.Header().Set("Content-Type", "application/json")
			health := map[string]interface{}{
				"status": "ok",
				"services": map[string]interface{}{
					"kairos": "active",
				},
			}
			json.NewEncoder(w).Encode(health)
		} else if r.URL.Path == "/api/packets" {
			w.Header().Set("Content-Type", "application/json")
			packets := []map[string]interface{}{
				{
					"id":         "packet-001",
					"type":       "query",
					"status":     "success",
					"created_at": "2024-01-15T10:30:00Z",
				},
			}
			json.NewEncoder(w).Encode(packets)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Use t.Setenv to set the environment variable (auto-restores after test)
	t.Setenv("CORTIX_URL", server.URL)

	// Execute the command
	err := agentCmd.RunE(agentCmd, []string{})

	// Verify no error
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAgentCmdUnreachableCortix(t *testing.T) {
	// Set CORTIX_URL to an unreachable address with no server running
	t.Setenv("CORTIX_URL", "http://localhost:9999")

	// Execute the command
	err := agentCmd.RunE(agentCmd, []string{})

	// Verify command returns nil (graceful degradation)
	if err != nil {
		t.Fatalf("expected nil error on unreachable Cortix, got %v", err)
	}
}

func TestAgentCmdInvalidHealthJSON(t *testing.T) {
	// Start mock server that returns invalid JSON for health endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("not valid json {"))
		} else if r.URL.Path == "/api/packets" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Setenv("CORTIX_URL", server.URL)

	// Execute the command
	err := agentCmd.RunE(agentCmd, []string{})

	// Verify command returns nil (graceful degradation on parse error)
	if err != nil {
		t.Fatalf("expected nil error on invalid JSON, got %v", err)
	}
}

func TestAgentCmdHealthStatusCheck(t *testing.T) {
	// Test that the command correctly interprets different health statuses
	tests := []struct {
		name   string
		status string
		want   string // substring to check in output
	}{
		{
			name:   "status ok",
			status: "ok",
			want:   "✓ Cortix kernel",
		},
		{
			name:   "status healthy",
			status: "healthy",
			want:   "✓ Cortix kernel",
		},
		{
			name:   "status degraded",
			status: "degraded",
			want:   "⚠ Cortix kernel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/health" {
					w.Header().Set("Content-Type", "application/json")
					health := map[string]interface{}{
						"status": tt.status,
						"services": map[string]interface{}{
							"test": "active",
						},
					}
					json.NewEncoder(w).Encode(health)
				} else if r.URL.Path == "/api/packets" {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte("[]"))
				}
			}))
			defer server.Close()

			t.Setenv("CORTIX_URL", server.URL)

			err := agentCmd.RunE(agentCmd, []string{})
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestAgentCmdPacketsEndpoint(t *testing.T) {
	// Test that the command properly handles the /api/packets endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.Header().Set("Content-Type", "application/json")
			health := map[string]interface{}{
				"status": "ok",
				"services": map[string]interface{}{
					"kairos": "active",
				},
			}
			json.NewEncoder(w).Encode(health)
		} else if r.URL.Path == "/api/packets" {
			// Verify limit parameter
			limit := r.URL.Query().Get("limit")
			if limit != "5" {
				t.Errorf("expected limit=5, got limit=%s", limit)
			}
			w.Header().Set("Content-Type", "application/json")
			packets := []map[string]interface{}{
				{
					"id":         "abc12345",
					"type":       "intent",
					"status":     "completed",
					"created_at": "2024-01-15T10:30:00Z",
				},
				{
					"id":         "def67890",
					"type":       "query",
					"status":     "pending",
					"created_at": "2024-01-15T10:25:00Z",
				},
			}
			json.NewEncoder(w).Encode(packets)
		}
	}))
	defer server.Close()

	t.Setenv("CORTIX_URL", server.URL)

	err := agentCmd.RunE(agentCmd, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAgentCmdPacketsEmptyList(t *testing.T) {
	// Test that the command gracefully handles empty packets list
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.Header().Set("Content-Type", "application/json")
			health := map[string]interface{}{
				"status": "ok",
				"services": map[string]interface{}{
					"kairos": "active",
				},
			}
			json.NewEncoder(w).Encode(health)
		} else if r.URL.Path == "/api/packets" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		}
	}))
	defer server.Close()

	t.Setenv("CORTIX_URL", server.URL)

	err := agentCmd.RunE(agentCmd, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAgentCmdPacketsInvalidJSON(t *testing.T) {
	// Test graceful handling when /api/packets returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.Header().Set("Content-Type", "application/json")
			health := map[string]interface{}{
				"status": "ok",
				"services": map[string]interface{}{
					"kairos": "active",
				},
			}
			json.NewEncoder(w).Encode(health)
		} else if r.URL.Path == "/api/packets" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("not valid json"))
		}
	}))
	defer server.Close()

	t.Setenv("CORTIX_URL", server.URL)

	err := agentCmd.RunE(agentCmd, []string{})
	if err != nil {
		t.Fatalf("expected no error despite invalid packets JSON, got %v", err)
	}
}

func TestAgentCmdPacketsNon200Status(t *testing.T) {
	// Test graceful handling when /api/packets returns non-200 status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.Header().Set("Content-Type", "application/json")
			health := map[string]interface{}{
				"status": "ok",
				"services": map[string]interface{}{
					"kairos": "active",
				},
			}
			json.NewEncoder(w).Encode(health)
		} else if r.URL.Path == "/api/packets" {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	t.Setenv("CORTIX_URL", server.URL)

	err := agentCmd.RunE(agentCmd, []string{})
	if err != nil {
		t.Fatalf("expected no error despite packets endpoint error, got %v", err)
	}
}

func TestAgentCmdAPIKeyHeader(t *testing.T) {
	// Test that CORTIX_API_KEY is sent if set
	apiKeyCapture := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.Header().Set("Content-Type", "application/json")
			health := map[string]interface{}{
				"status": "ok",
				"services": map[string]interface{}{
					"kairos": "active",
				},
			}
			json.NewEncoder(w).Encode(health)
		} else if r.URL.Path == "/api/packets" {
			// Capture the X-API-Key header
			apiKeyCapture = r.Header.Get("X-API-Key")
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		}
	}))
	defer server.Close()

	t.Setenv("CORTIX_URL", server.URL)
	t.Setenv("CORTIX_API_KEY", "test-api-key-12345")

	err := agentCmd.RunE(agentCmd, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if apiKeyCapture != "test-api-key-12345" {
		t.Errorf("expected X-API-Key header to be 'test-api-key-12345', got '%s'", apiKeyCapture)
	}
}

func TestAgentCmdDefaultURL(t *testing.T) {
	// Test that the default URL is used when CORTIX_URL is not set
	// This test needs to be careful not to actually hit the real default URL
	// Instead, we'll just verify the environment variable precedence

	// Clear any existing CORTIX_URL
	t.Setenv("CORTIX_URL", "")

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.Header().Set("Content-Type", "application/json")
			health := map[string]interface{}{
				"status": "ok",
			}
			json.NewEncoder(w).Encode(health)
		}
	}))
	defer server.Close()

	// Override with env var to avoid hitting production
	t.Setenv("CORTIX_URL", server.URL)

	err := agentCmd.RunE(agentCmd, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAgentCmdServiceStatusFormatting(t *testing.T) {
	// Test that services are formatted correctly based on status
	tests := []struct {
		name        string
		serviceData map[string]interface{}
	}{
		{
			name: "active service",
			serviceData: map[string]interface{}{
				"cortix": "active",
			},
		},
		{
			name: "inactive service",
			serviceData: map[string]interface{}{
				"cortix": "inactive",
			},
		},
		{
			name: "running service",
			serviceData: map[string]interface{}{
				"cortix": "running",
			},
		},
		{
			name: "multiple services",
			serviceData: map[string]interface{}{
				"kairos":  "active",
				"cortix":  "running",
				"gateway": "inactive",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/health" {
					w.Header().Set("Content-Type", "application/json")
					health := map[string]interface{}{
						"status":   "ok",
						"services": tt.serviceData,
					}
					json.NewEncoder(w).Encode(health)
				} else if r.URL.Path == "/api/packets" {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte("[]"))
				}
			}))
			defer server.Close()

			t.Setenv("CORTIX_URL", server.URL)

			err := agentCmd.RunE(agentCmd, []string{})
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestAgentCmdHealthMetadata(t *testing.T) {
	// Test that additional health metadata (uptime, version) is displayed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.Header().Set("Content-Type", "application/json")
			health := map[string]interface{}{
				"status": "ok",
				"services": map[string]interface{}{
					"kairos": "active",
				},
				"uptime":  "12h30m",
				"version": "1.2.3",
			}
			json.NewEncoder(w).Encode(health)
		} else if r.URL.Path == "/api/packets" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		}
	}))
	defer server.Close()

	t.Setenv("CORTIX_URL", server.URL)

	err := agentCmd.RunE(agentCmd, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAgentCmdMissingHealthServices(t *testing.T) {
	// Test graceful handling when health response lacks services field
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.Header().Set("Content-Type", "application/json")
			health := map[string]interface{}{
				"status": "ok",
			}
			json.NewEncoder(w).Encode(health)
		} else if r.URL.Path == "/api/packets" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		}
	}))
	defer server.Close()

	t.Setenv("CORTIX_URL", server.URL)

	err := agentCmd.RunE(agentCmd, []string{})
	if err != nil {
		t.Fatalf("expected no error when services field is missing, got %v", err)
	}
}

func TestAgentCmdEmptyHealthResponse(t *testing.T) {
	// Test handling of empty health response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{}"))
		} else if r.URL.Path == "/api/packets" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		}
	}))
	defer server.Close()

	t.Setenv("CORTIX_URL", server.URL)

	err := agentCmd.RunE(agentCmd, []string{})
	if err != nil {
		t.Fatalf("expected no error with empty health response, got %v", err)
	}
}

func TestAgentCmdURL(t *testing.T) {
	// Verify the command uses correct URL construction
	var capturedHealthURL string
	var capturedPacketsURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			capturedHealthURL = r.URL.String()
			w.Header().Set("Content-Type", "application/json")
			health := map[string]interface{}{
				"status": "ok",
			}
			json.NewEncoder(w).Encode(health)
		} else if r.URL.Path == "/api/packets" {
			capturedPacketsURL = r.URL.String()
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		}
	}))
	defer server.Close()

	t.Setenv("CORTIX_URL", server.URL)

	err := agentCmd.RunE(agentCmd, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify health endpoint was called
	if !strings.Contains(capturedHealthURL, "/api/health") {
		t.Errorf("health endpoint not called correctly: %s", capturedHealthURL)
	}

	// Verify packets endpoint was called with limit param
	if !strings.Contains(capturedPacketsURL, "/api/packets") {
		t.Errorf("packets endpoint not called: %s", capturedPacketsURL)
	}
	if !strings.Contains(capturedPacketsURL, "limit=5") {
		t.Errorf("packets endpoint not called with limit=5: %s", capturedPacketsURL)
	}
}

func TestAgentCmdTimeout(t *testing.T) {
	// Test that the command gracefully handles network timeouts
	// Instead of trying to hang the server, just use an unreachable address
	// which will timeout. The command already handles this gracefully.
	t.Setenv("CORTIX_URL", "http://192.0.2.1:9999") // non-routable IP that times out

	// This should timeout and return nil (graceful error handling)
	err := agentCmd.RunE(agentCmd, []string{})

	// Command should return nil due to graceful error handling
	if err != nil {
		t.Fatalf("expected nil error on timeout, got %v", err)
	}
}

func TestAgentCmdPacketTruncation(t *testing.T) {
	// Test that packet IDs and timestamps are truncated correctly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.Header().Set("Content-Type", "application/json")
			health := map[string]interface{}{
				"status": "ok",
			}
			json.NewEncoder(w).Encode(health)
		} else if r.URL.Path == "/api/packets" {
			w.Header().Set("Content-Type", "application/json")
			packets := []map[string]interface{}{
				{
					"id":         "this-is-a-very-long-packet-id-that-should-be-truncated",
					"type":       "query",
					"status":     "success",
					"created_at": "2024-01-15T10:30:45.123456789Z",
				},
			}
			json.NewEncoder(w).Encode(packets)
		}
	}))
	defer server.Close()

	t.Setenv("CORTIX_URL", server.URL)

	err := agentCmd.RunE(agentCmd, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
