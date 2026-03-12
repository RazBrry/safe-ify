package permissions

import "fmt"

// PermissionDeniedError is returned when a command is not permitted for a project.
type PermissionDeniedError struct {
	Command  string
	DeniedBy string // "global", "project", or "app"
}

// Error implements the error interface.
func (e *PermissionDeniedError) Error() string {
	return fmt.Sprintf("command %q is not permitted for this project (denied by %s)", e.Command, e.DeniedBy)
}
