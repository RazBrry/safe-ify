# PLN-2026-001 Research: Coolify v4 API Matrix

## 1. API Fundamentals

| Property | Value |
|----------|-------|
| Base URL | `http(s)://<host>:8000/api/v1` |
| Auth method | Bearer token in `Authorization` header |
| Token source | Coolify UI: Keys & Tokens > API tokens |
| Token format | `<id>\|<random-string>` (e.g., `3\|sk-example-not-a-real-token...`) |
| Token scope | Team-scoped; can only access owning team's resources |
| API version | v1 (OpenAPI 3.1 spec) |
| Content-Type | `application/json` |

## 2. Permission Model (Token Abilities)

| Ability | Scope |
|---------|-------|
| `read` | Read-only resource access (default); sensitive data redacted |
| `read:sensitive` | Read access including passwords, API keys |
| `write` | Create, update, delete resources |
| `deploy` | Trigger deployments and restarts |
| `*` | Full access to all resources and sensitive information |

**Note for safe-ify:** The Coolify token used by safe-ify needs at minimum `read` + `deploy` abilities. The `write` ability is NOT needed (safe-ify never creates/deletes resources).

## 3. Endpoints Used by safe-ify

### 3.1 Health & Version

| Endpoint | Method | Path | Parameters | Response | safe-ify usage |
|----------|--------|------|------------|----------|----------------|
| Healthcheck | GET | `/api/v1/healthcheck` | none | `"OK"` (200) | `auth add` validation |
| Version | GET | `/api/v1/version` | none | `"4.0.0-beta.460"` (200) | `doctor` version check |

### 3.2 Applications

| Endpoint | Method | Path | Parameters | Response | safe-ify usage |
|----------|--------|------|------------|----------|----------------|
| List apps | GET | `/api/v1/applications` | `?tag=<string>` (optional) | Array of Application objects | `list` command, `init` picker |
| Get app | GET | `/api/v1/applications/{uuid}` | none | Full Application object | `status` command |
| Get logs | GET | `/api/v1/applications/{uuid}/logs` | `?tail=<int>` (optional), `?since=<timestamp>` (optional) | Log lines | `logs` command |

### 3.3 Deployment Actions

| Endpoint | Method | Path | Parameters | Response | safe-ify usage |
|----------|--------|------|------------|----------|----------------|
| Deploy | POST | `/api/v1/deploy` | `?uuid=<string>`, `?force=<bool>`, `?tag=<string>` | `{"deployments": [{"message": "...", "resource_uuid": "...", "deployment_uuid": "..."}]}` | `deploy` command |
| Restart | POST | `/api/v1/applications/{uuid}/restart` | none | Restart confirmation | `redeploy` command |

> **Note:** The Coolify API documentation may list GET as an accepted method for deploy and restart. safe-ify uses POST exclusively for these side-effecting operations per RESTful convention (see D11).

### 3.4 Deployment Tracking (informational)

| Endpoint | Method | Path | Parameters | Response | safe-ify usage |
|----------|--------|------|------------|----------|----------------|
| Get deployment | GET | `/api/v1/deployments/{uuid}` | none | Deployment details | Future: deployment status polling |
| List deployments | GET | `/api/v1/deployments` | none | Array of deployments | Not used in v1 |
| Cancel deployment | POST | `/api/v1/deployments/{uuid}/cancel` | none | Cancellation confirmation | Not used in v1 (destructive) |

## 4. Endpoints Explicitly NOT Used (safety boundary)

| Endpoint | Method | Path | Reason excluded |
|----------|--------|------|-----------------|
| Create app | POST | `/api/v1/applications/*` | Destructive: creates infrastructure |
| Delete app | DELETE | `/api/v1/applications/{uuid}` | Destructive: deletes application |
| Update app | PATCH | `/api/v1/applications/{uuid}` | Dangerous: could change config |
| Stop app | GET | `/api/v1/applications/{uuid}/stop` | Destructive: stops production |
| Start app | GET | `/api/v1/applications/{uuid}/start` | Overlaps with deploy; start resumes stopped apps |
| Env vars | GET/POST/PATCH/DELETE | `/api/v1/applications/{uuid}/envs*` | Out of scope; exposes secrets |
| Servers | * | `/api/v1/servers/*` | Out of scope; infrastructure management |
| Databases | * | `/api/v1/databases/*` | Out of scope |
| Services | * | `/api/v1/services/*` | Out of scope (v1 is application-only) |

## 5. HTTP Status Codes

| Code | Meaning | safe-ify handling |
|------|---------|-------------------|
| 200 | Success | Parse response, return data |
| 400 | Bad request | Show API error message |
| 401 | Auth failure | Suggest re-running `safe-ify auth add` |
| 403 | Insufficient token permissions | Suggest checking token abilities |
| 404 | Resource not found | Suggest checking app UUID |
| 409 | Conflict (e.g., domain) | Show API error message |
| 422 | Validation failure | Show API error details |
| 429 | Rate limit exceeded | Show retry-after value from header |

## 6. Application Object (relevant fields)

Based on the API response for `GET /api/v1/applications/{uuid}`:

```json
{
  "id": 1,
  "uuid": "hgkks00",
  "name": "my-application",
  "description": "...",
  "fqdn": "https://app.example.com",
  "status": "running",
  "build_pack": "nixpacks",
  "git_repository": "https://github.com/user/repo",
  "git_branch": "main",
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-03-10T00:00:00Z"
}
```

**Status values observed:** `running`, `stopped`, `restarting`, `exited`.

## 7. Deploy Response Object

```json
{
  "deployments": [
    {
      "message": "Deployment request queued.",
      "resource_uuid": "hgkks00",
      "deployment_uuid": "dl8k4s0"
    }
  ]
}
```

## 8. Rate Limiting

- Server-level concurrency limits on deployments.
- HTTP 429 response includes `Retry-After` header.
- safe-ify does not retry automatically; surfaces the error to the agent.
