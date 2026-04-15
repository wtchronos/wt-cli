package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetLogFlags() {
	logLines = 10
	logFailures = false
	logSource = ""
}

func createTempLogsStructure(t *testing.T, tmpDir string) string {
	// Create ~/.nightagent-sync/ops/audits directory structure
	opsDir := filepath.Join(tmpDir, "nightagent-sync", "ops")
	auditsDir := filepath.Join(opsDir, "audits")
	wtDir := filepath.Join(tmpDir, ".wt")

	if err := os.MkdirAll(auditsDir, 0755); err != nil {
		t.Fatalf("failed to create audits dir: %v", err)
	}
	if err := os.MkdirAll(wtDir, 0755); err != nil {
		t.Fatalf("failed to create .wt dir: %v", err)
	}

	return tmpDir
}

func writeJSONLFile(t *testing.T, path string, lines []string) {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create parent dir: %v", err)
	}

	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func captureStdout(fn func() error) (string, error) {
	// Save original stdout
	oldStdout := os.Stdout

	// Create a pipe
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	// Redirect stdout to the write end of the pipe
	os.Stdout = w

	// Execute the function
	var execErr error
	execErr = fn()

	// Close the write end and restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return "", err
	}
	r.Close()

	return buf.String(), execErr
}

func TestLogCmd_NoFlags_ShowsAllEntries(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir = createTempLogsStructure(t, tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create ops-log.jsonl with 3 entries
	opsLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "ops-log.jsonl")
	opsLines := []string{
		`{"ts":"2026-04-14T10:00:00Z","type":"health","status":"ok","message":"system healthy"}`,
		`{"ts":"2026-04-14T10:05:00Z","type":"sync","status":"ok","message":"sync completed"}`,
		`{"ts":"2026-04-14T10:10:00Z","type":"check","status":"ok","message":"all checks passed"}`,
	}
	writeJSONLFile(t, opsLogPath, opsLines)

	resetLogFlags()

	output, err := captureStdout(func() error {
		return logCmd.RunE(logCmd, []string{})
	})

	if err != nil {
		t.Fatalf("logCmd.RunE failed: %v", err)
	}

	// Verify output contains all 3 entries
	if !strings.Contains(output, "system healthy") {
		t.Errorf("expected 'system healthy' in output, got: %s", output)
	}
	if !strings.Contains(output, "sync completed") {
		t.Errorf("expected 'sync completed' in output, got: %s", output)
	}
	if !strings.Contains(output, "all checks passed") {
		t.Errorf("expected 'all checks passed' in output, got: %s", output)
	}
	if strings.Contains(output, "No log entries found") {
		t.Errorf("should not show 'No log entries found' when entries exist: %s", output)
	}
}

func TestLogCmd_FailuresFlag_FiltersToFailures(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir = createTempLogsStructure(t, tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create ops-log.jsonl with mix of success and failure entries
	opsLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "ops-log.jsonl")
	opsLines := []string{
		`{"ts":"2026-04-14T10:00:00Z","type":"health","status":"ok","message":"system healthy"}`,
		`{"ts":"2026-04-14T10:05:00Z","type":"sync","status":"fail","message":"sync failed"}`,
		`{"ts":"2026-04-14T10:10:00Z","type":"check","status":"ok","message":"all checks passed"}`,
		`{"ts":"2026-04-14T10:15:00Z","type":"alert","error":"connection timeout","message":"error occurred"}`,
		`{"ts":"2026-04-14T10:20:00Z","type":"deploy","status":"degraded","message":"partial failure"}`,
	}
	writeJSONLFile(t, opsLogPath, opsLines)

	resetLogFlags()
	logFailures = true

	output, err := captureStdout(func() error {
		return logCmd.RunE(logCmd, []string{})
	})

	if err != nil {
		t.Fatalf("logCmd.RunE failed: %v", err)
	}

	// Should show failures
	if !strings.Contains(output, "sync failed") {
		t.Errorf("expected 'sync failed' in output, got: %s", output)
	}
	if !strings.Contains(output, "error occurred") {
		t.Errorf("expected 'error occurred' in output, got: %s", output)
	}
	if !strings.Contains(output, "partial failure") {
		t.Errorf("expected 'partial failure' in output, got: %s", output)
	}

	// Should NOT show successes
	if strings.Contains(output, "system healthy") {
		t.Errorf("should not show 'system healthy' when --failures set: %s", output)
	}
	if strings.Contains(output, "all checks passed") {
		t.Errorf("should not show 'all checks passed' when --failures set: %s", output)
	}
}

