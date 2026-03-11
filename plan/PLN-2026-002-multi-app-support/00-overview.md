# PLN-2026-002 — Multi-App Project Config Support

**Status:** Draft
**Owner:** Planner
**Created:** 2026-03-11
**Updated:** 2026-03-11

## Summary

Support multiple Coolify applications per project. Currently `.safe-ify.yaml` supports a single `app_uuid`. Monorepos and projects with separate frontend/backend apps need to target multiple apps from the same project config.

## Scope

1. New project config format with named apps, each having its own UUID and optional deny list
2. `--app` flag on all agent commands to select which app to target
3. Updated `init` command with an `add-app` sub-flow for adding apps to existing projects
4. Updated `doctor` command to validate and display all configured apps
5. Per-app permission enforcement (per-app deny lists merged with project-level and global deny lists)
6. Backward compatibility: existing single-app `.safe-ify.yaml` files continue to work without migration
7. Updated audit logging to include the app name (not just UUID)

## Out of Scope

- Multi-instance support (each project still targets one Coolify instance)
- Automated migration CLI command (old format is auto-detected and works as-is)
- Changes to the global config format
- Changes to the Coolify API client

## Slice Map

| Slice | Increment | Gate |
|-------|-----------|------|
| S1 | Config format: new multi-app types, backward-compat loading, runtime resolution with `--app` | T5 |
| S2 | CLI integration: `--app` flag on all agent commands, updated `init` (add-app flow), updated `doctor`, updated audit entry | T12 |

## Acceptance Criteria

- [ ] AC1: A `.safe-ify.yaml` with multiple named apps under an `apps:` key is loaded correctly
- [ ] AC2: Existing single-app `.safe-ify.yaml` files (with `app_uuid:`) load without error or migration
- [ ] AC3: `safe-ify deploy --app frontend --json` targets the correct app UUID
- [ ] AC4: `--app` flag is required when multiple apps are configured; omitting it produces a clear error
- [ ] AC5: When only one app is configured (old or new format), `--app` is optional and defaults to that app
- [ ] AC6: Per-app deny lists are merged with project-level and global deny lists
- [ ] AC7: `safe-ify init` can add an app to an existing multi-app config
- [ ] AC8: `safe-ify doctor` validates and displays all configured apps
- [ ] AC9: Audit log entries include the app name alongside the UUID
- [ ] AC10: All existing tests continue to pass
