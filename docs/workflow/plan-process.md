# Planning & Execution Workflow (SOP)

> **Scope:** This document is the single, complete Standard Operating Procedure for
> creating, executing, and closing plans in this project. It is loaded on-demand only
> (not auto-loaded into every session).

---

## 1. Overview

This project uses a **repo-first** planning model:

- **Active plans live in the repository** under `plan/PLN-YYYY-NNN/`. Every plan
  file is version-controlled, reviewed via PR, and readable without external tools.
- **Execution follows a sub-agent model.** A single orchestrator session (the main
  Claude conversation) reads the plan, spawns scoped sub-agents (Planner,
  Implementer, Reviewer, Tester), and drives a task board to completion. The human
  intervenes only at explicit phase gates.

The workflow has two hard phases:

1. **Phase 1 -- PLAN:** Create all plan files, assign a plan code, register in the
   Plan Index, stop, and ask for user confirmation.
2. **Phase 2 -- IMPLEMENT:** Only after explicit user confirmation (`GO IMPLEMENT`
   or `/implement PLN-YYYY-NNN`).

---

## 2. Plan Structure (Canonical)

Every plan lives in its own directory under `plan/`:

```
plan/PLN-YYYY-NNN/
  CONTEXT.md                         # Orchestrator entry point (~80 lines max)
  TASKS.md                           # Atomic task board
  00-overview.md                     # Plan Overzicht (mandatory doc 1)
  01-functional-spec.md              # Functionele Uitwerking (mandatory doc 2)
  02-tech-spec-plain.md              # Tech Spec Functioneel Uitgelegd (mandatory doc 3)
  03-tech-spec-dev-architecture.md   # Tech Spec Developer, shard: architecture
  04-tech-spec-dev-[component].md    # Tech Spec Developer, shard: per component
  05-tech-spec-dev-ops.md            # Tech Spec Developer, shard: migrations/tests/ops
  07-research-index.md               # Research overview & links
  08-research-api-matrix.md          # Or equivalent domain-specific research
  09-research-findings-decisions.md  # AUTHORITATIVE: all decisions
  10-research-open-questions.md      # Open questions & unknowns
```

### Mandatory documents (minimum 4)

| # | File | Purpose |
|---|------|---------|
| 1 | `00-overview.md` | Plan Overzicht -- scope, status, owner, timeline, acceptance criteria |
| 2 | `01-functional-spec.md` | Functionele Uitwerking -- user-facing behaviour, flows, edge cases |
| 3 | `02-tech-spec-plain.md` | Tech Spec Functioneel Uitgelegd -- conceptual technical explanation for non-developers |
| 4 | `03-tech-spec-dev-*.md` | Tech Spec Developer -- full technical specification, always sharded |

### Sharding rule

Document 4 (Tech Spec Developer) is **always split by component or concern**. Each
shard must be independently readable. Typical shards:

- `03-tech-spec-dev-architecture.md` -- system-level design, data model, API surface
- `04-tech-spec-dev-[component].md` -- one file per major component
- `05-tech-spec-dev-ops.md` -- migrations, test strategy, deployment, monitoring

### Required sections in `02-tech-spec-plain.md`

1. What is the problem?
2. How does it work (conceptually)?
3. What choices were made and why?
4. How does it interact with other parts of the system?
5. What are the risks and limitations?
6. Reference to the full technical spec (doc 4 shards)

### Research documents

Every plan includes at minimum:

| File | Purpose |
|------|---------|
| `07-research-index.md` | Lists all research documents with links |
| `08-research-api-matrix.md` | Domain-specific research (API capabilities, external system constraints) |
| `09-research-findings-decisions.md` | **Authoritative record** of all decisions made during planning and execution |
| `10-research-open-questions.md` | Unresolved questions, parking lot items |

---

## 2b. Slice Structure

A **slice** is a group of tasks that form an independently deployable increment.
Each slice:

- Delivers a working capability that can be tested and reviewed in isolation
- Gets its own git branch (worktree) off the current `main`
- Is independently reviewable — the diff is scoped to one increment
- Can be merged to `main` before subsequent slices begin

### Slice definition rules

