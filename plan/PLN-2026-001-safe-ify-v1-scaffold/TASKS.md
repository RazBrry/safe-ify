# PLN-2026-001 Task Board

| # | Slice | Task | Role | Model | Input files | Output | AC | Status |
|---|-------|------|------|-------|-------------|--------|----|--------|
| T1 | S1 | Implement project scaffold + global config | Implementer | Sonnet | 03-arch, 04-config, 05-ops | go.mod, Makefile, cmd/, internal/config/, internal/cli/root.go | AC1: `go build ./...` succeeds; AC2: global config loads/saves with 0600 | Done |
| T2 | S1 | Implement auth add/remove/list commands | Implementer | Sonnet | 04-cli-commands, 04-config | internal/cli/auth.go, internal/tui/ | AC1: `auth add` stores instance via TUI; AC2: `auth list` shows masked tokens; AC3: `auth remove` deletes instance | Done |
| T3 | S1 | Review scaffold + auth (S1 code) | Code Quality Reviewer | Opus | T1+T2 output, 03-arch, 04-cli-commands | verdict | AC: no blocking issues | Done |
| T4 | S1 | Test scaffold + auth (S1 tests) | Tester | Sonnet | T1+T2 output, 05-ops | internal/config/*_test.go | AC: all unit tests pass, config permission tests pass | Done |
| T5 | S1 | [GATE] Slice S1 approval | -- | -- | -- | -- | User says GO | Pending |
| T6 | S2 | Implement permission enforcement | Implementer | Sonnet | 04-config-permissions | internal/permissions/ | AC1: deny-only model works; AC2: project cannot escalate beyond global | Pending |
| T7 | S2 | Implement init command | Implementer | Sonnet | 04-cli-commands, 04-config-permissions | internal/cli/init.go | AC1: TUI selects instance + app; AC2: writes valid `.safe-ify.yaml`; AC3: project config loads with parent traversal | Pending |
| T8 | S2 | Review permissions + init (S2 code) | Code Quality Reviewer | Opus | T6+T7 output, 04-config-permissions, 04-cli-commands | verdict | AC: no blocking issues | Pending |
| T9 | S2 | Test permissions + init (S2 tests) | Tester | Sonnet | T6+T7 output, 05-ops | internal/permissions/*_test.go | AC: all permission enforcement tests pass including escalation prevention | Pending |
| T10 | S2 | [GATE] Slice S2 approval | -- | -- | -- | -- | User says GO | Pending |
| T11 | S3 | Implement Coolify API client | Implementer | Sonnet | 03-arch, 08-api-matrix | internal/coolify/ | AC1: client calls all 5 endpoints; AC2: error handling for all HTTP status codes | Pending |
| T12 | S3 | Implement agent commands (deploy/redeploy/logs/status/list) | Implementer | Sonnet | 04-cli-commands, 03-arch | internal/cli/deploy.go, redeploy.go, logs.go, status.go, list.go, output.go | AC1: all 5 commands produce valid JSON envelope; AC2: permission check before API call | Pending |
| T13 | S3 | Review API client + agent commands (S3 code) | Code Quality Reviewer | Opus | T11+T12 output, 03-arch, 04-cli-commands, 08-api-matrix | verdict | AC: no blocking issues | Pending |
| T14 | S3 | Test API client + agent commands (S3 tests) | Tester | Sonnet | T11+T12 output, 05-ops | internal/coolify/*_test.go, internal/cli/*_test.go | AC: integration tests with mocked API pass for all 5 endpoints | Pending |
| T15 | S3 | [GATE] Slice S3 approval | -- | -- | -- | -- | User says GO | Pending |
| T16 | S4 | Implement audit logging | Implementer | Sonnet | 03-arch | internal/audit/ | AC1: agent commands write audit log entries; AC2: format matches spec | Pending |
| T17 | S4 | Implement doctor command | Implementer | Sonnet | 04-cli-commands | internal/cli/doctor.go | AC1: validates setup; AC2: outputs valid CLAUDE.md markdown snippet; AC3: exit code 0 on success, 1 on failure | Pending |
| T18 | S4 | Review audit + doctor (S4 code) | Code Quality Reviewer | Opus | T16+T17 output, 03-arch, 04-cli-commands | verdict | AC: no blocking issues | Pending |
| T19 | S4 | Test audit + doctor (S4 tests) | Tester | Sonnet | T16+T17 output, 05-ops | internal/audit/*_test.go, internal/cli/doctor_test.go | AC: audit log tests pass, doctor output format tests pass | Pending |
| T20 | S4 | [GATE] Slice S4 approval | -- | -- | -- | -- | User says GO | Pending |
