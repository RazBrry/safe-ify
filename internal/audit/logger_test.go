package audit

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestEntry_String verifies that Entry.String() produces the expected
// pipe-delimited format: YYYY-MM-DDTHH:MM:SSZ | command | app_uuid | instance | result | duration_ms
func TestEntry_String(t *testing.T) {
	ts := time.Date(2026, 3, 11, 14, 5, 30, 0, time.UTC)
	e := Entry{
		Timestamp:  ts,
		Command:    "deploy",
		AppUUID:    "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		Instance:   "my-coolify",
		Result:     "ok",
		DurationMs: 123,
	}

	got := e.String()
	want := "2026-03-11T14:05:30Z | deploy | a1b2c3d4-e5f6-7890-abcd-ef1234567890 | my-coolify | ok | 123"
	if got != want {
		t.Errorf("Entry.String() mismatch\n  got:  %q\n  want: %q", got, want)
	}

	// Verify the parts individually.
	parts := strings.Split(got, " | ")
	if len(parts) != 6 {
		t.Fatalf("expected 6 pipe-delimited fields, got %d", len(parts))
	}
	if parts[0] != "2026-03-11T14:05:30Z" {
		t.Errorf("timestamp field: got %q, want %q", parts[0], "2026-03-11T14:05:30Z")
	}
	if parts[1] != "deploy" {
		t.Errorf("command field: got %q, want %q", parts[1], "deploy")
	}
	if parts[2] != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Errorf("app_uuid field: got %q", parts[2])
	}
	if parts[3] != "my-coolify" {
		t.Errorf("instance field: got %q", parts[3])
	}
	if parts[4] != "ok" {
		t.Errorf("result field: got %q", parts[4])
	}
	if parts[5] != "123" {
		t.Errorf("duration_ms field: got %q", parts[5])
	}
}

// TestEntry_String_UTC verifies that non-UTC timestamps are converted to UTC in
// the formatted output. Uses a fixed-offset timezone to avoid DST ambiguity.
func TestEntry_String_UTC(t *testing.T) {
	// Use a fixed UTC-5 offset to avoid DST ambiguity.
	loc := time.FixedZone("UTC-5", -5*60*60)
	// 2026-03-11 10:00:00 UTC-5 = 2026-03-11 15:00:00 UTC
	ts := time.Date(2026, 3, 11, 10, 0, 0, 0, loc)
	e := Entry{
		Timestamp:  ts,
		Command:    "status",
		AppUUID:    "uuid-1",
		Instance:   "inst-1",
		Result:     "ok",
		DurationMs: 0,
	}
	got := e.String()
	if !strings.HasPrefix(got, "2026-03-11T15:00:00Z") {
		t.Errorf("expected UTC timestamp prefix 2026-03-11T15:00:00Z, got: %q", got)
	}
}

// TestLogger_WritesEntry verifies that Log() creates the audit file and writes
// a correctly-formatted entry.
func TestLogger_WritesEntry(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.log")
	logger := NewLogger(logPath)

	ts := time.Date(2026, 3, 11, 9, 0, 0, 0, time.UTC)
	entry := Entry{
		Timestamp:  ts,
		Command:    "status",
		AppUUID:    "app-uuid-1",
		Instance:   "test-instance",
		Result:     "ok",
		DurationMs: 42,
	}

	if err := logger.Log(entry); err != nil {
		t.Fatalf("Log() returned error: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", logPath, err)
	}

	content := string(data)
	expectedLine := "2026-03-11T09:00:00Z | status | app-uuid-1 | test-instance | ok | 42"
	if !strings.Contains(content, expectedLine) {
		t.Errorf("log file does not contain expected line\n  want: %q\n  got:  %q", expectedLine, content)
	}
}

