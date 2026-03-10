# PLN-2026-001 Functionele Uitwerking

## 1. User Personas

### Human Operator
Developer who self-hosts Coolify and wants to let AI coding agents deploy and monitor applications safely. Interacts with `auth` and `init` commands via interactive TUI prompts. Never needs `--json` output.

### AI Coding Agent (e.g., Claude Code)
Automated agent that invokes safe-ify commands non-interactively. Requires `--json` output for machine parsing. Must never see raw API tokens or access destructive endpoints.

---

## 2. User Flows

### 2.1 First-time Setup (Human)

1. Human installs safe-ify via `go install` or `make build`.
2. Runs `safe-ify auth add`.
3. TUI form prompts for: instance name, Coolify URL, API token.
4. Tool validates the token by calling Coolify `/api/v1/healthcheck`.
5. On success: stores instance in `~/.config/safe-ify/config.yaml` (file permissions 0600).
6. On failure: shows error, does not store.

### 2.2 Managing Instances (Human)

- `safe-ify auth list` -- shows table of configured instances (name, URL, masked token).
- `safe-ify auth remove` -- TUI picker to select and confirm removal of an instance.

### 2.3 Project Linking (Human)

1. In a project repo root, human runs `safe-ify init`.
2. TUI prompts: select instance (from global config), select application (fetched from Coolify API), configure permission overrides.
3. Creates `.safe-ify.yaml` in repo root with: instance name reference, app/service UUID, permission overrides.
4. This file is safe to commit -- no secrets.

### 2.4 Agent Deployment (Agent)

1. Agent runs `safe-ify deploy --json`.
2. Tool loads `.safe-ify.yaml` from current directory (or parent traversal).
3. Resolves instance credentials from global config.
4. Checks permissions: is `deploy` allowed for this project?
5. If allowed: calls Coolify API `POST /api/v1/deploy` with `uuid={uuid}`.
6. Returns JSON:
```json
{
  "ok": true,
  "data": {
    "message": "Deployment queued.",
    "deployment_uuid": "dep-uuid-123"
  },
  "error": null
}
```
Or on error:
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

### 2.5 Agent Log Retrieval (Agent)

1. Agent runs `safe-ify logs --json --tail 100`.
2. Permission check: is `logs` allowed?
3. Calls `GET /api/v1/applications/{uuid}/logs?tail=100`.
4. Returns JSON:
```json
{
  "ok": true,
  "data": {
    "lines": ["log line 1", "log line 2"],
    "count": 2
  },
  "error": null
}
```

### 2.6 Agent Status Check (Agent)

1. Agent runs `safe-ify status --json`.
2. Calls `GET /api/v1/applications/{uuid}` and extracts status fields.
3. Returns JSON:
```json
{
  "ok": true,
  "data": {
    "uuid": "app-uuid",
    "name": "my-app",
    "status": "running",
    "fqdn": "https://app.example.com",
    "last_deployment": "2026-03-10T12:00:00Z"
  },
  "error": null
}
```

### 2.7 Doctor (Human or Agent)

1. Runs `safe-ify doctor`.
2. Validates: global config exists, instance reachable, project config valid, permissions resolved.
3. Outputs a markdown snippet suitable for appending to CLAUDE.md, listing available/permitted commands.

---

## 3. Permission Model

### 3.1 Global Permissions (Default)

All agent-facing commands are allowed by default in global config:
- `deploy`, `redeploy`, `logs`, `status`, `list`

### 3.2 Project-Level Overrides

`.safe-ify.yaml` can specify a `permissions` block that **restricts** the global set:

```yaml
instance: my-coolify
app_uuid: "abc-123"
permissions:
  deny:
    - deploy
    - redeploy
```

This means: for this project, the agent can only use `logs`, `status`, `list`.

### 3.3 Enforcement Rules

- Project permissions are always a **subset** of global permissions.
- A project `permissions.deny` list removes commands from the allowed set.
- There is no `allow` key at project level -- you can only deny.
- If global config denies a command, project config cannot re-enable it.
- Attempting a denied command returns a clear error: `"error": "command 'deploy' is not permitted for this project"`.

---

## 4. Config File Formats

### 4.1 Global Config (`~/.config/safe-ify/config.yaml`)

```yaml
instances:
  my-coolify:
    url: "https://coolify.example.com"
    token: "<your-coolify-token>"
defaults:
  permissions:
    deny: []  # nothing denied globally by default
```

File permissions: 0600 (owner read/write only). Tool refuses to read if permissions are too open.

### 4.2 Project Config (`.safe-ify.yaml`)

```yaml
instance: my-coolify
app_uuid: "hgkks00"
permissions:
  deny:
    - deploy
    - redeploy
```

No secrets. Safe to commit.

---

## 5. Command Reference

### 5.1 Human-Facing (Interactive TUI)

| Command | Description | Interactive |
|---------|-------------|-------------|
| `safe-ify auth add` | Add Coolify instance | Yes -- TUI form |
| `safe-ify auth remove` | Remove instance | Yes -- TUI picker |
| `safe-ify auth list` | List instances | No -- table output |
| `safe-ify init` | Link project to instance/app | Yes -- TUI form |

### 5.2 Agent-Facing (Non-Interactive)

| Command | Description | Flags |
|---------|-------------|-------|
| `safe-ify deploy` | Trigger deployment | `--json`, `--force` |
| `safe-ify redeploy` | Redeploy current version | `--json` |
| `safe-ify logs` | Fetch logs | `--json`, `--tail N` |
| `safe-ify status` | Check deployment status | `--json` |
| `safe-ify list` | List applications | `--json` |

All agent commands require a project config (`.safe-ify.yaml`) and respect project-level permissions. All output JSON when `--json` is passed; without `--json`, they output human-readable text. All output uses the standard `{ok, data, error}` envelope (see D6).

### 5.3 Utility

| Command | Description |
|---------|-------------|
| `safe-ify doctor` | Validate setup, output CLAUDE.md snippet |

---

## 6. Error Handling

| Scenario | Behaviour |
|----------|-----------|
| No global config | Error: "No configuration found. Run `safe-ify auth add` first." |
| Global config insecure permissions | Error: "Config file permissions too open. Expected 0600." |
| No `.safe-ify.yaml` in project | Error: "No project config found. Run `safe-ify init` in your project root." |
| Instance not found in global config | Error: "Instance 'X' not found. Run `safe-ify auth list` to see configured instances." |
| Permission denied | Error: "Command 'deploy' is not permitted for this project." |
| Coolify API error | Pass through status code and message from Coolify API |
| Network error | Error: "Cannot reach Coolify instance at URL. Check connectivity." |

---

## 7. Audit Logging

- All agent-facing command invocations are logged to `~/.config/safe-ify/audit.log`.
- Log format: `YYYY-MM-DDTHH:MM:SSZ | command | app_uuid | instance | result | duration_ms`
- Log file is append-only; no rotation in v1.
- Human-facing commands (auth, init) are NOT logged.
