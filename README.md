# safe-ify

A CLI safety layer for coding agents to interact with [Coolify](https://coolify.io/) v4.

**safe-ify** exposes only vetted, non-destructive operations (deploy, redeploy, logs, status, list) and prevents agents from accessing raw credentials or destructive API endpoints. Designed for monorepos with multiple Coolify apps.

## Features

- **Allowlist-only** — only safe operations exposed, never delete/stop/config-change
- **Two-layer config** — global secrets (`~/.config/safe-ify/config.yaml`, `0600`) + committable project config (`.safe-ify.yaml`)
- **Multi-app support** — manage multiple Coolify apps per project with `--app` flag
- **Three-layer permissions** — global, project, and per-app deny lists (each layer can only restrict further)
- **Machine-readable output** — all agent commands support `--json` with `{ok, data, error}` envelope
- **Audit logging** — append-only log of all agent actions
- **Zero credential leakage** — tokens never printed, never in JSON output, never in audit logs
- **TTY-guarded config** — `init`, `auth add`, `auth remove`, and `update` require an interactive terminal, so agents cannot modify configuration or the binary
- **Passphrase-protected** — authorization commands (`auth add`, `auth remove`, `init`) require a human-set passphrase

## Quick start

```bash
# Install
go install github.com/RazBrry/safe-ify/cmd/safe-ify@latest

# Or build from source
make build  # → ./bin/safe-ify

# Add your Coolify instance
safe-ify auth add

# Link a project (run from your repo root — multi-select your apps)
safe-ify init

# Run again to add/remove apps (already configured apps are pre-selected)
safe-ify init
```

## Usage

### Human commands (interactive TUI)

| Command | Description |
|---------|-------------|
| `safe-ify auth add` | Add or update a Coolify instance (URL + token) |
| `safe-ify auth remove` | Remove a configured instance |
| `safe-ify auth list` | List configured instances (tokens masked) |
| `safe-ify init` | Multi-select Coolify apps for this project (re-run to add/remove) |
| `safe-ify update` | Update safe-ify to the latest version |
| `safe-ify doctor` | Validate setup, output CLAUDE.md snippet |

### Agent commands (non-interactive, `--json`)

| Command | Description |
|---------|-------------|
| `safe-ify deploy --app api --json` | Trigger deployment |
| `safe-ify deploy --app api --json --wait` | Deploy and wait for completion (polls status) |
| `safe-ify redeploy --app api --json` | Redeploy current version ¹ |
| `safe-ify logs --app api --json --tail 50` | Fetch recent logs (default: 100 lines) |
| `safe-ify status --app api --json` | Check deployment status |
| `safe-ify list --json` | List available applications (no `--app` needed) |
| `safe-ify env list --app api --json` | List env var keys (add `--show-values` for values) |
| `safe-ify env get --app api --key DB_HOST --json` | Get a specific env var value |
| `safe-ify env set --app api --key DB_HOST --value localhost --json` | Create or update an env var |
| `safe-ify env delete --app api --key OLD_VAR --json` | Delete an env var |
| `safe-ify deployments --app api --json` | List deployment history (default: last 10) |
| `safe-ify domains --app api --json` | Show configured domains/URLs |
| `safe-ify resources --app api --json` | Show resource limits (CPU, memory) |
| `safe-ify rollback --app api --to <sha> --json` | Rollback to a previous commit/tag |
| `safe-ify preview-deploy --app api --branch <ref> --json` | Deploy a specific branch or tag |

> ¹ `redeploy` uses the Coolify `/restart` endpoint, which may return 403 on some Coolify versions even with the correct token scopes. If you hit this, use `deploy --force` instead.

`deploy`, `redeploy`, `rollback`, and `preview-deploy` all support `--wait` to poll until completion (`--timeout`, `--poll-interval` configurable).

## API token requirements

Create a Coolify API token at **Settings → API Tokens** with these scopes:

| Scope | Required for |
|-------|-------------|
| `read` | status, logs, list, deployments, domains, resources, deployment polling |
| `deploy` | deploy, redeploy, rollback, preview-deploy |
| `read:sensitive` | reading environment variable values (`env list --show-values`, `env get`) |
| `write` | modifying environment variables (`env set`, `env delete`) |

Minimum for current features: **`read` + `deploy`**.

## Multi-app config

For monorepos with multiple Coolify apps, `.safe-ify.yaml` uses a named app map:

```yaml
instance: my-coolify
apps:
  api:
    app_uuid: "abc123-..."
  web:
    app_uuid: "def456-..."
    permissions:
      deny: [deploy, env-write, rollback, preview-deploy]  # web app: read-only
```

`rollback` and `preview-deploy` are denied by default when adding new apps via `safe-ify init`. The human can opt in by deselecting them in the deny list picker.

Legacy single-app configs (`app_uuid` at root level) are auto-detected and work without changes.

## Security model

safe-ify enforces a strict separation between human and agent capabilities:

- **Agents can only** run the allowlisted commands (`deploy`, `redeploy`, `logs`, `status`, `list`, `env list/get/set/delete`, `deployments`, `domains`, `resources`, `rollback`, `preview-deploy`) within the permissions granted by the config.
- **Agents cannot** modify configuration or the binary — `init`, `auth add`, `auth remove`, and `update` require an interactive terminal (TTY check) and will refuse to run when called from a non-interactive shell.
- **Passphrase-protected** — authorization commands (`auth add`, `auth remove`, `init`) require a passphrase set on first use. This protects credential management and project bindings.
- **Permissions are deny-only** — each layer (global, project, per-app) can only restrict further, never grant back a denied command.
- **Tokens are never exposed** — not in CLI output, not in JSON responses, not in audit logs.

### Passphrase protection

The first time you run `safe-ify auth add`, you'll be prompted to set a passphrase (minimum 8 characters). This passphrase is required for all subsequent authorization commands:

| Protected command | Why |
|---|---|
| `auth add` | Adds or rotates Coolify credentials |
| `auth remove` | Removes Coolify credentials |
| `init` | Changes which instance/apps a project is bound to, modifies permission policy |

The passphrase hash (bcrypt) is stored in `~/.config/safe-ify/config.yaml`. Agent commands (`deploy`, `status`, etc.) are **not** affected — they never require the passphrase.

### Project config integrity (Ed25519 signing)

The project config (`.safe-ify.yaml`) lives in the repo and is readable by agents — but agents cannot silently edit it and have safe-ify trust the change.

**How it works:**

1. On first setup, safe-ify generates an Ed25519 keypair. The private key is encrypted with your passphrase (Argon2id + AES-GCM) and stored in global config. The public key is stored in plaintext.
2. When `init` saves `.safe-ify.yaml`, it also writes `.safe-ify.sig` — an Ed25519 signature of the exact file bytes.
3. Every agent command verifies the signature against the public key **before parsing the YAML**. If the signature is missing or invalid, the command is rejected.

This gives you:
- **Agents can read** the config (to know which app to target)
- **Agents cannot forge** a valid signature (they don't have the passphrase to unlock the private key)
- **Tampering is detected** — any edit to `.safe-ify.yaml` without re-signing via `init` breaks the signature
- **No password prompts** on agent commands — verification uses the public key only

Both `.safe-ify.yaml` and `.safe-ify.sig` should be committed to version control.

### Permission notes

- **`env-read` means secret access.** If your Coolify token has the `read:sensitive` scope, an agent with `env-read` permission can read environment variable values (including secrets) via `env get` and `env list --show-values`. If your agent should not have access to secrets, deny `env-read`.
- **`rollback` and `preview-deploy` are privileged.** Unlike `deploy` (which deploys the current branch), these commands let the agent choose *which* code to deploy. They are denied by default in new project configs.
- **`init` is an authorization change**, not a normal agent command. It controls which instance and apps a project talks to and sets permission policy — that's why it requires the passphrase.

### Threat model and security boundary

safe-ify's security controls are designed for one specific scenario: **an agent that interacts with Coolify exclusively through the safe-ify CLI**. In that model, the protections are:

- **TTY check** — agents cannot run authorization commands (`auth add`, `auth remove`, `init`, `update`)
- **Passphrase** — even at an interactive terminal, authorization commands require a human-set passphrase
- **Ed25519 signatures** — agents cannot silently edit `.safe-ify.yaml` (changing deny lists, swapping app UUIDs, or pointing to a different instance) and have safe-ify trust the change
- **Deny-only permissions** — each config layer can only restrict further, never grant back a denied command

**What safe-ify does NOT protect against:**

A fully malicious process running as the **same OS user** can bypass all of these controls. It can:

- Read `~/.config/safe-ify/config.yaml` directly, extracting API tokens, the passphrase hash, and the encrypted private signing key
- Replace the public key in global config to forge signatures on a tampered `.safe-ify.yaml`
- Call the Coolify API directly, bypassing safe-ify entirely
- Modify the safe-ify binary itself

This is an inherent limitation of same-user process isolation — file permissions (`0600`) protect against *other* users, not against processes running as the same user. safe-ify does not attempt to solve this problem.

**If you need a hard secret boundary**, run the agent as a **different OS user** and ensure:
- `~/.config/safe-ify/config.yaml` is owned by the human user with `0600` permissions
- The `safe-ify` binary is not writable by the agent user
- The agent user can only invoke `safe-ify` as a command, not read its config files

In summary: safe-ify is a **policy enforcement layer**, not a sandbox. It prevents well-behaved agents from accidentally performing dangerous operations. It does not defend against a process that is actively trying to subvert it from the same OS account.

## Tech stack

Go, [Cobra](https://github.com/spf13/cobra), [Charm](https://charm.sh/) (huh + lipgloss), YAML config.

## License

MIT
