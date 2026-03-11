# PLN-2026-002 — Functionele Uitwerking

## User Stories

### US1: Monorepo developer configures multiple apps
As a developer working in a monorepo, I want to register multiple Coolify apps (e.g., `frontend`, `api`, `worker`) in a single `.safe-ify.yaml` so that I can deploy each one from the same project directory.

### US2: Agent targets a specific app
As a coding agent, I want to run `safe-ify deploy --app api --json` to deploy a specific app, so that I can target the correct service in a multi-app project.

### US3: Single-app projects work unchanged
As a developer with an existing single-app `.safe-ify.yaml`, I want my config to continue working without changes, so that upgrading safe-ify does not break my workflow.

### US4: Per-app permission control
As a team lead, I want to deny `deploy` for the `production-db` app but allow it for `staging-api`, so that agents cannot accidentally deploy restricted services.

### US5: Doctor validates all apps
As a developer, I want `safe-ify doctor` to validate connectivity and permissions for all configured apps, so that I can diagnose issues across my full setup.

## Flows

### Flow 1: Init — first app (new project)
1. User runs `safe-ify init` in a directory without `.safe-ify.yaml`
2. Instance picker (unchanged)
3. Application picker (unchanged)
4. User is prompted for an app name (short label, e.g., `api`)
5. Permission deny list picker (unchanged — applies to this app)
6. Config saved as new multi-app format with one app entry

### Flow 2: Init — add app (existing project)
1. User runs `safe-ify init` in a directory with existing `.safe-ify.yaml`
2. Existing config detected; user is asked: "Add another app to this project?"
3. If yes: application picker, app name prompt, deny list picker
4. New app is appended to `apps:` map
5. If existing config uses old single-app format, it is migrated to the new format first

### Flow 3: Agent command with --app
1. Agent runs `safe-ify deploy --app frontend --json`
2. Config loaded; `frontend` resolved to its UUID
3. Permissions checked: global deny + project deny + app-specific deny
4. Coolify API called with the resolved UUID
5. Audit entry written with app name `frontend` and UUID

### Flow 4: Agent command without --app (single app)
1. Agent runs `safe-ify deploy --json` without `--app`
2. Config loaded; only one app configured
3. That app is used as the default
4. Command proceeds as today

### Flow 5: Agent command without --app (multiple apps)
1. Agent runs `safe-ify deploy --json` without `--app`
2. Config loaded; multiple apps configured
3. Error returned: `"multiple apps configured; use --app <name> to select one"`
4. JSON output: `{ok: false, error: {code: "APP_AMBIGUOUS", message: "..."}}`

## Edge Cases

- **Unknown app name:** `--app nonexistent` returns `APP_NOT_FOUND` error
- **Duplicate app name in config:** Prevented by YAML map keys being unique
- **Empty apps map:** Treated as missing config (same error as no `app_uuid`)
- **Old format with `app_uuid: ""` (empty):** Existing validation catches this
- **App name with special characters:** Validated to alphanumeric + hyphens only (same rule as instance names)