func TestLogCmd_FailuresFlag_ResultField(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir = createTempLogsStructure(t, tmpDir)
	t.Setenv("HOME", tmpDir)

	// Test with result field instead of status
	opsLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "ops-log.jsonl")
	opsLines := []string{
		`{"ts":"2026-04-14T10:00:00Z","type":"test","result":"pass","message":"test passed"}`,
		`{"ts":"2026-04-14T10:05:00Z","type":"test","result":"fail","message":"test failed"}`,
	}
	writeJSONLFile(t, opsLogPath, opsLines)

	resetLogFlags()
	logFailures = true

	output, err := captureStdout(func() error {
		return logCmd.RunE(logCmd, []string{})
	})

	if err != nil {
		t.Fatalf("logCmd.RunE failed: %v", err)
	}

	// Should show result=fail
	if !strings.Contains(output, "test failed") {
		t.Errorf("expected 'test failed' in output, got: %s", output)
	}
	// Should NOT show result=pass
	if strings.Contains(output, "test passed") {
		t.Errorf("should not show 'test passed' when --failures set: %s", output)
	}
}

func TestLogCmd_SourceOps_OnlyReadsOpsLog(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir = createTempLogsStructure(t, tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create multiple log files
	opsLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "ops-log.jsonl")
	opsLines := []string{
		`{"ts":"2026-04-14T10:00:00Z","type":"ops","message":"ops entry"}`,
	}
	writeJSONLFile(t, opsLogPath, opsLines)

	eventsLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "events.jsonl")
	eventsLines := []string{
		`{"ts":"2026-04-14T10:05:00Z","type":"event","message":"events entry"}`,
	}
	writeJSONLFile(t, eventsLogPath, eventsLines)

	auditLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "audits", "audit.jsonl")
	auditLines := []string{
		`{"ts":"2026-04-14T10:10:00Z","type":"audit","message":"audit entry"}`,
	}
	writeJSONLFile(t, auditLogPath, auditLines)

	resetLogFlags()
	logSource = "ops"

	output, err := captureStdout(func() error {
		return logCmd.RunE(logCmd, []string{})
	})

	if err != nil {
		t.Fatalf("logCmd.RunE failed: %v", err)
	}

	// Should only show ops entry
	if !strings.Contains(output, "ops entry") {
		t.Errorf("expected 'ops entry' in output, got: %s", output)
	}
	if strings.Contains(output, "events entry") {
		t.Errorf("should not show 'events entry' when --source ops: %s", output)
	}
	if strings.Contains(output, "audit entry") {
		t.Errorf("should not show 'audit entry' when --source ops: %s", output)
	}
}

func TestLogCmd_SourceEvents_OnlyReadsEventsLog(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir = createTempLogsStructure(t, tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create events log file
	eventsLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "events.jsonl")
	eventsLines := []string{
		`{"ts":"2026-04-14T10:05:00Z","type":"event","message":"events entry"}`,
	}
	writeJSONLFile(t, eventsLogPath, eventsLines)

	resetLogFlags()
	logSource = "events"

	output, err := captureStdout(func() error {
		return logCmd.RunE(logCmd, []string{})
	})

	if err != nil {
		t.Fatalf("logCmd.RunE failed: %v", err)
	}

	// Should show events entry
	if !strings.Contains(output, "events entry") {
		t.Errorf("expected 'events entry' in output, got: %s", output)
	}
}

