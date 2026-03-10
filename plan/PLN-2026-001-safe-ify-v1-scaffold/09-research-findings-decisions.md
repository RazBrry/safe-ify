# PLN-2026-001 Research Findings & Decisions

> **This is the authoritative record of all decisions made during planning and execution.**

---

### D1: Language and CLI Framework

**Date:** 2026-03-10
**Context:** Need a CLI tool that starts fast, distributes as a single binary, and is easy for agents to invoke.
**Options considered:**
1. Go + Cobra -- single binary, fast startup, mature CLI ecosystem, Charm libraries for TUI. Pros: no runtime deps, sub-100ms startup. Cons: more verbose than scripting languages.
2. Node.js + Commander -- faster to prototype. Cons: requires Node.js runtime, slower startup, larger distribution.
3. Rust + Clap -- fastest execution. Cons: longer development time, smaller TUI ecosystem.

**Decision:** Go + Cobra. Best balance of development speed, startup performance, and distribution simplicity. Charm ecosystem provides excellent TUI support.
**Consequences:** All source code is Go. Build via `go build`. Distribution via `go install`.

---

### D2: Two-Layer Config Architecture

**Date:** 2026-03-10
**Context:** Need to store Coolify credentials securely while allowing per-project settings to be committed to version control.
**Options considered:**
1. Single config file in home directory -- simple but no project-level customization.
2. Environment variables only -- no per-project settings, tokens visible in shell history.
3. Two files: global (secrets) + project (no secrets) -- separation of concerns.

**Decision:** Two-layer config. Global at `~/.config/safe-ify/config.yaml` (0600), project at `.safe-ify.yaml` (committable).
**Consequences:** Config loading must merge two layers. Project file must never contain secrets. Global file must enforce strict file permissions.

---

### D3: Deny-Only Permission Model

**Date:** 2026-03-10
**Context:** Need to allow per-project permission restrictions without risk of privilege escalation.
**Options considered:**
1. Allow + deny lists at project level -- flexible but introduces escalation risk if allow list is misused.
2. Deny-only at project level -- simpler, provably cannot escalate.
3. Explicit allow-list at project level -- requires listing all allowed commands, error-prone.

**Decision:** Deny-only at project level. All agent commands are allowed by default; project config can only deny, never allow.
**Consequences:** Permission resolution algorithm: start with all allowed, subtract global denials, subtract project denials. No mechanism to re-enable a globally denied command.

---

### D4: Coolify API Endpoint Selection

**Date:** 2026-03-10
**Context:** Coolify v4 API exposes many endpoints including destructive ones (delete, stop, modify). Need to select only safe endpoints.
**Options considered:**
1. Expose all read + deploy endpoints -- includes start/stop which could be misused.
2. Expose minimal set: deploy, restart, logs, list, status -- covers agent needs without destructive access.

**Decision:** Minimal safe set: deploy (via `/api/v1/deploy`), restart (as redeploy, via `/applications/{uuid}/restart`), logs, list applications, get application (status). Explicitly exclude: stop, start, delete, update, env vars, servers, databases, services.
**Consequences:** Agent cannot stop, delete, or modify applications. See `08-research-api-matrix.md` section 4 for full exclusion list.

---

### D5: Deploy vs Redeploy Mapping

**Date:** 2026-03-10
**Context:** Coolify API has `/deploy` (trigger new deployment) and `/applications/{uuid}/restart` (restart containers). Need clear CLI mapping.
**Options considered:**
1. Map `deploy` to `/deploy` and `redeploy` to `/restart` -- clear distinction.
2. Use only `/deploy` for both -- loses the "restart without rebuild" capability.

**Decision:** `safe-ify deploy` maps to `POST /api/v1/deploy?uuid={uuid}` (new deployment with build). `safe-ify redeploy` maps to `POST /api/v1/applications/{uuid}/restart` (restart/recreate containers without rebuild). POST used for both per D11.
**Consequences:** Deploy triggers a full build; redeploy only restarts. Force flag available on deploy for cache bypass.

---

### D6: JSON Output Envelope

**Date:** 2026-03-10
**Context:** Agent-facing commands need structured, parseable output. Need consistent format.
**Options considered:**
1. Raw Coolify API response passthrough -- inconsistent formats between endpoints.
2. Standardized envelope with `ok`, `data`, `error` -- consistent, easy for agents to parse.

**Decision:** Standard envelope: `{"ok": true/false, "data": {...}, "error": {...}}`. Error object includes `code` and `message` fields.
**Consequences:** All agent commands must wrap their output in this envelope. Error codes are enumerated constants.

---

### D7: Audit Log Format

