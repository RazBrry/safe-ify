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

## Deployment

No infrastructure changes. This is a CLI tool distributed as a binary. Users update by rebuilding (`make build`) or `go install`.

## Migration

No explicit migration step. The old config format is auto-detected and works as-is (D2). If a user runs `safe-ify init` on an existing legacy config and adds an app, the config is saved in the new format.

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