func TestLogCmd_EmptyLogFiles_NoEntriesMessage(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir = createTempLogsStructure(t, tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create empty log files
	opsLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "ops-log.jsonl")
	writeJSONLFile(t, opsLogPath, []string{})

	eventsLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "events.jsonl")
	writeJSONLFile(t, eventsLogPath, []string{})

	auditLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "audits", "audit.jsonl")
	writeJSONLFile(t, auditLogPath, []string{})

	resetLogFlags()

	output, err := captureStdout(func() error {
		return logCmd.RunE(logCmd, []string{})
	})

	if err != nil {
		t.Fatalf("logCmd.RunE failed: %v", err)
	}

	// Should show "No log entries found" message
	if !strings.Contains(output, "No log entries found") {
		t.Errorf("expected 'No log entries found' in output, got: %s", output)
	}
}

func TestLogCmd_EmptyLogFiles_WithFailuresFlag(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir = createTempLogsStructure(t, tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create empty log files
	opsLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "ops-log.jsonl")
	writeJSONLFile(t, opsLogPath, []string{})

	resetLogFlags()
	logFailures = true

	output, err := captureStdout(func() error {
		return logCmd.RunE(logCmd, []string{})
	})

	if err != nil {
		t.Fatalf("logCmd.RunE failed: %v", err)
	}

	// Should show "No failures found" when --failures is set
	if !strings.Contains(output, "No failures found") {
		t.Errorf("expected 'No failures found' in output, got: %s", output)
	}
}

func TestLogCmd_LinesFlag_LimitsOutput(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir = createTempLogsStructure(t, tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create ops-log.jsonl with 5 entries
	opsLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "ops-log.jsonl")
	opsLines := []string{
		`{"ts":"2026-04-14T10:00:00Z","type":"health","message":"entry1"}`,
		`{"ts":"2026-04-14T10:01:00Z","type":"health","message":"entry2"}`,
		`{"ts":"2026-04-14T10:02:00Z","type":"health","message":"entry3"}`,
		`{"ts":"2026-04-14T10:03:00Z","type":"health","message":"entry4"}`,
		`{"ts":"2026-04-14T10:04:00Z","type":"health","message":"entry5"}`,
	}
	writeJSONLFile(t, opsLogPath, opsLines)

	resetLogFlags()
	logLines = 2

	output, err := captureStdout(func() error {
		return logCmd.RunE(logCmd, []string{})
	})

	if err != nil {
		t.Fatalf("logCmd.RunE failed: %v", err)
	}

	// Should show only last 2 entries (entry4 and entry5)
	if !strings.Contains(output, "entry4") {
		t.Errorf("expected 'entry4' in output, got: %s", output)
	}
	if !strings.Contains(output, "entry5") {
		t.Errorf("expected 'entry5' in output, got: %s", output)
	}
	// Should NOT show earlier entries
	if strings.Contains(output, "entry1") {
		t.Errorf("should not show 'entry1' when --lines 2: %s", output)
	}
	if strings.Contains(output, "entry2") {
		t.Errorf("should not show 'entry2' when --lines 2: %s", output)
	}
	if strings.Contains(output, "entry3") {
		t.Errorf("should not show 'entry3' when --lines 2: %s", output)
	}
}

func TestLogCmd_NoLogFilesExist_NoEntriesMessage(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir = createTempLogsStructure(t, tmpDir)
	t.Setenv("HOME", tmpDir)

	// Don't create any log files, just the directories
	resetLogFlags()

	output, err := captureStdout(func() error {
		return logCmd.RunE(logCmd, []string{})
	})

	if err != nil {
		t.Fatalf("logCmd.RunE failed: %v", err)
	}

	// Should show "No log entries found" message
	if !strings.Contains(output, "No log entries found") {
		t.Errorf("expected 'No log entries found' in output, got: %s", output)
	}
}

