# PLN-2026-002 Task Board

| # | Slice | Task | Role | Model | Input files | Output | AC | Status |
|---|-------|------|------|-------|-------------|--------|----|--------|
| T1 | S1 | Implement config types, loading, and runtime resolution | Implementer | Sonnet | CONTEXT.md, 03-arch, 04-config-migration | internal/config/types.go, project.go, runtime.go, errors.go | AC: multi-app load, legacy compat, runtime resolve, error types | Pending |
| T2 | S1 | Implement permissions enforcer update | Implementer | Sonnet | CONTEXT.md, 03-arch | internal/permissions/enforcer.go | AC: NewEnforcer accepts app deny, three-layer merge works | Pending |
| T3 | S1 | Review S1 implementation | Reviewer | Opus | CONTEXT.md, 03-arch, 04-config-migration, T1+T2 output | verdict | AC: no blocking issues | Pending |
| T4 | S1 | Test config + permissions changes | Tester | Sonnet | CONTEXT.md, 05-ops, T1+T2 output | internal/config/*_test.go, internal/permissions/enforcer_test.go | AC: all tests pass, covers multi-app + legacy + errors + 3-layer deny | Pending |
| T5 | S1 | [GATE] Slice S1 approval | -- | -- | -- | -- | User says GO | Pending |
| T6 | S2 | Implement --app flag, agent.go changes, audit entry update | Implementer | Sonnet | CONTEXT.md, 04-cli-commands | internal/cli/root.go, agent.go, output.go, internal/audit/types.go, logger.go | AC: --app flag registered, resolveAgentConfig passes app, audit includes AppName | Pending |
| T7 | S2 | Implement init command multi-app support | Implementer | Sonnet | CONTEXT.md, 04-cli-commands | internal/cli/init.go, internal/tui/forms.go | AC: init creates multi-app config, add-app flow works | Pending |
| T8 | S2 | Implement doctor command multi-app support | Implementer | Sonnet | CONTEXT.md, 04-cli-commands | internal/cli/doctor.go | AC: doctor iterates all apps, CLAUDE.md snippet lists all | Pending |
| T9 | S2 | Review S2 implementation | Reviewer | Opus | CONTEXT.md, 04-cli-commands, T6+T7+T8 output | verdict | AC: no blocking issues | Pending |
| T10 | S2 | Test CLI + audit changes | Tester | Sonnet | CONTEXT.md, 05-ops, T6+T7+T8 output | internal/cli/*_test.go, internal/audit/*_test.go | AC: all tests pass, covers --app flag, ambiguous/not-found errors, list bypass, audit format | Pending |
| T11 | S2 | Run full test suite and verify backward compat | Tester | Sonnet | CONTEXT.md, 05-ops | -- | AC: `go test ./...` passes, legacy config test passes | Pending |
| T12 | S2 | [GATE] Slice S2 approval | -- | -- | -- | -- | User says GO | Pending |