1. Every plan must define at least one slice.
2. Each task in TASKS.md belongs to exactly one slice (via the `Slice` column).
3. A slice must include the full Implementer → Reviewer → Tester sequence for
   its deliverables. No slice leaves untested code.
4. A slice ends with a `[GATE]` row — this is the merge-readiness checkpoint.
5. Slices are ordered — S2 may depend on S1 being merged to `main`, but S1
   must be self-contained.

### Slice lifecycle

```
/plan-spec → plan with slices defined
/implement-spec PLN-YYYY-NNN S1 → implements S1 in a dedicated worktree
/review-worktree <branch-S1> → review with merge readiness analysis
  User merges S1 to main (manual)
/implement-spec PLN-YYYY-NNN S2 → new worktree from updated main
```

---

## 3. PLN Convention

### Format

```
PLN-YYYY-NNN
```

- `PLN` -- literal prefix (always uppercase).
- `YYYY` -- year of plan creation.
- `NNN` -- ascending, zero-padded sequence number within the year.
- Example: `PLN-2026-001`.

### Rules

1. Assign the code **immediately** at plan creation -- never defer.
2. Codes are **never reused**, even if a plan is abandoned.
3. The code must appear in:
   - The plan directory name (`plan/PLN-2026-001/`)
   - The title of `00-overview.md`
   - The `CONTEXT.md` header
   - The Plan Index (`plan/index.md`)
4. To determine the next available code, scan `plan/index.md` or list existing
   `plan/PLN-*` directories and increment.

---

## 4. Pre-Implementation Gate (Mandatory)

```
PLAN-FIRST GATE: No code changes before:
  1. Plan exists (all required files created)
  2. Plan code is registered in Plan Index
  3. Plan status is Approved
  4. User explicitly says "GO IMPLEMENT" or runs /implement PLN-YYYY-NNN
```

### Phase 1 -- PLAN

1. Create all plan files per section 2.
2. Assign plan code per section 3.
3. Register in Plan Index.
4. **Stop and ask for user confirmation.** Do not proceed to implementation.

### Phase 2 -- IMPLEMENT

1. Only begins after the user provides explicit confirmation.
2. Acceptable triggers: `GO IMPLEMENT`, `/implement PLN-YYYY-NNN`, or equivalent
   unambiguous approval.
3. If the user asks for code changes without an approved plan, respond with the
   PLAN-FIRST GATE message and begin Phase 1 instead.

---

## 5. Sub-Agent Roles

| Role | Model | Responsibility | Writes to |
|------|-------|---------------|-----------|
| **Planner** | Opus | All plan files, research, decisions. Never writes production code. | `plan/` files only |
| **Implementer** | Sonnet | Production code for one deliverable. Never plans or reviews. | Source files only |
| **Code Quality Reviewer** | Opus | Reviews Implementer output against spec and coding standards. Never modifies code. | Verdict only (pass/fail + notes) |
| **Tester** | Sonnet | Writes and runs tests for one deliverable. | `tests/` files only |

### Separation of duties (mandatory)

- The **Implementer** for a deliverable can never be the **Reviewer** or **Tester** for that same deliverable.
- The **Orchestrator** (main Claude session) is the only entity that updates `TASKS.md`.
- The **Planner** is the only role that writes to research files (`07-` through `10-`).
- If an Implementer discovers something that requires a new decision during
  implementation: **stop**, flag to Orchestrator, and wait for the Planner to
  document the decision in `09-research-findings-decisions.md` before continuing.

---

## 6. Orchestrator Loop

The main Claude session acts as the **Orchestrator**. The user intervenes only at
phase gates.

```
Session start:
  Read CONTEXT.md + TASKS.md (~160 lines combined)

Loop:
  1. Find next task with status "Pending" in TASKS.md
  2. Determine role + model from the TASKS.md row
  3. Spawn sub-agent via Task tool with a scoped prompt
     (include only the input files listed in the task row)
  4. Validate sub-agent output against acceptance criteria in TASKS.md
  5. Pass -> mark task "Done" in TASKS.md, continue loop
  6. Fail -> retry same role with failure notes (max 2 retries),
     then surface to user if still failing
  7. Phase gate -> surface to user, wait for approval before continuing

End when all tasks are Done.
```

---

