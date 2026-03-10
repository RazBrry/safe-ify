# PLN-2026-001 Tech Spec Developer: Config & Permissions

## 1. Config Loading

### 1.1 Global Config Path Resolution

1. Check `--config` flag.
2. Check `SAFE_IFY_CONFIG` environment variable.
3. Default: `~/.config/safe-ify/config.yaml`.

### 1.2 Global Config Security Check

Before reading the file:
1. `os.Stat()` the file.
2. Check `file.Mode().Perm()`.
3. If permissions are more open than `0600`, return `ConfigInsecureError`.
4. Also check directory permissions: must be no more open than `0700`.

```go
func CheckPermissions(path string) error {
    info, err := os.Stat(path)
    if err != nil {
        return fmt.Errorf("config not found: %w", err)
    }
    if info.Mode().Perm() & 0077 != 0 {
        return &ConfigInsecureError{
            Path:     path,
            Current:  info.Mode().Perm(),
            Expected: os.FileMode(0600),
        }
    }
    return nil
}
```

### 1.3 Global Config Loading

```go
func LoadGlobal(path string) (*GlobalConfig, error) {
    if err := CheckPermissions(path); err != nil {
        return nil, err
    }
    data, err := os.ReadFile(path)
    // ... yaml.Unmarshal into GlobalConfig
}
```

### 1.4 Global Config Saving

```go
func SaveGlobal(path string, cfg *GlobalConfig) error {
    data, _ := yaml.Marshal(cfg)
    dir := filepath.Dir(path)
    os.MkdirAll(dir, 0700)
    return os.WriteFile(path, data, 0600)
}
```

### 1.5 Project Config Path Resolution

1. Check `--project` flag.
2. Search current directory for `.safe-ify.yaml`.
3. Traverse parent directories until found or root reached.
4. If not found: return `ProjectConfigNotFoundError`.

### 1.6 Project Config Loading

```go
func LoadProject(path string) (*ProjectConfig, error) {
    data, err := os.ReadFile(path)
    // ... yaml.Unmarshal into ProjectConfig
    // Validate: instance name not empty, app_uuid not empty
}
```

### 1.7 Project Config Saving

```go
func SaveProject(path string, cfg *ProjectConfig) error {
    data, _ := yaml.Marshal(cfg)
    return os.WriteFile(path, data, 0644)  // no secrets, safe to be readable
}
```

---

## 2. Permission Enforcement

### 2.1 All Agent Commands

The complete set of agent-facing commands that can be controlled:

```go
var AllAgentCommands = []string{
    "deploy",
    "redeploy",
    "logs",
    "status",
    "list",
}
```

### 2.2 Permission Resolution Algorithm

```go
func ResolvePermissions(global GlobalConfig, project ProjectConfig) map[string]bool {
    allowed := make(map[string]bool)

    // Start: all commands allowed
    for _, cmd := range AllAgentCommands {
        allowed[cmd] = true
    }

    // Apply global denials
    for _, cmd := range global.Defaults.Permissions.Deny {
        allowed[cmd] = false
    }

    // Apply project denials (can only remove, never add)
    for _, cmd := range project.Permissions.Deny {
        allowed[cmd] = false
    }

    return allowed
}
```

### 2.3 Enforcement Check

```go
func (e *Enforcer) Check(command string) error {
    if !e.allowed[command] {
        denied := e.deniedBy(command)  // "global" or "project"
        return &PermissionDeniedError{
            Command:  command,
            DeniedBy: denied,
        }
    }
    return nil
}
```

### 2.4 Key Invariant

Project permissions can NEVER escalate beyond global. This is enforced by the algorithm: project denials are applied AFTER global denials, and there is no "allow" mechanism at the project level.

Proof: if global denies "deploy", the `allowed["deploy"]` is already `false`. Project config has no way to set it back to `true`.

### 2.5 Validation

On `safe-ify init`, validate that deny list only contains valid command names:

```go
func ValidateDenyList(deny []string) error {
    valid := map[string]bool{ /* AllAgentCommands */ }
    for _, cmd := range deny {
        if !valid[cmd] {
            return fmt.Errorf("unknown command in deny list: %q", cmd)
        }
    }
    return nil
}
```

---

## 3. Runtime Config Resolution

```go
func ResolveRuntime(global *GlobalConfig, project *ProjectConfig) (*RuntimeConfig, error) {
    inst, ok := global.Instances[project.Instance]
    if !ok {
        return nil, &InstanceNotFoundError{Name: project.Instance}
    }

    allowed := ResolvePermissions(*global, *project)

    return &RuntimeConfig{
        InstanceName: project.Instance,
        InstanceURL:  inst.URL,
        Token:        inst.Token,
        AppUUID:      project.AppUUID,
        AllowedCmds:  allowed,
    }, nil
}
```

---

## 4. Config Types (YAML Tags)

### 4.1 Global Config YAML

```yaml
# ~/.config/safe-ify/config.yaml
instances:
  <name>:                    # string key, alphanumeric + hyphens
    url: <string>            # full URL including protocol
    token: <string>          # Coolify API token
defaults:
  permissions:
    deny: [<string>, ...]    # list of agent commands to deny globally
```

### 4.2 Project Config YAML

```yaml
# .safe-ify.yaml
instance: <string>           # must match a key in global instances
app_uuid: <string>           # Coolify application UUID
permissions:
  deny: [<string>, ...]      # list of agent commands to deny for this project
```

---

## 5. Error Types

```go
type ConfigInsecureError struct {
    Path     string
    Current  os.FileMode
    Expected os.FileMode
}

type ConfigNotFoundError struct {
    Path string
}

type ProjectConfigNotFoundError struct {
    SearchRoot string
}

type InstanceNotFoundError struct {
    Name string
}

type PermissionDeniedError struct {
    Command  string
    DeniedBy string  // "global" or "project"
}
```

All error types implement `error` interface and provide user-friendly messages.
