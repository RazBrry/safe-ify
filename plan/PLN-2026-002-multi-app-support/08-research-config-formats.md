# PLN-2026-002 — Research: Config Format Options

## Question

How should multiple apps be represented in `.safe-ify.yaml`, and how do we maintain backward compatibility with the existing single-app format?

## Options Explored

### Option A: Array of apps

```yaml
instance: my-coolify
apps:
  - name: frontend
    uuid: abc-123
  - name: api
    uuid: def-456
```

**Pros:** Preserves insertion order. Explicit name field.
**Cons:** Allows duplicate names (must validate). Lookups require iteration. More verbose.

### Option B: Map of apps (chosen — D1)

```yaml
instance: my-coolify
apps:
  frontend:
    uuid: abc-123
  api:
    uuid: def-456
```

**Pros:** Duplicate names impossible (YAML map keys). Direct lookup by name. Concise. Consistent with `instances` map in global config.
**Cons:** YAML map order is not guaranteed (not a problem — order is irrelevant for this use case).

### Option C: Flat multi-uuid field

```yaml
instance: my-coolify
app_uuids:
  frontend: abc-123
  api: def-456
```

**Pros:** Very compact.
**Cons:** No place for per-app settings (deny lists). Would need a separate `app_permissions` section, making the config harder to read and maintain.

## Backward Compatibility Strategies

### Strategy 1: Auto-detection (chosen — D2)

Detect format by checking which fields are present:
- `app_uuid` present --> legacy format, normalize internally
- `apps` present --> new format
- Both present --> error

**Pros:** Zero user action needed. Old configs work immediately. Simple implementation.
**Cons:** Must handle both formats in the loader.

### Strategy 2: Explicit version field

Add `version: 2` to the config. Loader checks version to decide parsing.

**Pros:** Explicit and clear.
**Cons:** Adds friction. Existing configs need editing to add `version: 1`.

### Strategy 3: Migration command

`safe-ify migrate-config` converts old format to new.

**Pros:** Clean separation.
**Cons:** Extra command. Users must remember to run it. Old format eventually unsupported.

## Per-App Permissions

### Three-layer merge (chosen — D3)

```
Effective deny = global.deny ∪ project.deny ∪ app.deny
```

Each layer can only add restrictions. This is consistent with the existing two-layer model (global + project) and extends it naturally.

### Alternative: App deny overrides project deny

Rejected. Violates the "can only restrict, never escalate" principle. An app-level config could remove a project-level restriction.
