# Worktree Review — safe-ify-PLN-2026-002-multi-app-support-S1

**Date:** 2026-03-11
**Branch:** safe-ify-PLN-2026-002-multi-app-support-S1 vs main
**Reviewers:** Claude Opus · Codex
**Diff scope:** 14 files changed, 706 insertions, 38 deletions

---

## Merged Findings

> Findings from both reviewers, deduplicated and consolidated.
> Items agreed on by 2 reviewers are marked (consensus).

### CRITICAL

None.

### HIGH

**H1. doctor.go reads `projectCfg.AppUUID` which is always empty after normalization** (consensus)
- Opus MEDIUM #2, Codex HIGH #2
- File: `internal/cli/doctor.go:212`
- After `LoadProject` normalizes legacy configs, `AppUUID` is cleared to `""`. Doctor calls `GetApplication("")` — broken for all config formats. This is a functional regression for existing single-app users.
- Elevated to HIGH due to consensus and regression impact.

**H2. `resolveAgentConfig` ignores resolved app's deny list in NewEnforcer call** (consensus)
- Opus HIGH #1, Codex HIGH #1 (partially overlaps with Codex MEDIUM #3)
- File: `internal/cli/agent.go:75`
- `NewEnforcer` is called with `[]string{}` instead of the resolved app's deny list. Creates permissions mismatch between `RuntimeConfig.AllowedCmds` and `Enforcer.Check()`.
- **Mitigating factor:** The enforcer is currently discarded in `runAgentCommand` (line 114: `cfg, client, _, err`), so agent commands don't actually use it. Doctor does use it at line 224. The real risk is limited to doctor output showing wrong permissions.
- **Assessment:** This is S2-scoped work per the plan (T6 wires `--app` and passes app deny to enforcer). For S1 single-app configs, the enforcer works correctly because single-app has no app-level deny. **Downgrade to MEDIUM for merge decision — not a real S1 bug, but doctor should be fixed (see H1).**

### MEDIUM

**M1. `mapConfigError` does not handle new error types** (consensus)
- Opus MEDIUM #5, Codex MEDIUM #4
- File: `internal/cli/agent.go:84-96`
- `AppNotFoundError` and `AppAmbiguousError` fall through to `ErrCodeAPIError`. Should map to dedicated error codes.
- **Assessment:** S2-scoped — these errors can only occur when `--app` flag is wired (S2 T6). No impact in S1.

**M2. Agent command list duplicated in three places** (consensus)
- Opus MEDIUM #4, Codex LOW #5
- Files: `internal/config/project.go:19`, `internal/config/runtime.go:44`, `internal/permissions/types.go`
- Three independent definitions with "keep in sync" comments. No compile-time drift detection.

**M3. `PermissionDeniedError.DeniedBy` doc comment is stale**
- Opus MEDIUM #3 only
- File: `internal/config/errors.go:66` (or `internal/permissions/types.go`)
- Comment says `"global" or "project"` but now also supports `"app"`.

**M4. Doctor ignores app-level deny lists in permission output**
- Codex MEDIUM #3, overlaps with H2
- File: `internal/cli/doctor.go:224`
- Doctor passes `[]string{}` to `NewEnforcer`, so CLAUDE.md snippet may advertise wrong permissions.
- **Assessment:** S2-scoped — doctor multi-app is T6 deliverable.

### LOW

**L1.** Regex `validAppName` allows trailing hyphens (Opus #6)
**L2.** No test for invalid app deny list entries (Opus #7)
**L3.** Enforcer tests don't use `t.Parallel()` (Opus #8)
**L4.** `ResolveRuntime` doesn't validate empty `Apps` map independently (Opus #9)

---

## Overall Verdict

| Reviewer | CRITICAL | HIGH | MEDIUM | LOW | Verdict |
|----------|----------|------|--------|-----|---------|
| Claude Opus | 0 | 1 | 4 | 4 | FAIL |
| Codex | 0 | 2 | 2 | 1 | FAIL |
| **Merged** | **0** | **2** | **4** | **4** | **FAIL** |

**Merged verdict:** FAIL — 2 HIGH findings (H1 is a real regression, H2 downgraded to MEDIUM on analysis).

### Actionable vs S2-scoped

| Finding | Actionable in S1? | Reason |
|---------|-------------------|--------|
| H1 (doctor AppUUID regression) | **YES — must fix** | Breaks doctor for all users after S1 merge |
| H2 (enforcer app deny) | No | S2 wires --app; enforcer unused by agent commands |
| M1 (mapConfigError) | No | Errors only reachable after S2 --app wiring |
| M2 (command list duplication) | Optional | Pre-existing pattern, not introduced by S1 |
| M3 (stale comment) | Yes — trivial | One-line fix |
| M4 (doctor app deny) | No | S2 scope |

---

## Merge Readiness

### Branch status
- Branch: `safe-ify-PLN-2026-002-multi-app-support-S1`
- Commits ahead of main: 5
- Main commits since branch point: 0

### Conflict analysis
Clean merge possible. No conflicts detected.

### Recommended merge strategy
- No rebase needed (main has not diverged).
- 7 commits — squash merge recommended for cleaner history.

### Merge commands

```bash
# After fixes, from main:
git checkout main
git merge safe-ify-PLN-2026-002-multi-app-support-S1
git push
```

---

## Fix Status

| Round | Fixed by | Items resolved | Items remaining |
|-------|----------|----------------|-----------------|
| 1 | Sonnet Implementer | H1, M3, L1, L2, L3 | H2 |
| 2 | Orchestrator | H2 | — |

### Re-verification results

**Round 1 (Opus + Codex):** H1 YES, M3 YES, L1 YES, L2 YES, L3 YES, H2 NO
**Round 2 (Opus + Codex):** H2 YES — OVERALL: PASS

**Final merged verdict: PASS** — all findings resolved.