## 7. CONTEXT.md Structure

CONTEXT.md is the orchestrator's entry point. It must stay under ~80 lines.

```markdown
# PLN-YYYY-NNN Context

Status: [Draft | In Review | Approved | In Progress | Done]
Updated: YYYY-MM-DD
Owner: [role or person]

## Key Decisions (compact -- full entries in 09-research-findings-decisions.md)

| ID | Decision |
|----|----------|
| D1 | Brief one-liner summarizing the decision |

## Deliverable -> Files Map

| Deliverable | Role | Model | Read these files |
|-------------|------|-------|-----------------|
| D1: Component name | Implementer | Sonnet | 04-tech-spec-dev-component.md |

## Slice gates (requires user approval — each slice reviewed + merged independently)

| Slice | Gate task | Increment |
|-------|-----------|-----------|
| S1 | T# | What works after S1 merges |
| S2 | T# | What works after S2 merges |

## Out of scope

- Item 1
```

---

## 8. TASKS.md Structure

```markdown
# PLN-YYYY-NNN Task Board

| # | Slice | Task | Role | Model | Input files | Output | AC | Status |
|---|-------|------|------|-------|-------------|--------|----|--------|
| T1 | S1 | Implement X | Implementer | Sonnet | 04, 09 section D1 | apps/.../x.py | AC1: passes type check | Pending |
| T2 | S1 | Review X | Reviewer | Opus | T1 output, 03 | verdict | AC: no blocking issues | Pending |
| T3 | S1 | Test X | Tester | Sonnet | T1 output, 04 | tests/test_x.py | AC: all tests pass | Pending |
| T4 | S1 | [GATE] Slice S1 approval | -- | -- | -- | -- | User says GO | Pending |
```

Status values: `Pending` | `In Progress` | `Done` | `Failed` | `Blocked`

---

## 9. Research Integrity Rule

**No decision is created without a full entry in `09-research-findings-decisions.md`.**

Decision entry format:

```markdown
### D<N>: <Title>

**Date:** YYYY-MM-DD
**Context:** Why this decision was needed.
**Options considered:**
1. Option A -- pros / cons
2. Option B -- pros / cons

**Decision:** Option chosen and why.
**Consequences:** What this means for implementation.
```

---

## 10. Plan Creation Process (Step by Step)

1. Determine the next available plan code (scan `plan/index.md` or `plan/PLN-*` dirs).
2. Create the plan directory `plan/PLN-YYYY-NNN/`.
3. Create `00-overview.md` — scope, status (Draft), owner, date, acceptance criteria.
4. Create `01-functional-spec.md` — user stories, flows, edge cases.
5. Create `02-tech-spec-plain.md` — all 6 required sections.
6. Create tech spec developer shards (`03-` through `05-` or more).
7. Create research documents (`07-` through `10-`).
8. Create `CONTEXT.md` — compact orchestrator entry point per section 7 template.
9. Create `TASKS.md` — atomic task board per section 8 template.
10. Register in `plan/index.md`.
11. Verify completeness — all files exist, plan code consistent everywhere.
12. **Stop and request user confirmation. Do NOT proceed to implementation.**

---

## 11. Plan Closure Process

1. Verify implementation against scope (compare against AC in `00-overview.md`).
2. Require review by a separate review agent (not the Implementer).
3. Require test evidence (all tests pass).
4. Mark all acceptance criteria as completed in `00-overview.md`.
5. Set plan status to Done in `00-overview.md` and `CONTEXT.md`.
6. Update Plan Index — move from Active to Completed.
7. Verify documentation consistency.

### Definition of Done — Plan Ready for Implementation

- [ ] All mandatory plan files exist and are internally consistent.
- [ ] Scope and out-of-scope are explicitly defined.
- [ ] Acceptance criteria are measurable and testable.
- [ ] CONTEXT.md and TASKS.md are populated.
- [ ] One task file exists per TASKS.md row (in `tasks/T[N].md`).

### Definition of Done — Plan Fully Closed

- [ ] All acceptance criteria marked as completed.
- [ ] Plan Index and plan file statuses in sync.
- [ ] A separate Review agent has tested and approved.
- [ ] Plan status is `Done` in all locations.
