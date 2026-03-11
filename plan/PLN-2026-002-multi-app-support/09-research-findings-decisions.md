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
