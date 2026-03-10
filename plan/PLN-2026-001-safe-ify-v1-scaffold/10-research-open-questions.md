# PLN-2026-001 Open Questions

## Active

### Q1: Coolify API version compatibility

**Question:** How stable is the Coolify v4 API across beta releases? Will endpoints change between beta.460 and GA?
**Impact:** API client may need updates if endpoints change.
**Mitigation:** Pin to beta.460 behavior. `doctor` command checks version. Document API version in user-facing output.
**Status:** Accepted risk. Monitor Coolify changelog.

### Q2: Token ability requirements

**Question:** Does the Coolify API enforce token abilities strictly? If a token has `read` + `deploy` but not `write`, can we guarantee safe-ify never accidentally needs `write`?
**Impact:** Token setup instructions for users.
**Mitigation:** All safe-ify operations only use `read` and `deploy` abilities. Document minimum required token abilities in `auth add` flow.
**Status:** Accepted. Verified against API matrix.

### Q3: Log streaming vs polling

**Question:** The Coolify logs endpoint returns historical logs, not a real-time stream. Should `safe-ify logs` support `--follow` mode via polling?
**Impact:** Agent experience when watching deployments.
**Mitigation:** v1 does not implement `--follow`. Historical logs with `--tail` are sufficient for agent use cases.
**Status:** Deferred to v2.

## Resolved

(None yet -- all questions documented at plan creation time.)

## Parking Lot (future versions)

- MCP server integration for Claude Code
- Environment variable management (read-only)
- Deployment status polling/waiting (`safe-ify deploy --wait`)
- Log rotation for audit log
- Multi-team support (multiple tokens per instance)
- Service resource support (not just applications)
