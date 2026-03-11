# Task Prompt Templates

Standard prompt templates and task file specifications for the plan and implementation workflow.

---

## 1. How the system works

### Task files vs TASKS.md

Every task in TASKS.md has a corresponding **task file** at `plan/PLN-YYYY-NNN/tasks/T[N].md`. Created by the Planner sub-agent during the plan phase, they serve as:

1. **Self-contained agent instructions** — the Orchestrator passes the file path to the Task tool.
2. **Single source of truth** — agents append their work log directly into the task file.

### How the Orchestrator uses task files

1. Reads `plan//tasks/T[N].md` (just the header — Role, Model, and AC sections)
2. Passes the **full file path** to the sub-agent as the prompt
3. The sub-agent reads the task file, performs the work, and **appends its work log** under `## Work Log`
4. The Orchestrator re-reads the task file to validate the AC section

### Sub-agent rules

- Sub-agents read ONLY files listed in their task file's "Read" section.
- Sub-agents append to `## Work Log`. They never modify the static sections above the `---` divider.
- Sub-agents never modify TASKS.md, CONTEXT.md, or plan index files.
- One sub-agent, one task file. Never batch multiple task files into one sub-agent.

---

## 2. Task File Format

Every `plan/PLN-YYYY-NNN/tasks/T[N].md` follows this exact structure:

```markdown
# T[N] — [Task Title]

**Plan:** PLN-YYYY-NNN
**Slice:** S[N]
**Role:** [Implementer | Code Quality Reviewer | Tester | Planner]
**Model:** [sonnet | opus]

## Context

[1-3 sentences describing what this task is for and why it matters. Reference key decisions (e.g., "per D3, D7").]

## Read

> Sub-agent: read ONLY the files listed here. Do not read any other files.
> If you need a file not listed, stop and report: DECISION NEEDED: [file] required but not listed.

- plan/PLN-YYYY-NNN/CONTEXT.md
- [primary spec file]
- [secondary spec file or implemented file if this is a review/test task]

## Instructions

[Detailed, step-by-step instructions. Reference spec sections and decisions. A fresh agent must be able to execute by reading only this file and the files in the Read list.]

## Output

[Exact file path(s) to produce or modify.]

## Acceptance Criteria

- [ ] [AC item 1 — measurable and independently verifiable]
- [ ] [AC item 2]

## Do NOT

- [Role-specific prohibitions]
- Read files not listed in the Read section above
- Modify TASKS.md, CONTEXT.md, or plan index files

---

## Work Log

> This section is appended by agents during execution. Static sections above this line are read-only.

### Files Touched

[Agent appends: list of files created or modified]

### Subtasks Completed

- [ ] [AC item 1] — [brief note]

### Test Results

[Commands run and their output]

### Conclusion

[Agent appends: PASS or FAIL, plus one-paragraph summary]

### Review Findings

[Reviewer sub-agent appends here]

### Codex Findings

[Codex appends here via the Orchestrator]
```

---

## 3. Planner Template

```
Role: Planner
Model: opus
Task: [task title from TASKS.md]

Read and execute: plan/[PLN-YYYY-NNN]/tasks/[T#].md

This file contains your full instructions, the files to read, your deliverable, and acceptance criteria.
After completing your work, append your results to the ## Work Log section of that file.
```

---

## 4. Implementer Template

```
Role: Implementer
Model: sonnet
Task: [task title from TASKS.md]

Read and execute: plan/[PLN-YYYY-NNN]/tasks/[T#].md

This file contains your full instructions, the files to read, your deliverable, and acceptance criteria.
After completing your work, append your results to the ## Work Log section of that file.
```

---

## 5. Code Quality Reviewer Template

```
Role: Code Quality Reviewer
Model: opus
Task: [task title from TASKS.md]

Read and execute: plan/[PLN-YYYY-NNN]/tasks/[T#].md

This file contains your full instructions, the files to read, your review checklist, and acceptance criteria.
After completing your review, append your findings to the ## Review Findings subsection of the ## Work Log.
Output your verdict as PASS or FAIL with specific issues listed.

Additionally, assess the quality of the task file itself:
- Was the Read list correct and complete for the deliverable?
- Were the Instructions specific enough, referencing spec sections and D-codes?
- Do the AC items test what the spec actually requires?
Append this as "Task File Quality Notes" at the end of ## Review Findings. This is advisory — it does not affect your PASS/FAIL verdict.
```

---

## 6. Tester Template