func TestLogCmd_MissingMessage_BuildsFromServices(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir = createTempLogsStructure(t, tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create ops-log.jsonl with entry that has no message but has services data
	opsLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "ops-log.jsonl")
	opsLines := []string{
		`{"ts":"2026-04-14T10:00:00Z","type":"health","services":{"active":5,"total":6}}`,
	}
	writeJSONLFile(t, opsLogPath, opsLines)

	resetLogFlags()

	output, err := captureStdout(func() error {
		return logCmd.RunE(logCmd, []string{})
	})

	if err != nil {
		t.Fatalf("logCmd.RunE failed: %v", err)
	}

	// Should build message from services data: "services 5/6"
	if !strings.Contains(output, "services 5/6") {
		t.Errorf("expected 'services 5/6' in output, got: %s", output)
	}
}

func TestLogCmd_StatusIcon_ForFailures(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir = createTempLogsStructure(t, tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create ops-log.jsonl with failure entry
	opsLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "ops-log.jsonl")
	opsLines := []string{
		`{"ts":"2026-04-14T10:00:00Z","type":"health","status":"fail","message":"system failure"}`,
	}
	writeJSONLFile(t, opsLogPath, opsLines)

	resetLogFlags()

	output, err := captureStdout(func() error {
		return logCmd.RunE(logCmd, []string{})
	})

	if err != nil {
		t.Fatalf("logCmd.RunE failed: %v", err)
	}

	// Should show red X icon for failures (ANSI escape codes present)
	if !strings.Contains(output, "\033[31m") {
		t.Errorf("expected red color code in output for failure: %s", output)
	}
}

func TestLogCmd_SourceAudit_OnlyReadsAuditLog(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir = createTempLogsStructure(t, tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create audit log file
	auditLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "audits", "audit.jsonl")
	auditLines := []string{
		`{"ts":"2026-04-14T10:10:00Z","type":"audit","message":"audit entry"}`,
	}
	writeJSONLFile(t, auditLogPath, auditLines)

	// Also create ops log (should be ignored)
	opsLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "ops-log.jsonl")
	opsLines := []string{
		`{"ts":"2026-04-14T10:00:00Z","type":"ops","message":"ops entry"}`,
	}
	writeJSONLFile(t, opsLogPath, opsLines)

	resetLogFlags()
	logSource = "audit"

	output, err := captureStdout(func() error {
		return logCmd.RunE(logCmd, []string{})
	})

	if err != nil {
		t.Fatalf("logCmd.RunE failed: %v", err)
	}

	// Should only show audit entry
	if !strings.Contains(output, "audit entry") {
		t.Errorf("expected 'audit entry' in output, got: %s", output)
	}
	if strings.Contains(output, "ops entry") {
		t.Errorf("should not show 'ops entry' when --source audit: %s", output)
	}
}

func TestLogCmd_CombinedFlags_FailuresAndSource(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir = createTempLogsStructure(t, tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create ops-log.jsonl with mixed entries
	opsLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "ops-log.jsonl")
	opsLines := []string{
		`{"ts":"2026-04-14T10:00:00Z","type":"health","status":"ok","message":"system ok"}`,
		`{"ts":"2026-04-14T10:05:00Z","type":"health","status":"fail","message":"system fail"}`,
	}
	writeJSONLFile(t, opsLogPath, opsLines)

	// Create events log with failure
	eventsLogPath := filepath.Join(tmpDir, "nightagent-sync", "ops", "events.jsonl")
	eventsLines := []string{
		`{"ts":"2026-04-14T10:10:00Z","type":"event","status":"fail","message":"event fail"}`,
	}
	writeJSONLFile(t, eventsLogPath, eventsLines)

	resetLogFlags()
	logFailures = true
	logSource = "ops"

	output, err := captureStdout(func() error {
		return logCmd.RunE(logCmd, []string{})
	})

	if err != nil {
		t.Fatalf("logCmd.RunE failed: %v", err)
	}

	// Should show only ops failures
	if !strings.Contains(output, "system fail") {
		t.Errorf("expected 'system fail' in output, got: %s", output)
	}
	// Should NOT show successes
	if strings.Contains(output, "system ok") {
		t.Errorf("should not show 'system ok' when --failures set: %s", output)
	}
	// Should NOT show events failures
	if strings.Contains(output, "event fail") {
		t.Errorf("should not show 'event fail' when --source ops: %s", output)
	}
}
