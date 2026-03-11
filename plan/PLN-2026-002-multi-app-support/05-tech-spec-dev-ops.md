# PLN-2026-002 — Tech Spec Developer: Ops, Testing & DoR/DoD

## Test Strategy

### Unit Tests

#### Config layer (`internal/config/`)

| Test | What it verifies |
|------|-----------------|
| `TestLoadProject_MultiApp` | Multi-app YAML loads correctly, Apps map populated, AppUUID cleared |
| `TestLoadProject_Legacy` | Legacy single-app YAML loads and normalizes to Apps map with key "default" |
| `TestLoadProject_BothFormats` | Config with both `app_uuid` and `apps` returns error (D5) |
| `TestLoadProject_EmptyApps` | Config with empty `apps:` map returns error |
| `TestLoadProject_InvalidAppName` | App name with invalid characters returns error |
| `TestLoadProject_MissingAppUUID` | App entry with empty `uuid` returns error |
| `TestSaveProject_MultiApp` | Multi-app config serializes correctly, no `app_uuid` field in output |
| `TestResolveRuntime_SingleApp_NoFlag` | Single app, empty `--app` flag, resolves to the only app |
| `TestResolveRuntime_MultiApp_WithFlag` | Multiple apps, `--app=api`, resolves correctly |
| `TestResolveRuntime_MultiApp_NoFlag` | Multiple apps, empty `--app`, returns AppAmbiguousError |
| `TestResolveRuntime_AppNotFound` | `--app=unknown`, returns AppNotFoundError |
| `TestResolveRuntime_ThreeLayerDeny` | Global deny + project deny + app deny merged correctly |

#### Permissions layer (`internal/permissions/`)

| Test | What it verifies |
|------|-----------------|
| `TestEnforcer_WithAppDeny` | NewEnforcer with app deny list restricts additional commands |
| `TestEnforcer_AppDenyDoesNotEscalate` | App deny cannot un-deny something denied at project/global level |

#### Audit layer (`internal/audit/`)

| Test | What it verifies |
|------|-----------------|
| `TestEntry_StringWithAppName` | Entry.String() includes app name in output |
| `TestEntry_StringEmptyAppName` | Entry.String() handles empty AppName gracefully |

### Integration Tests (CLI layer)

These tests use Cobra's test mode (command execution in-process with captured output).

| Test | What it verifies |
|------|-----------------|
| `TestDeployCmd_WithAppFlag` | `--app` flag passed through, correct UUID used |
| `TestDeployCmd_AmbiguousApp` | Multi-app config without `--app` returns APP_AMBIGUOUS JSON error |
| `TestDeployCmd_AppNotFound` | `--app=unknown` returns APP_NOT_FOUND JSON error |
| `TestListCmd_NoAppRequired` | `list` works without `--app` even with multi-app config |
| `TestDoctorCmd_MultiApp` | Doctor output includes all apps |

### Existing Tests

All existing tests must continue to pass. The legacy format normalization ensures backward compatibility at the config layer. Existing CLI tests use the legacy format and will exercise the normalization path.

### Coverage Targets

- **Minimum requirement:** Every new public function and method must have at least one unit test.
- **Named tests:** All tests listed in the Unit Tests and Integration Tests tables above are required. The total is 19 new test functions (12 config, 2 permissions, 2 audit, 5 CLI integration).
- **Branch coverage:** Each error path (AppNotFoundError, AppAmbiguousError, both-format rejection, empty apps, invalid name, missing UUID) must have a dedicated test case.
- **Regression:** `go test ./...` must pass with 0 failures. No existing test may be removed or weakened.

## Deployment

No infrastructure changes. This is a CLI tool distributed as a binary. Users update by rebuilding (`make build`) or `go install`.

## Migration

No explicit migration step. The old config format is auto-detected and works as-is (D2). If a user runs `safe-ify init` on an existing legacy config and adds an app, the config is saved in the new format.

## Monitoring & Observability

### Audit log format change (D4)

The audit log gains a new `AppName` field, changing the pipe-delimited format from:

```
timestamp | command | app_uuid | instance | result | duration_ms
```

to:

```
timestamp | command | app_name | app_uuid | instance | result | duration_ms
```

**Implications:**
- The audit log is append-only and human-read. There are no programmatic parsers to break.
- Existing log entries (written before this change) will not have the `app_name` field. Any future tooling that parses the audit log must handle both formats (6-field and 7-field lines).
- No alerting or monitoring infrastructure exists for this CLI tool. The audit log itself is the observability mechanism.
- No action required beyond documenting the format change. Users reading old log entries will see the previous format; new entries will include the app name.

## Definition of Ready (Slice)

- [ ] All spec files for the slice are reviewed and approved
- [ ] No open questions blocking the slice
- [ ] Preceding slices (if any) are merged to main

## Definition of Done (Slice)

- [ ] All tasks in the slice are Done
- [ ] All tests pass (`go test ./...`)
- [ ] No lint errors (`make lint`)
- [ ] Code reviewed by Reviewer sub-agent
- [ ] Tests written and passing by Tester sub-agent
- [ ] Backward compatibility verified (existing single-app configs work)
