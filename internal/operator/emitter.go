package operator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Event represents a project-level event emitted to the operator surface.
type Event struct {
	Source    string            `json:"source"`
	Type     string            `json:"type"`
	Project  string            `json:"project"`
	Message  string            `json:"message"`
	Tags     []string          `json:"tags,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Ts       string            `json:"ts"`
}

// Emitter sends events to the operator surface (Cortix kernel).
type Emitter struct {
	CortixURL string
	APIKey    string
	Project   string
	Tags      []string
	Verbose   bool
}

// Emit sends an event to the Cortix kernel /api/events endpoint.
// Falls back to writing to a local event log if the endpoint is unreachable.
func (e *Emitter) Emit(eventType, message string, metadata map[string]string) error {
	ev := Event{
		Source:   "wt-cli",
		Type:     eventType,
		Project:  e.Project,
		Message:  message,
		Tags:     e.Tags,
		Metadata: metadata,
		Ts:       time.Now().UTC().Format(time.RFC3339),
	}

	if e.CortixURL == "" {
		return e.logLocal(ev)
	}

	body, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	url := e.CortixURL + "/api/events"
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if e.APIKey != "" {
		req.Header.Set("X-API-Key", e.APIKey)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		if e.Verbose {
			fmt.Fprintf(os.Stderr, "[wt] cortix unreachable, logging locally: %v\n", err)
		}
		return e.logLocal(ev)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		if e.Verbose {
			fmt.Fprintf(os.Stderr, "[wt] cortix returned %d, logging locally\n", resp.StatusCode)
		}
		return e.logLocal(ev)
	}

	if e.Verbose {
		fmt.Fprintf(os.Stderr, "[wt] event emitted: %s/%s\n", eventType, e.Project)
	}
	return nil
}

// logLocal writes the event to .wt/events.jsonl as a fallback.
func (e *Emitter) logLocal(ev Event) error {
	dir := ".wt"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(dir+"/events.jsonl", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	line, _ := json.Marshal(ev)
	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}
