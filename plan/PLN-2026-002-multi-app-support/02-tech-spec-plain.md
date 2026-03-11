# PLN-2026-002 — Tech Spec Functioneel Uitgelegd

## 1. What is the problem?

safe-ify currently supports only one Coolify application per project. The `.safe-ify.yaml` file has a single `app_uuid` field. Monorepos and projects with multiple services (frontend, API, workers) need to manage deployments for several apps from the same directory. Today, this requires separate config files or manual UUID switching, which is error-prone.

## 2. How does it work (conceptually)?

The project config file (`.safe-ify.yaml`) gets a new structure. Instead of one `app_uuid`, it can contain a named map of apps:

```yaml
instance: my-coolify
apps:
  frontend:
    uuid: abc-123
  api:
    uuid: def-456
    deny: [deploy]
```

Each app has a name (the map key), a UUID, and an optional deny list. Agent commands accept an `--app` flag to select which app to target. When only one app exists, the flag is optional.

The system reads the config, finds the app by name, merges its deny list with the project-level and global deny lists, and proceeds as before.

## 3. What choices were made and why?

- **Map-based naming** (D1): Apps are keyed by name in a YAML map rather than using an array. This prevents duplicate names, makes lookups simple, and produces readable configs.
- **Backward compatibility via detection** (D2): The loader checks for the presence of `app_uuid` (old format) vs `apps` (new format). Old configs work without any migration step.
- **Three-layer deny merge** (D3): Permissions are resolved as global deny + project deny + app deny. Each layer can only add restrictions, never remove them. This maintains the "can only restrict, never escalate" principle.
- **App name in audit log** (D4): The audit log entry gains an `AppName` field so operators can quickly identify which service was acted on.

## 4. How does it interact with other parts of the system?

- **Config layer** (`internal/config/`): New types, updated loading/saving, updated runtime resolution.
- **Permissions layer** (`internal/permissions/`): Updated `NewEnforcer` to accept a per-app deny list.
- **CLI layer** (`internal/cli/`): New `--app` persistent flag, updated `init`, updated `doctor`.
- **Audit layer** (`internal/audit/`): New `AppName` field in `Entry`.
- **Global config**: No changes. Instances are unaffected.
- **Coolify client**: No changes. The client already works with UUIDs passed to it.

## 5. What are the risks and limitations?

- **Config file growth**: Large monorepos with many apps could produce verbose configs, but this is unlikely to be a practical issue.
- **Old-format detection ambiguity**: If a config file has both `app_uuid` and `apps`, the loader must pick one. Decision: reject as invalid (D5).
- **No interactive app selection for agents**: Agents must always use `--app` (or rely on single-app default). No TUI fallback for agent commands.

## 6. Reference to the full technical spec

- Architecture: `03-tech-spec-dev-architecture.md`
- Config migration: `04-tech-spec-dev-config-migration.md`
- CLI commands: `04-tech-spec-dev-cli-commands.md`
- Test strategy & ops: `05-tech-spec-dev-ops.md`
