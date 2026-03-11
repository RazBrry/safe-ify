# safe-ify

A CLI safety layer for coding agents to interact with [Coolify](https://coolify.io/) v4.

**safe-ify** exposes only vetted, non-destructive operations (deploy, redeploy, logs, status, list) and prevents agents from accessing raw credentials or destructive API endpoints. Designed for monorepos with multiple Coolify apps.

## Features

- **Allowlist-only** — only safe operations exposed, never delete/stop/config-change
- **Two-layer config** — global secrets (`~/.config/safe-ify/config.yaml`, `0600`) + committable project config (`.safe-ify.yaml`)
- **Multi-app support** — manage multiple Coolify apps per project with `--app` flag
- **Three-layer permissions** — global, project, and per-app deny lists (each layer can only restrict further)
- **Machine-readable output** — all agent commands support `--json` with `{ok, data, error}` envelope
- **Audit logging** — append-only log of all agent actions
- **Zero credential leakage** — tokens never printed, never in JSON output, never in audit logs
- **TTY-guarded config** — `init`, `auth add`, and `auth remove` require an interactive terminal, so agents cannot modify configuration

## Quick start

```bash
# Install
go install github.com/RazBrry/safe-ify/cmd/safe-ify@latest

# Or build from source
make build  # → ./bin/safe-ify

# Add your Coolify instance
safe-ify auth add

# Link a project (run from your repo root — multi-select your apps)
safe-ify init

# Run again to add/remove apps (already configured apps are pre-selected)
safe-ify init
```

## Usage

### Human commands (interactive TUI)

| Command | Description |
|---------|-------------|
| `safe-ify auth add` | Add a Coolify instance (URL + token) |
| `safe-ify auth remove` | Remove a configured instance |
| `safe-ify auth list` | List configured instances (tokens masked) |
| `safe-ify init` | Multi-select Coolify apps for this project (re-run to add/remove) |
| `safe-ify doctor` | Validate setup, output CLAUDE.md snippet |

### Agent commands (non-interactive, `--json`)

| Command | Description |
|---------|-------------|
| `safe-ify deploy --app api --json` | Trigger deployment |
| `safe-ify redeploy --app api --json` | Redeploy current version |
| `safe-ify logs --app api --json --tail 50` | Fetch recent logs |
| `safe-ify status --app api --json` | Check deployment status |
| `safe-ify list --json` | List available applications (no `--app` needed) |

## Multi-app config

For monorepos with multiple Coolify apps, `.safe-ify.yaml` uses a named app map:

```yaml
instance: my-coolify
apps:
  api:
    app_uuid: "abc123-..."
  web:
    app_uuid: "def456-..."
    permissions:
      deny: [deploy]  # web app can't be deployed by agents
```

Legacy single-app configs (`app_uuid` at root level) are auto-detected and work without changes.

## Security model

safe-ify enforces a strict separation between human and agent capabilities:

- **Agents can only** run the five allowlisted commands (`deploy`, `redeploy`, `logs`, `status`, `list`) within the permissions granted by the config.
- **Agents cannot** modify configuration — `init`, `auth add`, and `auth remove` require an interactive terminal (TTY check) and will refuse to run when called from a non-interactive shell.
- **Permissions are deny-only** — each layer (global, project, per-app) can only restrict further, never grant back a denied command.
- **Tokens are never exposed** — not in CLI output, not in JSON responses, not in audit logs.

## Tech stack

Go, [Cobra](https://github.com/spf13/cobra), [Charm](https://charm.sh/) (huh + lipgloss), YAML config.

## License

MIT
