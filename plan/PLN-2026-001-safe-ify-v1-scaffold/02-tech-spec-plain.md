# PLN-2026-001 Tech Spec Functioneel Uitgelegd

## 1. What is the problem?

AI coding agents like Claude Code are powerful tools, but giving them direct access to infrastructure APIs is dangerous. A coding agent with a Coolify API token could accidentally delete applications, stop production services, or leak credentials. There is no built-in way to give an agent "deploy and read logs" access without also giving it "delete everything" access.

safe-ify solves this by sitting between the agent and the Coolify API. It only exposes a curated set of safe operations and enforces per-project permission restrictions.

---

## 2. How does it work (conceptually)?

Think of safe-ify as a locked toolbox. The human sets up which tools are in the box (global config) and can further lock specific tools for specific projects (project config). The agent can only use unlocked tools.

There are three layers:

1. **Setup layer** (human only): Interactive terminal forms let the human register Coolify instances and link projects. This is where credentials live, locked away in a protected config file.

2. **Permission layer** (automatic): When the agent runs a command, safe-ify checks both the global and project configs. If the command is on any deny list, it refuses. The project can only make things MORE restrictive, never less.

3. **Execution layer** (agent-triggered): If the permission check passes, safe-ify calls the Coolify REST API on the agent's behalf, formats the result as JSON, and returns it. The agent never sees the API token.

---

## 3. What choices were made and why?

| Choice | Rationale |
|--------|-----------|
| **Go language** | Single binary, no runtime dependencies, fast startup (<100ms), cross-platform. Ideal for a CLI tool that agents invoke frequently. |
| **Cobra CLI framework** | Industry standard for Go CLIs. Provides subcommand routing, flag parsing, help generation. |
| **Charm ecosystem for TUI** | `huh` for forms and `lipgloss` for styling give polished interactive prompts for human-facing commands, without heavy dependencies. |
| **Two config files** | Separation of concerns: secrets stay in the global config (never in the repo), while project-level settings are committable and shareable. |
| **Deny-only project permissions** | Simpler mental model and prevents escalation bugs. You can only take away permissions at the project level, never add them. |
| **YAML for config** | Human-readable, well-supported in Go, familiar to developers. |
| **Audit log as append-only file** | Simple, no external dependencies, sufficient for v1. Agents should be accountable for their actions. |
| **`--json` flag on agent commands** | Agents need structured output to parse reliably. Humans get readable text by default. |

---

## 4. How does it interact with other parts of the system?

```
+----------------+     +------------+     +-------------------+
|  Claude Code   | --> |  safe-ify  | --> | Coolify v4 API    |
|  (agent)       |     |  (CLI)     |     | (self-hosted PaaS)|
+----------------+     +------+-----+     +-------------------+
                              |
                    +---------+---------+
                    |                   |
            ~/.config/safe-ify/    .safe-ify.yaml
            config.yaml            (in project repo)
            (secrets, 0600)        (no secrets, committed)
```

- **Claude Code** invokes safe-ify as a subprocess. It reads JSON output from stdout.
- **safe-ify** reads its config files, checks permissions, calls the Coolify API, and returns results.
- **Coolify API** is a REST API with Bearer token auth. safe-ify talks to it over HTTPS.
- **CLAUDE.md** is updated by `safe-ify doctor` to tell agents which commands are available.

---

## 5. What are the risks and limitations?

| Risk | Mitigation |
|------|------------|
| **Token stored on disk** | File permissions enforced at 0600; safe-ify refuses to run if permissions are too open. |
| **Agent could edit config files directly** | Config files are not in the project directory. The global config requires filesystem-level access the agent should not have. |
| **Coolify API changes** | Pinned to v4.0.0-beta.460 API. Version check in `doctor` command. |
| **No rate limiting** | Relies on Coolify's built-in rate limiting (HTTP 429). safe-ify surfaces the error. |
| **Audit log grows unbounded** | Acceptable for v1. Log rotation is out of scope. |
| **No MCP integration** | v1 is CLI-only. MCP integration is a planned future enhancement. |

---

## 6. Reference to full technical spec

For implementation details, see the developer technical specification shards:

- **Architecture**: `03-tech-spec-dev-architecture.md` -- project structure, data model, API client design
- **CLI Commands**: `04-tech-spec-dev-cli-commands.md` -- command implementations, flag handling, TUI forms
- **Config & Permissions**: `04-tech-spec-dev-config-permissions.md` -- config loading, permission enforcement, file security
- **Operations**: `05-tech-spec-dev-ops.md` -- build system, test strategy, CI, DoR/DoD
