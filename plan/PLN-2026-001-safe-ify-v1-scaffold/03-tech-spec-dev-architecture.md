# PLN-2026-001 Tech Spec Developer: Architecture

## 1. Project Structure

```
safe-ify/
  cmd/
    safe-ify/
      main.go                 # Entrypoint, Cobra root command
  internal/
    cli/
      root.go                 # Root command setup, global flags
      auth.go                 # auth add/remove/list commands
      init.go                 # init command
      deploy.go               # deploy command
      redeploy.go             # redeploy command
      logs.go                 # logs command
      status.go               # status command
      list.go                 # list command
      doctor.go               # doctor command
      output.go               # JSON/text output helpers
    config/
      global.go               # Global config loading/saving
      project.go              # Project config loading/saving
      types.go                # Config struct types
    permissions/
      enforcer.go             # Permission enforcement logic
      types.go                # Permission types and constants
    coolify/
      client.go               # HTTP client for Coolify API
      applications.go         # Application-related API calls
      deploy.go               # Deploy/redeploy API calls
      types.go                # API request/response types
    audit/
      logger.go               # Audit log writer
      types.go                # Audit log entry type
    tui/
      forms.go                # Charm huh form builders
      styles.go               # Lipgloss style definitions
  Makefile
  go.mod
  go.sum
  .safe-ify.yaml              # Example project config (gitignored in safe-ify's own repo)
```

## 2. Module Dependencies

```
cmd/safe-ify/main.go
  -> internal/cli (Cobra commands)
       -> internal/config (load/save configs)
       -> internal/permissions (enforce permissions)
       -> internal/coolify (API calls)
       -> internal/audit (log actions)
       -> internal/tui (interactive forms)
```

No circular dependencies. Each package depends only on packages below it in this hierarchy.

## 3. Data Model

### 3.1 Global Config (`GlobalConfig`)

```go
type GlobalConfig struct {
    Instances map[string]Instance `yaml:"instances"`
    Defaults  DefaultSettings     `yaml:"defaults"`
}

type Instance struct {
    URL   string `yaml:"url"`
    Token string `yaml:"token"`
}

type DefaultSettings struct {
    Permissions PermissionConfig `yaml:"permissions"`
}

type PermissionConfig struct {
    Deny []string `yaml:"deny"`
}
```

### 3.2 Project Config (`ProjectConfig`)

```go
type ProjectConfig struct {
    Instance    string           `yaml:"instance"`
    AppUUID     string           `yaml:"app_uuid"`
    Permissions PermissionConfig `yaml:"permissions"`
}
```

### 3.3 Resolved Runtime Config

```go
type RuntimeConfig struct {
    InstanceName string
    InstanceURL  string
    Token        string
    AppUUID      string
    AllowedCmds  map[string]bool  // computed from global + project deny lists
}
```

## 4. Coolify API Client

### 4.1 Client Interface

```go
type Client interface {
    Healthcheck(ctx context.Context) error
    ListApplications(ctx context.Context) ([]Application, error)
    GetApplication(ctx context.Context, uuid string) (*Application, error)
    Deploy(ctx context.Context, uuid string, force bool) (*DeployResponse, error)
    Restart(ctx context.Context, uuid string) (*DeployResponse, error)
    GetLogs(ctx context.Context, uuid string, tail int) (*LogsResponse, error)
}
```

### 4.2 HTTP Implementation

- Uses Go stdlib `net/http` with a custom `http.Client`.
- Base URL from instance config, all paths prefixed with `/api/v1`.
- Bearer token set in `Authorization` header.
- Timeout: 30 seconds for API calls, 60 seconds for log streaming.
- User-Agent: `safe-ify/1.0`.

### 4.3 Error Handling

API errors are wrapped in a `CoolifyError` type:

```go
type CoolifyError struct {
    StatusCode int
    Message    string
    Raw        string
}
```

HTTP 429 (rate limit): surface `Retry-After` header value in error message.
HTTP 401: suggest re-running `safe-ify auth add`.
HTTP 404: suggest checking app UUID.

## 5. Output Format

### 5.1 JSON Mode (`--json`)

All agent-facing commands return a consistent JSON envelope:

```json
{
  "ok": true,
  "data": { ... },
  "error": null
}
```

On error:

```json
{
  "ok": false,
  "data": null,
  "error": {
    "code": "PERMISSION_DENIED",
    "message": "Command 'deploy' is not permitted for this project."
  }
}
```

Error codes: `PERMISSION_DENIED`, `CONFIG_NOT_FOUND`, `INSTANCE_NOT_FOUND`, `API_ERROR`, `NETWORK_ERROR`, `CONFIG_INSECURE`.

### 5.2 Text Mode (default)

Human-readable formatted output using lipgloss styling.

## 6. Config File Security

- Global config directory: `~/.config/safe-ify/`
- Directory created with 0700 permissions.
- Config file created with 0600 permissions.
- On every load: check file permissions. If more permissive than 0600, refuse to read and show error.
- Token is never logged, never printed (masked in `auth list`), never included in JSON output.
