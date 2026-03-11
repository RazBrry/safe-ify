# PLN-2026-002 Task Board

| # | Slice | Task | Role | Model | Input files | Output | AC | Status |
|---|-------|------|------|-------|-------------|--------|----|--------|
| T1 | S1 | Implement config types, loading, and runtime resolution | Implementer | Sonnet | CONTEXT.md, 03-arch, 04-config-migration | internal/config/types.go, project.go, runtime.go, errors.go | AC: multi-app load, legacy compat, runtime resolve, error types | Pending |
| T2 | S1 | Implement permissions enforcer update | Implementer | Sonnet | CONTEXT.md, 03-arch, internal/permissions/enforcer.go, internal/cli/agent.go, internal/cli/doctor.go | internal/permissions/enforcer.go, internal/cli/agent.go, internal/cli/doctor.go | AC: NewEnforcer accepts app deny, three-layer merge works, callers updated | Pending |
| T3 | S1 | Review S1 implementation | Reviewer | Opus | CONTEXT.md, 03-arch, 04-config-migration, T1+T2 output (incl. agent.go, doctor.go) | verdict | AC: no blocking issues | Pending |
| T4 | S1 | Test config + permissions changes | Tester | Sonnet | CONTEXT.md, 05-ops, T1+T2 output | internal/config/*_test.go, internal/permissions/enforcer_test.go | AC: all tests pass, covers multi-app + legacy + errors + 3-layer deny | Pending |
| T5 | S1 | [GATE] Slice S1 approval | -- | -- | -- | -- | User says GO | Pending |
| T6 | S2 | Implement S2 CLI layer: --app flag, agent.go, init, doctor, audit | Implementer | Sonnet | CONTEXT.md, 04-cli-commands, 03-arch | internal/cli/root.go, agent.go, output.go, init.go, doctor.go, internal/tui/forms.go, internal/audit/types.go | AC: --app flag, resolveAgentConfig, init multi-app, doctor multi-app, audit AppName | Pending |
| T7 | S2 | Review S2 implementation | Reviewer | Opus | CONTEXT.md, 04-cli-commands, 09-decisions, T6 output | verdict | AC: no blocking issues | Pending |
| T8 | S2 | Test CLI + audit changes | Tester | Sonnet | CONTEXT.md, 05-ops, T6 output | internal/cli/*_test.go, internal/audit/*_test.go | AC: all tests pass, covers --app flag, ambiguous/not-found errors, list bypass, audit format, init multi-app, doctor multi-app | Pending |
| T9 | S2 | Run full test suite, lint, and verify backward compat | Tester | Sonnet | CONTEXT.md, 05-ops, T6+T8 output | -- | AC: `go test ./...` passes, `go test -race ./...` passes, `make lint` passes, legacy config test passes | Pending |
| T10 | S2 | [GATE] Slice S2 approval | -- | -- | -- | -- | User says GO | Pending |