// TestLogger_AppendsToExisting verifies that successive Log() calls append entries
// rather than overwriting the file.
func TestLogger_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.log")
	logger := NewLogger(logPath)

	ts1 := time.Date(2026, 3, 11, 9, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 3, 11, 9, 1, 0, 0, time.UTC)

	entry1 := Entry{Timestamp: ts1, Command: "deploy", AppUUID: "uuid-1", Instance: "inst", Result: "ok", DurationMs: 10}
	entry2 := Entry{Timestamp: ts2, Command: "status", AppUUID: "uuid-2", Instance: "inst", Result: "error", DurationMs: 20}

	if err := logger.Log(entry1); err != nil {
		t.Fatalf("Log(entry1): %v", err)
	}
	if err := logger.Log(entry2); err != nil {
		t.Fatalf("Log(entry2): %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)

	line1 := "2026-03-11T09:00:00Z | deploy | uuid-1 | inst | ok | 10"
	line2 := "2026-03-11T09:01:00Z | status | uuid-2 | inst | error | 20"

	if !strings.Contains(content, line1) {
		t.Errorf("first entry missing from log\n  want line containing: %q\n  content: %q", line1, content)
	}
	if !strings.Contains(content, line2) {
		t.Errorf("second entry missing from log\n  want line containing: %q\n  content: %q", line2, content)
	}

	// Count lines — there should be exactly 2 non-empty lines.
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(strings.TrimRight(content, "\n")))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) != 2 {
		t.Errorf("expected 2 log lines, got %d; content: %q", len(lines), content)
	}
}

// TestLogger_CreatesDirectory verifies that Log() creates the log directory (and
// any missing parents) if they do not exist.
func TestLogger_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	// Use a deeply nested path that does not exist yet.
	logPath := filepath.Join(dir, "nested", "deeply", "audit.log")
	logger := NewLogger(logPath)

	entry := Entry{
		Timestamp:  time.Now().UTC(),
		Command:    "list",
		AppUUID:    "uuid-x",
		Instance:   "inst-x",
		Result:     "ok",
		DurationMs: 5,
	}

	if err := logger.Log(entry); err != nil {
		t.Fatalf("Log() returned error when directory did not exist: %v", err)
	}

	// Verify the directory was created.
	logDir := filepath.Dir(logPath)
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Errorf("expected directory %q to be created, but it does not exist", logDir)
	}

	// Verify the file exists and has content.
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("expected log file %q to be created, but it does not exist", logPath)
	}
}

// TestLogger_FilePermissions verifies that the audit log file is created with
// 0600 permissions (readable/writable only by the owner).
func TestLogger_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.log")
	logger := NewLogger(logPath)

	entry := Entry{
		Timestamp:  time.Now().UTC(),
		Command:    "redeploy",
		AppUUID:    "uuid-perm",
		Instance:   "inst-perm",
		Result:     "ok",
		DurationMs: 1,
	}

	if err := logger.Log(entry); err != nil {
		t.Fatalf("Log(): %v", err)
	}

	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("Stat(%q): %v", logPath, err)
	}

	mode := info.Mode().Perm()
	if mode != 0o600 {
		t.Errorf("expected file permissions 0600, got %04o", mode)
	}
}

// TestLogger_ConcurrentWrites verifies that concurrent calls to Log() do not
// cause data loss (all entries must appear in the file).
func TestLogger_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.log")
	logger := NewLogger(logPath)

	const numEntries = 10
	var wg sync.WaitGroup
	wg.Add(numEntries)

	for i := 0; i < numEntries; i++ {
		i := i // capture loop var
		go func() {
			defer wg.Done()
			entry := Entry{
				Timestamp:  time.Now().UTC(),
				Command:    "status",
				AppUUID:    fmt.Sprintf("uuid-%d", i),
				Instance:   "inst",
				Result:     "ok",
				DurationMs: int64(i),
			}
			if err := logger.Log(entry); err != nil {
				t.Errorf("goroutine %d: Log() error: %v", i, err)
			}
		}()
	}

	wg.Wait()

	// Read the file and count lines.
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := strings.TrimRight(string(data), "\n")
	var count int
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			count++
		}
	}

	if count != numEntries {
		t.Errorf("expected %d log entries after concurrent writes, got %d\nfile content:\n%s", numEntries, count, string(data))
	}
}
