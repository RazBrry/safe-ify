package permissions

import (
	"errors"
	"testing"

	"github.com/erwinmaasbach/safe-ify/internal/config"
)

// helpers

func makeGlobal(deny ...string) config.GlobalConfig {
	return config.GlobalConfig{
		Defaults: config.DefaultSettings{
			Permissions: config.PermissionConfig{Deny: deny},
		},
	}
}

func makeProject(deny ...string) config.ProjectConfig {
	return config.ProjectConfig{
		Permissions: config.PermissionConfig{Deny: deny},
	}
}

// allCommandsAllowed returns true if all 5 agent commands are allowed.
func allCommandsAllowed(e *Enforcer) bool {
	for _, cmd := range AllAgentCommands {
		if e.Check(cmd) != nil {
			return false
		}
	}
	return true
}

// --- ResolvePermissions tests (via NewEnforcer) ---

func TestResolvePermissions_NoRestrictions(t *testing.T) {
	e := NewEnforcer(makeGlobal(), makeProject())
	for _, cmd := range AllAgentCommands {
		if err := e.Check(cmd); err != nil {
			t.Errorf("expected %q to be allowed, got error: %v", cmd, err)
		}
	}
}

func TestResolvePermissions_GlobalDeny(t *testing.T) {
	e := NewEnforcer(makeGlobal("deploy"), makeProject())

	if err := e.Check("deploy"); err == nil {
		t.Error("expected deploy to be denied, got nil")
	}

	for _, cmd := range []string{"redeploy", "logs", "status", "list"} {
		if err := e.Check(cmd); err != nil {
			t.Errorf("expected %q to be allowed after global deny of deploy, got: %v", cmd, err)
		}
	}
}

func TestResolvePermissions_ProjectDeny(t *testing.T) {
	e := NewEnforcer(makeGlobal(), makeProject("logs"))

	if err := e.Check("logs"); err == nil {
		t.Error("expected logs to be denied, got nil")
	}

	for _, cmd := range []string{"deploy", "redeploy", "status", "list"} {
		if err := e.Check(cmd); err != nil {
			t.Errorf("expected %q to be allowed after project deny of logs, got: %v", cmd, err)
		}
	}
}

// TestResolvePermissions_ProjectCannotEscalate is the critical escalation
// prevention test: even when the project deny list is empty, a globally denied
// command must remain denied.
func TestResolvePermissions_ProjectCannotEscalate(t *testing.T) {
	// Global denies "deploy"; project deny list is intentionally empty.
	e := NewEnforcer(makeGlobal("deploy"), makeProject())

	if err := e.Check("deploy"); err == nil {
		t.Fatal("ESCALATION INVARIANT VIOLATED: globally denied command 'deploy' was allowed at project level")
	}

	// All other commands should still be allowed.
	for _, cmd := range []string{"redeploy", "logs", "status", "list"} {
		if err := e.Check(cmd); err != nil {
			t.Errorf("expected %q to be allowed, got: %v", cmd, err)
		}
	}
}

func TestResolvePermissions_CombinedDenials(t *testing.T) {
	e := NewEnforcer(makeGlobal("deploy"), makeProject("logs"))

	for _, denied := range []string{"deploy", "logs"} {
		if err := e.Check(denied); err == nil {
			t.Errorf("expected %q to be denied, got nil", denied)
		}
	}

	for _, allowed := range []string{"redeploy", "status", "list"} {
		if err := e.Check(allowed); err != nil {
			t.Errorf("expected %q to be allowed, got: %v", allowed, err)
		}
	}
}

// --- Enforcer.Check tests ---

func TestEnforcer_AllowedCommand(t *testing.T) {
	e := NewEnforcer(makeGlobal(), makeProject())
	if err := e.Check("status"); err != nil {
		t.Errorf("Check(\"status\") on unrestricted enforcer: expected nil, got %v", err)
	}
}

func TestEnforcer_DeniedCommand(t *testing.T) {
	e := NewEnforcer(makeGlobal("deploy"), makeProject())
	err := e.Check("deploy")
	if err == nil {
		t.Fatal("expected PermissionDeniedError, got nil")
	}
	var pde *PermissionDeniedError
	if !errors.As(err, &pde) {
		t.Errorf("expected *PermissionDeniedError, got %T: %v", err, err)
	}
}

func TestEnforcer_DeniedBy_Global(t *testing.T) {
	e := NewEnforcer(makeGlobal("deploy"), makeProject())
	err := e.Check("deploy")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pde *PermissionDeniedError
	if !errors.As(err, &pde) {
		t.Fatalf("expected *PermissionDeniedError, got %T", err)
	}
	if pde.DeniedBy != "global" {
		t.Errorf("expected DeniedBy==\"global\", got %q", pde.DeniedBy)
	}
}

func TestEnforcer_DeniedBy_Project(t *testing.T) {
	e := NewEnforcer(makeGlobal(), makeProject("logs"))
	err := e.Check("logs")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pde *PermissionDeniedError
	if !errors.As(err, &pde) {
		t.Fatalf("expected *PermissionDeniedError, got %T", err)
	}
	if pde.DeniedBy != "project" {
		t.Errorf("expected DeniedBy==\"project\", got %q", pde.DeniedBy)
	}
}

// --- ValidateDenyList tests ---

func TestValidateDenyList_ValidCommands(t *testing.T) {
	if err := ValidateDenyList(AllAgentCommands); err != nil {
		t.Errorf("expected all 5 valid commands to pass validation, got: %v", err)
	}
}

func TestValidateDenyList_InvalidCommand(t *testing.T) {
	if err := ValidateDenyList([]string{"delete"}); err == nil {
		t.Error("expected error for unknown command \"delete\", got nil")
	}
}

func TestValidateDenyList_EmptyList(t *testing.T) {
	if err := ValidateDenyList([]string{}); err != nil {
		t.Errorf("expected empty deny list to pass validation, got: %v", err)
	}
}
