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
| `safe-ify deploy --app api --json --wait` | Deploy and wait for completion (polls status) |
| `safe-ify redeploy --app api --json` | Redeploy current version ¹ |
| `safe-ify logs --app api --json --tail 50` | Fetch recent logs (default: 100 lines) |
| `safe-ify status --app api --json` | Check deployment status |
| `safe-ify list --json` | List available applications (no `--app` needed) |
| `safe-ify env list --app api --json` | List env var keys (add `--show-values` for values) |
| `safe-ify env get --app api --key DB_HOST --json` | Get a specific env var value |
| `safe-ify env set --app api --key DB_HOST --value localhost --json` | Create or update an env var |
| `safe-ify env delete --app api --key OLD_VAR --json` | Delete an env var |

> ¹ `redeploy` uses the Coolify `/restart` endpoint, which may return 403 on some Coolify versions even with the correct token scopes. If you hit this, use `deploy --force` instead.

Both `deploy` and `redeploy` support `--wait` to poll until completion (`--timeout`, `--poll-interval` configurable).

## API token requirements

Create a Coolify API token at **Settings → API Tokens** with these scopes:

| Scope | Required for |
|-------|-------------|
| `read` | status, logs, list, deployment polling |
| `deploy` | deploy, redeploy |
| `read:sensitive` | reading environment variable values (`env list --show-values`, `env get`) |
| `write` | modifying environment variables (`env set`, `env delete`) |

Minimum for current features: **`read` + `deploy`**.

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
      deny: [deploy, env-write]  # web app: no deploys, no env changes
```

Legacy single-app configs (`app_uuid` at root level) are auto-detected and work without changes.

## Security model

safe-ify enforces a strict separation between human and agent capabilities:

- **Agents can only** run the allowlisted commands (`deploy`, `redeploy`, `logs`, `status`, `list`, `env list/get/set/delete`) within the permissions granted by the config.
- **Agents cannot** modify configuration — `init`, `auth add`, and `auth remove` require an interactive terminal (TTY check) and will refuse to run when called from a non-interactive shell.
- **Permissions are deny-only** — each layer (global, project, per-app) can only restrict further, never grant back a denied command.
- **Tokens are never exposed** — not in CLI output, not in JSON responses, not in audit logs.

## Tech stack

Go, [Cobra](https://github.com/spf13/cobra), [Charm](https://charm.sh/) (huh + lipgloss), YAML config.

## License

MIT
