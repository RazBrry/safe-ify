package audit

import (
	"fmt"
	"os"
	"path/filepath"
)

// Logger writes audit entries to an append-only log file.
type Logger struct {
	path string
}

// DefaultAuditLogPath returns the default path for the audit log file.
func DefaultAuditLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "safe-ify", "audit.log")
	}
	return filepath.Join(home, ".config", "safe-ify", "audit.log")
}

// NewLogger creates a new Logger. If path is empty, defaults to DefaultAuditLogPath().
func NewLogger(path string) *Logger {
	if path == "" {
		path = DefaultAuditLogPath()
	}
	return &Logger{path: path}
}

// Log writes the entry to the audit log file, creating the file and its parent
// directory if they do not exist.
func (l *Logger) Log(entry Entry) error {
	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("audit: cannot create log directory: %w", err)
	}

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("audit: cannot open log file: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintln(f, entry.String())
	if err != nil {
		return fmt.Errorf("audit: cannot write log entry: %w", err)
	}
	return nil
}
