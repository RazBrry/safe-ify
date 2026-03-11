# PLN-2026-002 — Tech Spec Developer: Architecture

## Data Model

### New config types (`internal/config/types.go`)

```go
// AppConfig holds the configuration for a single named app within a project.
type AppConfig struct {
    UUID        string           `yaml:"uuid"`
    Permissions PermissionConfig `yaml:"permissions"`
}

// ProjectConfig represents a per-project config stored in .safe-ify.yaml.
// Supports two formats:
//   - Legacy: Instance + AppUUID + Permissions (single app)
//   - Multi:  Instance + Apps map + Permissions (project-level deny)
type ProjectConfig struct {
    Instance    string               `yaml:"instance"`
    AppUUID     string               `yaml:"app_uuid,omitempty"`     // legacy single-app
    Apps        map[string]AppConfig `yaml:"apps,omitempty"`         // multi-app
    Permissions PermissionConfig     `yaml:"permissions"`            // project-level deny
}
```

### RuntimeConfig changes

```go
type RuntimeConfig struct {
    InstanceName string
    InstanceURL  string
    Token        string
    AppUUID      string
    AppName      string          // NEW: the selected app's name
    AllowedCmds  map[string]bool // computed from global + project + app deny lists
}
```

### Audit Entry changes

```go
type Entry struct {
    Timestamp  time.Time
    Command    string
    AppUUID    string
    AppName    string  // NEW
    Instance   string
    Result     string
    DurationMs int64
}
```

## Config Resolution Flow

```
LoadProject(path)
  |
  ├── Has app_uuid? (legacy) ──> Normalize to single-entry Apps map, key = "default"
  ├── Has apps? (multi) ──────> Use as-is
  └── Has both? ──────────────> Return error (D5)
  |
  v
ResolveRuntime(global, project, appName)
  |
  ├── Look up appName in project.Apps
  ├── If appName == "" and len(Apps) == 1 ──> use the only app
  ├── If appName == "" and len(Apps) > 1 ──> return AppAmbiguousError
  ├── If appName not found ────────────────> return AppNotFoundError
  |
  v
  Merge deny lists: global.Deny ∪ project.Deny ∪ app.Deny
  |
  v
  RuntimeConfig{AppUUID, AppName, AllowedCmds, ...}
```

## New Error Types (`internal/config/errors.go`)

```go
type AppNotFoundError struct {
    Name           string
    AvailableApps  []string
}

type AppAmbiguousError struct {
    AvailableApps []string
}
```

## Permission Enforcement Changes

`permissions.NewEnforcer` currently takes `(global, project)`. It needs a third parameter for the app-level deny list:

```go
func NewEnforcer(global GlobalConfig, project ProjectConfig, appDeny []string) *Enforcer
```

The merge order: global deny -> project deny -> app deny. Each layer can only restrict further.

## Package Dependency

No new packages. No new external dependencies. All changes are within existing packages:
- `internal/config` — types, loading, runtime resolution
- `internal/permissions` — enforcer signature
- `internal/audit` — entry type
- `internal/cli` — flag handling, init, doctor
- `internal/tui` — new form for app name input
