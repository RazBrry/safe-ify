package permissions

import "fmt"

// AllAgentCommands is the complete set of agent-facing commands that can be
// controlled via permission deny lists.
var AllAgentCommands = []string{
	"deploy",
	"redeploy",
	"logs",
	"status",
	"list",
	"env-read",
	"env-write",
}

// PermissionDeniedError is returned when a command is not permitted for a project.
type PermissionDeniedError struct {
	Command  string
	DeniedBy string // "global", "project", or "app"
}

// Error implements the error interface.
func (e *PermissionDeniedError) Error() string {
	return fmt.Sprintf("command %q is not permitted for this project (denied by %s)", e.Command, e.DeniedBy)
}
