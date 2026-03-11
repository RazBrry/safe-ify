# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**safe-ify** is a CLI tool written in Go that acts as a permissioned safety layer for coding agents (like Claude Code) to interact with [Coolify](https://coolify.io/) v4 — an open-source, self-hostable PaaS. It exposes only vetted, non-destructive operations (deploy, redeploy, logs, status, list) and prevents agents from accessing raw credentials or destructive endpoints.

## Tech Stack

- **Language**: Go
- **CLI framework**: [Cobra](https://github.com/spf13/cobra)
- **TUI**: [Charm](https://charm.sh/) ecosystem (`huh` for forms, `lipgloss` for styling)
- **HTTP client**: Go stdlib `net/http`
- **Config format**: YAML (`gopkg.in/yaml.v3`)
- **Build**: `make build` → `./bin/safe-ify`
- **Testing**: Go built-in (`go test`)

## Architecture

```
cmd/
  safe-ify/
    main.go              # Entrypoint
internal/
  cli/
    root.go              # Root Cobra command
    auth.go              # auth add/remove/list (TUI)
    init.go              # init command (TUI)
    deploy.go            # Agent command: deploy
    redeploy.go          # Agent command: redeploy
    logs.go              # Agent command: logs
    status.go            # Agent command: status
    list.go              # Agent command: list
    doctor.go            # Doctor command
    output.go            # JSON envelope formatting
  config/
    global.go            # Global config (~/.config/safe-ify/config.yaml, 0600)
    project.go           # Project config (.safe-ify.yaml, committable, no secrets)
    types.go             # Config structs
  permissions/
    enforcer.go          # Deny-only permission enforcement
  coolify/
    client.go            # Coolify v4 API client
    types.go             # API request/response types
  audit/
    logger.go            # Append-only audit log
  tui/
    forms.go             # Charm huh forms
    styles.go            # Lipgloss styles
```

### Key Design Principles

- **Allowlist-only**: Only deploy, redeploy, logs, status, list endpoints exposed. Never delete, stop, or config-change.
- **Two-layer config**: Global config (secrets, `0600`) + project config (committable, no secrets). Project can only restrict, never escalate.
- **Machine-readable output**: All agent commands support `--json` with standard `{ok, data, error}` envelope.
- **Token output contract**: Tokens are never printed, never in JSON output, never in audit logs.
- **Fail-safe**: Surface Coolify API errors clearly. Never retry or guess.

## Development Commands

```bash
# Build
make build

# Run during development
go run ./cmd/safe-ify <command>

# Run all tests
go test ./...

# Run a single test file
go test ./internal/config/ -run TestConfigLoad

# Lint
make lint

# Install locally
go install ./cmd/safe-ify
```

## Configuration

- **Global**: `~/.config/safe-ify/config.yaml` — instance definitions (name, URL, token), file permissions `0600`
- **Project**: `.safe-ify.yaml` in repo root — references instance by name, app ID, permission overrides (no secrets)

## CLI Commands

### Human-facing (interactive TUI)
| Command | Description |
|---------|-------------|
| `safe-ify auth add` | Add a Coolify instance (URL + token) |
| `safe-ify auth remove` | Remove a configured instance |
| `safe-ify auth list` | List configured instances (tokens masked) |
| `safe-ify init` | Link project to Coolify instance/app, configure permissions |

### Agent-facing (non-interactive, `--json`)
| Command | Description |
|---------|-------------|
| `safe-ify deploy --json` | Trigger deployment |
| `safe-ify deploy --json --wait` | Deploy and wait for completion (polls status) |
| `safe-ify redeploy --json` | Redeploy current version |
| `safe-ify logs --json --tail N` | Fetch recent logs (default: 100 lines) |
| `safe-ify status --json` | Check deployment status |
| `safe-ify list --json` | List available applications |
| `safe-ify doctor` | Validate setup, output CLAUDE.md snippet |

## Planning Workflow

Active plans live in `plan/`. See `docs/workflow/plan-process.md` for the full SOP.

Current plan: `plan/PLN-2026-001-safe-ify-v1-scaffold/` — implements this entire tool in 4 slices.
