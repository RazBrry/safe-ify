package audit

import (
	"fmt"
	"time"
)

// Entry represents a single audit log entry for an agent-facing command invocation.
type Entry struct {
	Timestamp  time.Time
	Command    string
	AppUUID    string
	Instance   string
	Result     string // "ok" or "error"
	DurationMs int64
}

// String formats the entry as a pipe-delimited log line.
// Format: YYYY-MM-DDTHH:MM:SSZ | command | app_uuid | instance | result | duration_ms
func (e Entry) String() string {
	return fmt.Sprintf("%s | %s | %s | %s | %s | %d",
		e.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
		e.Command,
		e.AppUUID,
		e.Instance,
		e.Result,
		e.DurationMs,
	)
}