```
Role: Tester
Model: sonnet
Task: [task title from TASKS.md]

Read and execute: plan/[PLN-YYYY-NNN]/tasks/[T#].md

This file contains your full instructions, the files to read, your test deliverable, and acceptance criteria.
After completing your work, append your results to the ## Work Log section.
Include the full test command output under ## Test Results.
```

---

## 7. Codex Second-Eyes Reviewer (Orchestrator invokes via Bash)

The Codex Second-Eyes Reviewer is invoked by the Orchestrator via the Bash tool. NOT a Claude sub-agent. Uses `codex` CLI 0.104+.

**Always use `codex exec --full-auto -C <dir>` with an explicit prompt. Do NOT use `codex exec review`.**

### Gemini fallback

If `codex exec` fails (non-zero exit AND output contains "rate limit", "quota", or "session limit"):

```bash
gemini -y -p "<same prompt as Codex>" > /tmp/codex-review-[PLN-YYYY-NNN]-<context>.md 2>&1
```

---

### Plan review — invocation

```bash
codex exec --full-auto -C <repo_root> \
  -o /tmp/codex-review-plan-[PLN-YYYY-NNN].md \
  "$(cat <<'EOF'
<plan review prompt — see below>
EOF
)"
```

### Plan review prompt

```
You are a senior engineer performing a pre-implementation review of a plan.

Plan location: plan/[PLN-YYYY-NNN]/
Task files location: plan/[PLN-YYYY-NNN]/tasks/

Read every file in plan/[PLN-YYYY-NNN]/ and every task file in plan/[PLN-YYYY-NNN]/tasks/.

Then produce a structured review covering ALL of the following dimensions. For each item output one line:
[N]. [CHECK NAME] | [PASS / FAIL] | [explanation — one sentence]

## Review dimensions

### 1. Security & data integrity
- Sensitive data (credentials, PII) handled correctly — no hardcoded secrets in specs
- Auth flows described have no obvious bypasses
- No insecure patterns proposed

### 2. Architecture & best practices
- Tech choices align with the existing stack
- No anti-patterns: sync calls in async context, N+1 queries, unbounded loops
- Money values handled as Decimal or integer cents — never float

### 3. Completeness & consistency
- Every deliverable in 00-overview.md has a corresponding TASKS.md task
- TASKS.md task sequence for each deliverable: Implementer → Reviewer → Tester
- No contradictions between spec files

### 4. Task file quality

For EVERY task file in tasks/T[N].md, read the file in full and check:

**4a. Existence and structure**
- Every TASKS.md row has a corresponding task file
- Each task file contains all mandatory sections: Context, Read, Instructions, Output, Acceptance Criteria, Do NOT, Work Log

**4b. Read list correctness**
- The Read list includes CONTEXT.md
- The Read list includes the correct primary spec file(s) for this specific deliverable
- For Reviewer tasks: includes the implemented output file(s) being reviewed
- For Tester tasks: includes the implementation file(s) AND the test strategy spec
- No unnecessary files listed; no files that don't exist yet at the time this task runs

**4c. Instructions quality**
- For Implementer tasks: references specific spec sections and D-codes
- For Reviewer tasks: enumerates exactly what to check
- For Tester tasks: includes exact test commands
- Instructions do not leave execution steps to guesswork

**4d. Acceptance Criteria quality**
- AC items are measurable and independently verifiable
- AC items test what the spec actually requires
- For Reviewer tasks: includes a verdict criterion
- For Tester tasks: includes a pass criterion for the test run

**4e. Do NOT section**
- Contains role-appropriate prohibitions
- Prohibits reading files outside the Read list

### 5. Code quality signals in the spec
- API endpoints follow RESTful conventions
- Pydantic schemas specified for request/response bodies
- Migration strategy described; rollback plan exists
- Test coverage targets stated

### 6. Operational readiness
- Environment variables / secrets required are listed
- Monitoring / alerting implications noted

## Output format

Output a numbered checklist. Each line:
[N]. [CHECK NAME] | [PASS / FAIL] | [explanation — one sentence]

After the checklist:
OVERALL: PASS (if ALL items PASS) or FAIL (list of item numbers that failed)

Do not suggest fixes. Only report findings.
```

---

### Implementation review — invocation

```bash
codex exec --full-auto -C <worktree_dir> \
  -o /tmp/codex-review-[PLN-YYYY-NNN]-T[N].md \
  "$(cat <<'EOF'
<implementation review prompt — see below>
EOF
)"
```

### Implementation review prompt

