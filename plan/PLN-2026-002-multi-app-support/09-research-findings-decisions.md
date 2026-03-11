# PLN-2026-002 — Research Findings & Decisions

## Decisions

### D1: Map-based app naming in config

**Date:** 2026-03-11
**Context:** Multiple apps need unique names in the project config. Need to decide between array, map, or flat structure.
**Options considered:**
1. Array of apps with explicit `name` field -- allows duplicate names, requires validation, more verbose
2. Map keyed by app name -- prevents duplicates via YAML semantics, direct lookup, concise
3. Flat `app_uuids` map -- too compact, no room for per-app settings

**Decision:** Option 2 (map keyed by app name). Consistent with the existing `instances` map in global config. YAML prevents duplicate keys. Direct O(1) lookup.
**Consequences:** App names must be valid YAML map keys. We enforce alphanumeric + hyphens for safety.

### D2: Backward compatibility via auto-detection

**Date:** 2026-03-11
**Context:** Existing users have `.safe-ify.yaml` with `app_uuid`. Upgrading safe-ify should not break their setup.
**Options considered:**
1. Auto-detect format by checking field presence -- zero user action, simple
2. Explicit version field -- requires editing old configs
3. Migration command -- extra step, old format eventually unsupported

**Decision:** Option 1 (auto-detect). Check for `app_uuid` (legacy) vs `apps` (new). Normalize legacy to multi-app internally. No migration needed.
**Consequences:** Loader has a small branching path. Internal representation is always multi-app. Legacy configs get key `"default"` when normalized.

### D3: Three-layer permission deny merge

**Date:** 2026-03-11
**Context:** Per-app deny lists need to integrate with existing global and project deny lists.
**Options considered:**
1. Three-layer union: global ∪ project ∪ app -- extends existing model naturally
2. App deny overrides project deny -- violates "never escalate" principle

**Decision:** Option 1 (three-layer union). Maintains the "can only restrict, never escalate" invariant.
**Consequences:** `NewEnforcer` gains a third parameter. `ResolveRuntime` must extract the app-specific deny list and pass it through.

### D4: App name in audit log

**Date:** 2026-03-11
**Context:** With multiple apps, the audit log needs to identify which app was targeted. Currently logs only the UUID.
**Options considered:**
1. Add `AppName` field to audit Entry -- simple, human-readable
2. Keep UUID only -- sufficient but harder to read

**Decision:** Option 1. Add `AppName` to the pipe-delimited log line between command and app_uuid.
**Consequences:** Log line format changes. Existing entries lack the field. Acceptable because the log is human-read, not machine-parsed.

### D5: Reject ambiguous config with both app_uuid and apps

**Date:** 2026-03-11
**Context:** What if a config file has both `app_uuid` and `apps`? Could happen from manual editing.
**Options considered:**
1. Prefer `apps`, ignore `app_uuid` -- silent data loss risk
2. Reject as invalid -- fail fast, clear error message

**Decision:** Option 2 (reject as invalid). Return a clear error: "config has both 'app_uuid' and 'apps'; use only one format."
**Consequences:** Users must clean up manually if they create this state. Unlikely to occur in practice.

### D6: list command bypasses --app requirement

**Date:** 2026-03-11
**Context:** The `list` command lists all applications on the Coolify instance. It does not target a specific app UUID.
**Options considered:**
1. Require `--app` for all commands including `list` -- consistent but unnecessary
2. Make `list` bypass the app requirement -- practical, since it does not use an app UUID

**Decision:** Option 2. `resolveAgentConfig` accepts an `appRequired` parameter. `list` passes `false`.
**Consequences:** `resolveAgentConfig` signature changes. Minor refactor in `agent.go`.

### D7: S2 consolidated into single implementer task

**Date:** 2026-03-11
**Context:** Plan review (Codex) found that S2 had three implementer tasks (T6, T7, T8) sharing one reviewer and one tester, violating the Implementer -> Reviewer -> Tester chain requirement per plan-process.md section 2b rule 3.
**Options considered:**
1. Give each implementer task its own reviewer and tester -- results in 9 tasks for 3 small deliverables, excessive overhead
2. Consolidate the three implementer tasks into one -- the CLI layer changes are tightly coupled (T7/T8 depend on agent.go changes from T6) and belong to the same package set

**Decision:** Option 2. Consolidate old T6 (--app flag + agent.go + audit), T7 (init), and T8 (doctor) into a single T6 task. This gives S2 a clean T6 (Implementer) -> T7 (Reviewer) -> T8 (Tester) -> T9 (Full suite tester) -> T10 (Gate) sequence.
**Consequences:** S2 has fewer, larger tasks. The single implementer task is comprehensive but all changes are in the same layer (internal/cli + internal/tui + internal/audit). Task file numbering from T6-T10 replaces the original T6-T12.

### D8: Legacy normalization uses empty app deny list

**Date:** 2026-03-11
**Context:** Plan review found T1 instruction step 3 contradicted 04-tech-spec-dev-config-migration.md. T1 normalized legacy config by copying `cfg.Permissions` (project-level deny) into the app entry, but the spec states the app gets an empty deny list.
**Options considered:**
1. Copy project deny into app deny -- double-applies the project deny during ResolveRuntime (once as project, once as app)
2. Use empty deny list for the app entry -- correct, since project-level deny is applied separately in ResolveRuntime

**Decision:** Option 2. The normalized "default" app entry gets `Permissions: PermissionConfig{Deny: []string{}}`. The project-level `cfg.Permissions` stays at the project level and is merged separately during `ResolveRuntime`.
**Consequences:** T1 instructions updated to match the spec exactly. No double-application of project deny list.