**Date:** 2026-03-10
**Context:** Need to record agent actions for accountability.
**Options considered:**
1. JSON log file -- structured, queryable. Cons: harder to read with tail.
2. Pipe-delimited text -- simple, readable with standard tools. Cons: less structured.
3. SQLite database -- queryable. Cons: adds dependency.

**Decision:** Pipe-delimited text file at `~/.config/safe-ify/audit.log`. Format: `timestamp | command | app_uuid | instance | result | duration_ms`.
**Consequences:** Simple append-only file. No rotation in v1. Parseable with `awk` or `cut`.

---

### D8: TUI Library Choice

**Date:** 2026-03-10
**Context:** Human-facing commands need interactive prompts. Need lightweight TUI.
**Options considered:**
1. Charm `huh` + `lipgloss` -- purpose-built for Go CLI forms, polished UI.
2. `survey` (AlecAivazis) -- popular but less maintained.
3. `promptui` -- simple but limited form support.

**Decision:** Charm ecosystem: `huh` for forms and multi-select, `lipgloss` for styled output.
**Consequences:** Human-facing commands use `huh.NewForm()`, `huh.NewSelect()`, `huh.NewConfirm()`. Output styling uses `lipgloss`.

---

### D9: Config File Format

**Date:** 2026-03-10
**Context:** Need human-readable config format with good Go library support.
**Options considered:**
1. YAML -- human-readable, widely known, good Go support (`gopkg.in/yaml.v3`).
2. TOML -- more precise, less ambiguous. Cons: less familiar to most developers.
3. JSON -- universal support. Cons: no comments, verbose.

**Decision:** YAML for both config files. Using `gopkg.in/yaml.v3`.
**Consequences:** Config structs use `yaml:` struct tags.

---

### D10: Project Config Lookup Strategy

**Date:** 2026-03-10
**Context:** Agent may invoke safe-ify from a subdirectory of the project. Need to find `.safe-ify.yaml`.
**Options considered:**
1. Current directory only -- simple but fragile.
2. Parent directory traversal (like `.git` discovery) -- robust.
3. Explicit `--project` flag only -- requires agent to always pass flag.

**Decision:** Parent traversal by default (search current dir, then parent, up to filesystem root). Override with `--project` flag.
**Consequences:** `LoadProject()` implements upward directory walk.

---

### D11: POST for Side-Effecting API Operations

**Date:** 2026-03-10
**Context:** Coolify API documentation lists both GET and POST as accepted methods for deploy and restart endpoints. Need to choose the correct HTTP method for safe-ify.
**Options considered:**
1. GET for deploy/restart -- matches some Coolify API documentation examples. Cons: violates RESTful convention; GET must be safe and idempotent per RFC 7231.
2. POST for deploy/restart -- correct RESTful semantics for side-effecting operations.

**Decision:** Use POST for deploy (`/api/v1/deploy`) and restart (`/api/v1/applications/{uuid}/restart`). GET is used only for read operations (healthcheck, version, list, get, logs). The Coolify API accepts POST for these endpoints.
**Consequences:** All API call specs and the API matrix updated to specify POST for deploy and restart. The Coolify client `doRequest` helper must pass the correct method per operation.

---

### D12: `list` Command Requires Project Config

**Date:** 2026-03-10
**Context:** The `list` command was initially specified to work without a project config, using the global config directly. However, this bypasses project-level denial of the `list` command, breaking the deny-only permission model (D3).
**Options considered:**
1. Allow `list` without project config -- convenient but creates a permission bypass.
2. Require project config for `list` -- consistent with all other agent commands, respects project-level deny lists.

**Decision:** `list` follows the same startup sequence as all other agent commands: load project config, load global config, resolve permissions, check `list` permission. If no project config is found, error with "No project config found. Run `safe-ify init` first."
**Consequences:** All agent commands have a uniform startup sequence. Project-level denial of `list` is always enforced. The `--instance` flag on `list` is removed; the instance comes from project config like all other commands.

---

### D13: Token Output Contract for Secret Isolation

**Date:** 2026-03-10
**Context:** Security of token handling was described in terms of filesystem access restrictions, which is insufficient -- the tool itself must guarantee tokens never appear in any output channel.
**Options considered:**
1. Rely on filesystem access controls -- insufficient; does not prevent the tool from leaking tokens in its own output.
2. Enforce at the tool's output boundary -- tokens are structurally never included in stdout, stderr, JSON output, or audit log.

**Decision:** Secret isolation is enforced by the tool's output contract: tokens are never printed, never included in JSON output, never logged in audit entries. The tool itself never exposes tokens in any output channel. This is enforced structurally in the code (token field is never passed to any output/logging function).
**Consequences:** Security does not depend on filesystem access restrictions alone. The tool's output guarantee is testable and verifiable.
