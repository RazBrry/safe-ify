# PLN-2026-001 Plan Overzicht: safe-ify v1 Scaffold

**Status:** Draft
**Owner:** Planner (Opus)
**Created:** 2026-03-10
**Updated:** 2026-03-10

---

## Summary

Build the complete scaffold and working v1 of **safe-ify**, a CLI tool written in Go that acts as a permissioned safety layer for coding agents (like Claude Code) to interact with Coolify v4 (self-hosted PaaS). The tool prevents agents from accessing destructive Coolify endpoints or raw credentials.

## Scope

### In scope

- Go project scaffold with Cobra CLI framework
- Two-layer config system: global (`~/.config/safe-ify/config.yaml`) and project (`.safe-ify.yaml`)
- Human-facing TUI commands: `auth add`, `auth remove`, `auth list`, `init`
- Agent-facing commands with `--json` output: `deploy`, `redeploy`, `logs`, `status`, `list`
- `doctor` command outputting CLAUDE.md snippet
- Permission model: project-level can only restrict, never escalate
- Audit logging of all agent actions
- Unit tests for permission enforcement and config loading
- Integration tests with mocked Coolify API
- Distribution via `go install` + `make build`

### Out of scope

- Environment variable management
- MCP server integration
- Creating/deleting Coolify applications
- Managing server settings or databases
- Homebrew/goreleaser distribution
- Client-side deploy queuing

## Acceptance Criteria

- [ ] AC1: `go build ./...` succeeds with zero errors
- [ ] AC2: `safe-ify auth add` interactively stores instance config at `~/.config/safe-ify/config.yaml` with 0600 permissions
- [ ] AC3: `safe-ify auth remove` and `safe-ify auth list` manage instances correctly
- [ ] AC4: `safe-ify init` creates `.safe-ify.yaml` linking project to a Coolify instance/app
- [ ] AC5: Permission enforcement: project-level permissions are always a subset of global permissions
- [ ] AC6: `safe-ify deploy --json`, `redeploy --json`, `logs --json`, `status --json`, `list --json` produce valid JSON output
- [ ] AC7: `safe-ify doctor` outputs a valid CLAUDE.md markdown snippet
- [ ] AC8: All agent actions are audit-logged locally
- [ ] AC9: Unit tests pass for permission enforcement and config loading
- [ ] AC10: Integration tests pass with mocked Coolify API
- [ ] AC11: Non-network CLI operations complete in under 100ms
- [ ] AC12: `make build` produces `./bin/safe-ify` binary

---

## Slice Map

| Slice | Title | Increment (what works after merge) | Tasks | Depends on |
|-------|-------|------------------------------------|-------|------------|
| S1 | Project scaffold + config system | `go build` works, `safe-ify auth add/remove/list` works with TUI, global config stored securely at `~/.config/safe-ify/config.yaml` with 0600, unit tests pass | T1--T5 | -- |
| S2 | Project config + permissions | `safe-ify init` works with TUI, `.safe-ify.yaml` created, permission enforcement works (project config restricts but never escalates), unit tests pass | T6--T10 | S1 |
| S3 | Coolify API client + operational commands | `safe-ify deploy/redeploy/logs/status/list --json` work against real Coolify, integration tests with mocked API pass | T11--T15 | S2 |
| S4 | Doctor + audit logging | `safe-ify doctor` outputs CLAUDE.md snippet, audit log records all agent actions, full test suite passes | T16--T20 | S3 |
