# PLN-2026-002 — Tech Spec Developer: CLI Command Changes

## --app Flag

### Registration

Add `--app` as a persistent flag on the root command in `internal/cli/root.go`:

```go
rootCmd.PersistentFlags().StringVar(&appName, "app", "", "Target app name (required when multiple apps configured)")
```

This makes `--app` available to all subcommands.

### Propagation

In `resolveAgentConfig` (`internal/cli/agent.go`):

1. Read the `--app` flag value: `appName, _ := cmd.Root().PersistentFlags().GetString("app")`
2. Pass `appName` to `config.ResolveRuntime(globalCfg, projectCfg, appName)`
3. The returned `RuntimeConfig` now contains the resolved `AppUUID` and `AppName`

### Error Codes

New error codes in `internal/cli/output.go`:

```go
ErrCodeAppNotFound  = "APP_NOT_FOUND"
ErrCodeAppAmbiguous = "APP_AMBIGUOUS"
```

Map in `mapConfigError`:
- `*config.AppNotFoundError` --> `ErrCodeAppNotFound`
- `*config.AppAmbiguousError` --> `ErrCodeAppAmbiguous`

## Agent Commands (deploy, redeploy, logs, status, list)

No changes to individual command files. All changes are in the shared `resolveAgentConfig` and `runAgentCommand` functions in `agent.go`:

- `resolveAgentConfig` passes `--app` to `ResolveRuntime`
- `runAgentCommand` uses `cfg.AppName` in the audit entry (see Audit section)
- Permission checking in each command already uses `cfg.AllowedCmds` which will now include app-level deny list contributions

The `list` command is special: it lists applications on the instance and does not target a specific app. However, it still requires a valid project config. With multi-app support, `list` should work without `--app` even when multiple apps exist, since it does not target a specific app. Implementation: `list` should call `ResolveRuntime` with a special mode or handle `AppAmbiguousError` gracefully.

**Decision (D6):** `list` command bypasses the `--app` requirement. It uses the instance from the project config but does not need an app UUID. `resolveAgentConfig` will accept an `appRequired bool` parameter.

## Init Command (`internal/cli/init.go`)

### New Behavior

1. Check if `.safe-ify.yaml` exists in cwd
2. If **no existing config**: run the current flow but save in new multi-app format:
   - Instance picker
   - Application picker (Coolify app)
   - App name prompt (new TUI form): "What name should this app have in your config?" (default: app name from Coolify, sanitized)
   - Deny list picker
   - Save with `apps:` map containing one entry
3. If **existing config found**: prompt "Add another app?"
   - If yes: load existing config, application picker, app name prompt, deny list picker
   - Validate app name is not already used
   - Append to `Apps` map, save
   - If existing config is legacy format, normalize to new format during load (per D2)
   - If no: print message and exit

### New TUI Form

Add `InitAppNameForm` in `internal/tui/forms.go`:

```go
func InitAppNameForm(defaultName string, existingNames []string, name *string) *huh.Form
```

Validates: non-empty, matches `^[a-zA-Z0-9][a-zA-Z0-9-]*$`, not in `existingNames`.

## Doctor Command (`internal/cli/doctor.go`)

### Updated Checks

The doctor currently runs 8 checks. Multi-app changes:

- **Check 7 (App UUID check)**: Loop over all apps in the config. For each app, call `GetApplication` with its UUID. Report PASS/FAIL per app.
- **Check 8 (Permissions check)**: Loop over all apps. For each app, build an enforcer with the merged deny list and report allowed/denied commands.
- **CLAUDE.md snippet**: List each app with its status in the available commands table.

Check count changes from 8 to 8 (same structure, but checks 7 and 8 iterate over apps).

## Audit Entry Update (`internal/audit/`)

### Entry struct

Add `AppName string` to `Entry` in `internal/audit/types.go`.

### String format

Update `Entry.String()` to include app name:

```
YYYY-MM-DDTHH:MM:SSZ | command | app_name | app_uuid | instance | result | duration_ms
```

**Note:** This changes the log line format. Existing log entries will not have the app name field. This is acceptable because the audit log is append-only and read by humans, not parsed programmatically (no existing parsers to break).

### Usage in agent.go

In `runAgentCommand`, set `entry.AppName = cfg.AppName`.
