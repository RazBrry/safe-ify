# PLN-2026-002 — Tech Spec Developer: Config Format & Backward Compatibility

## Current Format (Legacy)

```yaml
instance: my-coolify
app_uuid: abc-123-def
permissions:
  deny: []
```

Fields: `instance` (required), `app_uuid` (required), `permissions.deny` (optional).

## New Format (Multi-App)

```yaml
instance: my-coolify
permissions:
  deny: []          # project-level deny (applies to ALL apps)
apps:
  frontend:
    uuid: abc-123
    permissions:
      deny: []      # app-level deny (additional restrictions for this app)
  api:
    uuid: def-456
    permissions:
      deny: [deploy]
```

Fields: `instance` (required), `apps` (required, at least one entry), `permissions.deny` (optional, project-level).

Each app entry: `uuid` (required), `permissions.deny` (optional).

## Backward Compatibility Strategy (D2)

Detection logic in `LoadProject`:

1. Unmarshal YAML into `ProjectConfig` (which now has both `AppUUID` and `Apps` fields)
2. If both `AppUUID != ""` and `len(Apps) > 0` --> return error (D5: ambiguous format)
3. If `AppUUID != ""` and `Apps` is empty --> legacy format detected:
   - Normalize: create `Apps` map with key `"default"`, UUID = `AppUUID`, empty deny list
   - Clear `AppUUID` field (internal representation is always multi-app)
4. If `Apps` is not empty --> new format, use directly
5. If both are empty --> return error (no apps configured)

This means the internal representation is **always** the multi-app format after loading. All downstream code works with `Apps` map only.

## Validation Rules

- `instance` must be non-empty
- `apps` must have at least one entry (after normalization)
- Each app key must match `^[a-zA-Z0-9][a-zA-Z0-9-]*$`
- Each app must have a non-empty `uuid`
- App deny list entries must be valid agent command names
- Project-level deny list entries must be valid agent command names

## SaveProject Changes

`SaveProject` must write the new multi-app format. It never writes the legacy format. This means:
- If a legacy config is loaded and saved back (e.g., by `init --add-app`), it is written in the new format
- The `app_uuid` field is never written by the new code

## ResolveRuntime Changes

Current signature: `ResolveRuntime(global, project) -> RuntimeConfig`

New signature: `ResolveRuntime(global, project, appName string) -> RuntimeConfig`

- `appName` comes from the `--app` CLI flag
- If `appName == ""` and exactly one app exists, use it (the app name is set in RuntimeConfig)
- If `appName == ""` and multiple apps exist, return `AppAmbiguousError`
- If `appName` is provided but not found in `Apps`, return `AppNotFoundError`
- Deny list merge: `global.Defaults.Permissions.Deny` + `project.Permissions.Deny` + `app.Permissions.Deny`

## File Changes Summary

| File | Change |
|------|--------|
| `internal/config/types.go` | Add `AppConfig` struct, add `Apps` field to `ProjectConfig`, add `AppName` to `RuntimeConfig` |
| `internal/config/project.go` | Update `LoadProject` with format detection/normalization, update `SaveProject` for new format, add app name validation |
| `internal/config/runtime.go` | Update `ResolveRuntime` signature, add app lookup + three-layer deny merge |
| `internal/config/errors.go` | Add `AppNotFoundError`, `AppAmbiguousError` |
