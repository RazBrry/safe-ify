package cli

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/erwinmaasbach/safe-ify/internal/audit"
	"github.com/erwinmaasbach/safe-ify/internal/config"
	"github.com/erwinmaasbach/safe-ify/internal/coolify"
	"github.com/erwinmaasbach/safe-ify/internal/permissions"
	"github.com/spf13/cobra"
)

// errExitCode1 is a sentinel error that signals the command should exit with
// code 1 without printing an additional error message (output was already written).
var errExitCode1 = errors.New("")

// resolveAgentConfig loads project + global configs, resolves the runtime
// config, builds a permission enforcer, and returns a ready-to-use Coolify client.
// The returned Enforcer should be used to check per-command permissions via enforcer.Check().
// When appRequired is false (e.g. for the list command), AppAmbiguousError is handled
// gracefully by clearing the app fields instead of returning an error.
func resolveAgentConfig(cmd *cobra.Command, appRequired bool) (*config.RuntimeConfig, *coolify.Client, *permissions.Enforcer, error) {
	// 1. Find and load project config (with parent traversal).
	projectOverride, _ := cmd.Root().PersistentFlags().GetString("project")
	var (
		projectPath string
		err         error
	)
	if projectOverride != "" {
		projectPath = projectOverride
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("cannot determine working directory: %w", err)
		}
		found, err := config.FindProjectConfig(cwd)
		if err != nil {
			return nil, nil, nil, err
		}
		projectPath = found
	}

	projectCfg, err := config.LoadProject(projectPath)
	if err != nil {
		return nil, nil, nil, err
	}

	// 2. Load global config.
	globalOverride, _ := cmd.Root().PersistentFlags().GetString("config")
	var globalPath string
	if globalOverride != "" {
		globalPath = globalOverride
	} else {
		globalPath, err = config.DefaultGlobalConfigPath()
		if err != nil {
			return nil, nil, nil, err
		}
	}

	globalCfg, err := config.LoadGlobal(globalPath)
	if err != nil {
		return nil, nil, nil, err
	}

	// 3. Read the --app flag and resolve runtime config.
	selectedApp, _ := cmd.Root().PersistentFlags().GetString("app")
	runtime, err := config.ResolveRuntime(globalCfg, projectCfg, selectedApp)
	if err != nil {
		if !appRequired {
			// For commands that don't target a specific app (e.g. list), handle
			// AppAmbiguousError gracefully by using instance info without an app.
			if ambig, ok := err.(*config.AppAmbiguousError); ok {
				_ = ambig
				// Build a partial runtime with instance info but no app.
				inst := globalCfg.Instances[projectCfg.Instance]
				runtime = &config.RuntimeConfig{
					InstanceName: projectCfg.Instance,
					InstanceURL:  inst.URL,
					Token:        inst.Token,
					AppUUID:      "",
					AppName:      "",
					AllowedCmds:  map[string]bool{"deploy": true, "redeploy": true, "logs": true, "status": true, "list": true},
				}
				err = nil
			}
		}
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// 4. Build permission enforcer from the canonical permissions package.
	// Pass the resolved app's deny list for three-layer enforcement.
	var appDeny []string
	if runtime.AppName != "" {
		if app, ok := projectCfg.Apps[runtime.AppName]; ok {
			appDeny = app.Permissions.Deny
		}
	}
	enforcer := permissions.NewEnforcer(*globalCfg, *projectCfg, appDeny)

	// 5. Create Coolify client.
	client := coolify.NewClient(runtime.InstanceURL, runtime.Token)

	return runtime, client, enforcer, nil
}

// mapConfigError maps a config-layer error to the appropriate JSON error code.
func mapConfigError(err error) string {
	switch err.(type) {
	case *config.ProjectConfigNotFoundError:
		return ErrCodeConfigNotFound
	case *config.ConfigNotFoundError:
		return ErrCodeConfigNotFound
	case *config.ConfigInsecureError:
		return ErrCodeConfigInsecure
	case *config.InstanceNotFoundError:
		return ErrCodeInstanceNotFound
	case *config.AppNotFoundError:
		return ErrCodeAppNotFound
	case *config.AppAmbiguousError:
		return ErrCodeAppAmbiguous
	default:
		return ErrCodeAPIError
	}
}

// mapCoolifyError maps a Coolify client error to the appropriate JSON error code.
// Network/transport failures map to NETWORK_ERROR; API response errors map to API_ERROR.
func mapCoolifyError(err error) string {
	var netErr *coolify.NetworkError
	if errors.As(err, &netErr) {
		return ErrCodeNetworkError
	}
	return ErrCodeAPIError
}

// runAgentCommand is an audit middleware wrapper that all five agent commands use.
// It resolves config, records timing, invokes fn, writes an audit entry, and
// returns the result. Audit write errors are printed to stderr but do not fail
// the command. appRequired controls whether an ambiguous app selection is an error
// (true for deploy/redeploy/logs/status) or gracefully handled (false for list).
func runAgentCommand(cmd *cobra.Command, commandName string, appRequired bool, fn func(cfg *config.RuntimeConfig, client *coolify.Client) (interface{}, error)) error {
	cfg, client, _, err := resolveAgentConfig(cmd, appRequired)
	if err != nil {
		return err
	}

	logger := audit.NewLogger(audit.DefaultAuditLogPath())
	start := time.Now()

	result, fnErr := fn(cfg, client)
	_ = result

	duration := time.Since(start).Milliseconds()

	auditResult := "ok"
	if fnErr != nil {
		auditResult = "error"
	}

	entry := audit.Entry{
		Timestamp:  start.UTC(),
		Command:    commandName,
		AppName:    cfg.AppName,
		AppUUID:    cfg.AppUUID,
		Instance:   cfg.InstanceName,
		Result:     auditResult,
		DurationMs: duration,
	}
	if logErr := logger.Log(entry); logErr != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "audit: %s\n", logErr)
	}

	return fnErr
}