```
You are a senior engineer performing a second-eyes review of an implementation task.

Plan: [PLN-YYYY-NNN]
Task: [T# — task title]

Read:
- plan/[PLN-YYYY-NNN]/CONTEXT.md
- plan/[PLN-YYYY-NNN]/tasks/[T#].md
- plan/[PLN-YYYY-NNN]/[primary spec file for this task]

Then read the implemented output files listed in the task file's Output section.

Verify the implementation against the plan and task brief. For each check output one line:
[N]. [CHECK NAME] | [PASS / FAIL] | [explanation — one sentence]

## Review dimensions

### 1. Spec compliance
- Implementation matches what the spec and task brief describe
- All AC from the task file are met
- No scope creep

### 2. Security
- No hardcoded secrets
- Input validation present (Pydantic schemas for all external data)
- No obvious injection vectors

### 3. Code quality
- Type hints on all function signatures
- Async/await used correctly
- Error handling: exceptions caught at appropriate level, not swallowed silently

### 4. Test coverage
- Tests cover the happy path
- Tests cover at least one error/edge case
- Tests are isolated

### 5. Integration fit
- New code follows existing patterns
- Imports and module structure consistent
- No duplicate functionality

### 6. Task file quality (advisory — does not affect OVERALL verdict)

Read plan/[PLN-YYYY-NNN]/tasks/[T#].md in full. Assess the task file quality:

6a. READ LIST ADEQUACY | [PASS / FAIL] | [explanation]
6b. INSTRUCTION SPECIFICITY | [PASS / FAIL] | [explanation]
6c. AC ALIGNMENT | [PASS / FAIL] | [explanation]
6d. SCOPE GUARD | [PASS / FAIL] | [explanation]

## Output format

Output a numbered checklist. Each line:
[N]. [CHECK NAME] | [PASS / FAIL] | [explanation — one sentence]

After the checklist:
OVERALL: PASS (if ALL items 1–5 PASS) or FAIL (list of item numbers that failed)

Note: FAIL on dimension 6 is advisory only. OVERALL verdict is based on dimensions 1–5.
```

---

### Iteration protocol (plan and implementation)

**Round 1:**
1. Run Codex (or Gemini fallback).
2. Read output file.
3. If `OVERALL: PASS` → append findings to task file's `## Codex Findings` → proceed.
4. If `OVERALL: FAIL` → record exact failing item lines to `/tmp/codex-findings-[PLN-YYYY-NNN]-<context>.md`.

**Fix iteration:**
5. Spawn Claude sub-agent (Planner for plan issues, Implementer for code issues):
   - "Read the task file — the ## Codex Findings section lists the issues to fix."
   - "Address ALL FAIL items. Do not change anything that was PASS."

**Re-verification — target only original failing items:**
6. Run Codex again with:

```
You are verifying that previously reported findings have been resolved.

Plan location: plan/[PLN-YYYY-NNN]/

In a previous review, the following items FAILED:
[Paste exact FAIL lines]

Re-read the same files. For Dimension 4 findings: read every task file listed in the failing items in full.
For each previously failing item ONLY, re-check and output:
[N]. [ORIGINAL CHECK NAME] | [PASS / FAIL] | [explanation]

OVERALL: PASS (all previously-failing items now PASS) or FAIL (list which still fail)

Do not introduce new findings. Only re-check the original failing items.
```

7. If `OVERALL: PASS` → append to `## Codex Findings` → proceed.
8. If `OVERALL: FAIL` → go back to step 5. **No iteration cap.**

---

## 8. Decision Escalation Template

When an Implementer needs a decision not covered by the spec, they append to `## Work Log → ## Conclusion`:

```
DECISION NEEDED — stopping implementation

Decision required: [clear description]
Context: [why this came up, what the options are]
Impact: [which files/deliverables are affected]

Waiting for Orchestrator to assign a Planner sub-agent to document this decision.
```

---

## 9. Slice Gate Report Template

```
SLICE GATE: [S#] — [slice title] complete — approval needed

## Increment delivered
[One-sentence description of what works after this slice merges — from Slice Map]

## Completed tasks
[List of Done tasks in this slice]

## What was produced
[Files created or modified, grouped by deliverable]

## Quality status
[Summary of Reviewer verdicts, Tester results, Codex verdicts]

## Next slice
[What S[N+1] will deliver — from Slice Map]

## Approval needed
Type "approve" to finalize this slice and push for review.
After review passes, merge to main, then: /implement-spec PLN-YYYY-NNN S[N+1]
```
