# PLN-2026-002 Context

Status: Draft
Updated: 2026-03-11
Owner: Planner

## Key Decisions (compact -- full entries in 09-research-findings-decisions.md)

| ID | Decision |
|----|----------|
| D1 | Apps stored as YAML map keyed by name (not array) for uniqueness and direct lookup |
| D2 | Backward compat via auto-detection: `app_uuid` = legacy, `apps` = new, both = error |
| D3 | Three-layer deny merge: global + project + app (each layer can only restrict further) |
| D4 | Audit log gains AppName field between command and app_uuid |
| D5 | Config with both `app_uuid` and `apps` is rejected as invalid |
| D6 | `list` command bypasses `--app` requirement (does not target a specific app) |
| D7 | S2 consolidated into single implementer task for clean I->R->T chain |
| D8 | Legacy normalization uses empty app deny list (not project deny copy) |

## Deliverable -> Files Map

| Deliverable | Role | Model | Read these files |
|-------------|------|-------|-----------------|
| D1: Config types + loading + runtime | Implementer | Sonnet | 03-tech-spec-dev-architecture.md, 04-tech-spec-dev-config-migration.md |
| D2: Permissions enforcer update | Implementer | Sonnet | 03-tech-spec-dev-architecture.md |
| D3: CLI layer (--app flag, agent.go, init, doctor, audit) | Implementer | Sonnet | 04-tech-spec-dev-cli-commands.md, 03-tech-spec-dev-architecture.md |

## Slice gates (requires user approval -- each slice reviewed + merged independently)

| Slice | Gate task | Increment |
|-------|-----------|-----------|
| S1 | T5 | Config layer: multi-app types, loading, backward compat, runtime resolution, permission enforcement |
| S2 | T10 | CLI layer: --app flag, init add-app, doctor multi-app, audit app name |

## Out of scope

- Multi-instance per project
- Explicit migration CLI command
- Changes to global config format
- Changes to Coolify API client
