# PLN-2026-001 Context

Status: Draft
Updated: 2026-03-10
Owner: Planner (Opus)

## Key Decisions (compact -- full entries in 09-research-findings-decisions.md)

| ID | Decision |
|----|----------|
| D1 | Go + Cobra CLI framework, Charm TUI libraries |
| D2 | Two-layer config: global (secrets, 0600) + project (committable, no secrets) |
| D3 | Deny-only permission model at project level -- can only restrict, never escalate |
| D4 | Minimal safe API set: deploy, restart, logs, list, get-app; exclude stop/delete/update/envs |
| D5 | `deploy` maps to `/api/v1/deploy`, `redeploy` maps to `/applications/{uuid}/restart` |
| D6 | Standard JSON envelope: `{ok, data, error}` for all agent commands |
| D7 | Audit log: pipe-delimited text, append-only, at `~/.config/safe-ify/audit.log` |
| D8 | Charm `huh` for forms, `lipgloss` for styled output |
| D9 | YAML config format using `gopkg.in/yaml.v3` |
| D10 | Project config: parent directory traversal (like .git discovery) |
| D11 | POST for side-effecting operations (deploy, restart); GET only for reads |
| D12 | `list` command requires project config, same as all agent commands |
| D13 | Secret isolation enforced by tool output contract, not filesystem access |

## Deliverable -> Files Map

| Deliverable | Role | Model | Read these files |
|-------------|------|-------|-----------------|
| D-scaffold: Go project, Makefile, root cmd | Implementer | Sonnet | 03-tech-spec-dev-architecture.md, 05-tech-spec-dev-ops.md |
| D-config: Config loading/saving/security | Implementer | Sonnet | 04-tech-spec-dev-config-permissions.md |
| D-auth: Auth add/remove/list commands | Implementer | Sonnet | 04-tech-spec-dev-cli-commands.md |
| D-permissions: Permission enforcement | Implementer | Sonnet | 04-tech-spec-dev-config-permissions.md |
| D-init: Init command with TUI | Implementer | Sonnet | 04-tech-spec-dev-cli-commands.md |
| D-coolify: API client | Implementer | Sonnet | 03-tech-spec-dev-architecture.md, 08-research-api-matrix.md |
| D-agent-cmds: deploy/redeploy/logs/status/list | Implementer | Sonnet | 04-tech-spec-dev-cli-commands.md |
| D-doctor: Doctor command | Implementer | Sonnet | 04-tech-spec-dev-cli-commands.md |
| D-audit: Audit logging | Implementer | Sonnet | 03-tech-spec-dev-architecture.md |

## Slice gates (requires user approval -- each slice reviewed + merged independently)

| Slice | Gate task | Increment |
|-------|-----------|-----------|
| S1 | T5 | `go build` works, `auth add/remove/list` with TUI, global config secured |
| S2 | T10 | `init` works, `.safe-ify.yaml` created, permission enforcement tested |
| S3 | T15 | Agent commands work against Coolify, integration tests pass |
| S4 | T20 | `doctor` outputs CLAUDE.md snippet, audit logging works |

## Out of scope

- Environment variable management
- MCP server integration
- Creating/deleting Coolify applications
- Managing server settings or databases
- Homebrew/goreleaser distribution
- Client-side deploy queuing
