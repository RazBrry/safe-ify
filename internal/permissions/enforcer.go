package permissions

import (
	"fmt"

	"github.com/RazBrry/safe-ify/internal/config"
)

// Enforcer holds the resolved allow/deny state for agent commands.
type Enforcer struct {
	allowed  map[string]bool
	deniedBy map[string]string // command -> "global", "project", or "app"
}

// NewEnforcer builds an Enforcer by applying global, project, and app deny lists.
// Global denials are applied first; project denials can only further restrict;
// app denials can only further restrict beyond project.
func NewEnforcer(global config.GlobalConfig, project config.ProjectConfig, appDeny []string) *Enforcer {
	allowed := make(map[string]bool, len(AllAgentCommands))
	deniedBy := make(map[string]string)

	// Start with all commands allowed.
	for _, cmd := range AllAgentCommands {
		allowed[cmd] = true
	}

	// Apply global denials.
	for _, cmd := range global.Defaults.Permissions.Deny {
		allowed[cmd] = false
		deniedBy[cmd] = "global"
	}

	// Apply project denials (can only restrict further, never escalate).
	for _, cmd := range project.Permissions.Deny {
		if allowed[cmd] {
			// Only mark denied-by project if not already denied globally.
			deniedBy[cmd] = "project"
		}
		allowed[cmd] = false
	}

	// Apply app denials (can only restrict further, never escalate).
	for _, cmd := range appDeny {
		if allowed[cmd] {
			deniedBy[cmd] = "app"
		}
		allowed[cmd] = false
	}

	return &Enforcer{
		allowed:  allowed,
		deniedBy: deniedBy,
	}
}

// Check returns a PermissionDeniedError if the command is not allowed, or nil.
func (e *Enforcer) Check(command string) error {
	if !e.allowed[command] {
		source := e.deniedBy[command]
		if source == "" {
			source = "unknown"
		}
		return &PermissionDeniedError{
			Command:  command,
			DeniedBy: source,
		}
	}
	return nil
}

// AllowedCommands returns the list of commands that are currently allowed.
func (e *Enforcer) AllowedCommands() []string {
	var result []string
	for _, cmd := range AllAgentCommands {
		if e.allowed[cmd] {
			result = append(result, cmd)
		}
	}
	return result
}

// DeniedCommands returns the list of commands that are currently denied.
func (e *Enforcer) DeniedCommands() []string {
	var result []string
	for _, cmd := range AllAgentCommands {
		if !e.allowed[cmd] {
			result = append(result, cmd)
		}
	}
	return result
}

// ValidateDenyList returns an error if any entry in deny is not a known agent command.
func ValidateDenyList(deny []string) error {
	valid := make(map[string]bool, len(AllAgentCommands))
	for _, cmd := range AllAgentCommands {
		valid[cmd] = true
	}
	for _, cmd := range deny {
		if !valid[cmd] {
			return fmt.Errorf("unknown command in deny list: %q", cmd)
		}
	}
	return nil
}
