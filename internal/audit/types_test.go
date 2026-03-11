package audit

import (
	"strings"
	"testing"
	"time"
)

// TestEntry_StringWithAppName verifies that Entry.String() includes the AppName
// in the correct pipe-delimited position (field index 2, between command and app_uuid).
func TestEntry_StringWithAppName(t *testing.T) {
	ts := time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)
	e := Entry{
		Timestamp:  ts,
		Command:    "deploy",
		AppName:    "api",
		AppUUID:    "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		Instance:   "prod",
		Result:     "ok",
		DurationMs: 55,
	}

	got := e.String()

	// Verify AppName appears in output.
	if !strings.Contains(got, "api") {
		t.Errorf("Entry.String() does not contain AppName %q\n  got: %q", "api", got)
	}

	// Verify the pipe-delimited format has 7 fields.
	parts := strings.Split(got, " | ")
	if len(parts) != 7 {
		t.Fatalf("expected 7 pipe-delimited fields, got %d: %q", len(parts), got)
	}

	// Field[0] = timestamp
	if parts[0] != "2026-03-11T12:00:00Z" {
		t.Errorf("field[0] timestamp: got %q, want %q", parts[0], "2026-03-11T12:00:00Z")
	}

	// Field[1] = command
	if parts[1] != "deploy" {
		t.Errorf("field[1] command: got %q, want %q", parts[1], "deploy")
	}

	// Field[2] = app_name — this is the position being tested.
	if parts[2] != "api" {
		t.Errorf("field[2] app_name: got %q, want %q", parts[2], "api")
	}

	// Field[3] = app_uuid
	if parts[3] != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Errorf("field[3] app_uuid: got %q, want %q", parts[3], "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	}

	// Field[4] = instance
	if parts[4] != "prod" {
		t.Errorf("field[4] instance: got %q, want %q", parts[4], "prod")
	}

	// Field[5] = result
	if parts[5] != "ok" {
		t.Errorf("field[5] result: got %q, want %q", parts[5], "ok")
	}

	// Field[6] = duration_ms
	if parts[6] != "55" {
		t.Errorf("field[6] duration_ms: got %q, want %q", parts[6], "55")
	}
}

// TestEntry_StringEmptyAppName verifies that Entry.String() handles an empty
// AppName gracefully: the field is present but empty (not skipped), so the
// output still has 7 pipe-delimited fields.
func TestEntry_StringEmptyAppName(t *testing.T) {
	ts := time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)
	e := Entry{
		Timestamp:  ts,
		Command:    "status",
		AppName:    "", // empty
		AppUUID:    "uuid-x",
		Instance:   "staging",
		Result:     "ok",
		DurationMs: 10,
	}

	got := e.String()

	// Must still produce 7 pipe-delimited fields.
	parts := strings.Split(got, " | ")
	if len(parts) != 7 {
		t.Fatalf("expected 7 pipe-delimited fields even with empty AppName, got %d: %q", len(parts), got)
	}

	// Field[2] must be the empty string (the AppName placeholder).
	if parts[2] != "" {
		t.Errorf("field[2] app_name: expected empty string, got %q", parts[2])
	}

	// Field[1] = command (ensure it was not displaced)
	if parts[1] != "status" {
		t.Errorf("field[1] command: got %q, want %q", parts[1], "status")
	}

	// Field[3] = app_uuid (ensure it was not displaced)
	if parts[3] != "uuid-x" {
		t.Errorf("field[3] app_uuid: got %q, want %q", parts[3], "uuid-x")
	}
}
