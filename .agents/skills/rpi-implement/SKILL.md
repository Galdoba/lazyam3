---
name: rpi-implement
description: How to execute the Implement phase of the RPI Strategy (Research > Plan > Implement) for agentic software engineering. Use this skill whenever the user wants to execute tasks from a validated plan.md, run the implementation workflow, or ship code from a FACTS-qualified plan. Trigger whenever the user mentions implementing a plan, executing tasks from plan.md, running the implementation phase, applying a plan, or wants to start building from a plan document. Also trigger when asked to execute tasks one at a time or all at once from a plan.
---

# RPI Implement Phase

Execute a validated technical implementation plan by completing atomic tasks, opening draft pull requests for human review, iterating on feedback, and delivering production-ready code through short-lived branches with squash merge.

**Goal**: Transform the FACTS-qualified task breakdown into working, tested code delivered via GitHub pull requests. Leave with all plan tasks marked complete, all quality gates passing, and clean branch state.

## When to Use This Skill

Use whenever the user wants to:
- Execute tasks from a validated `plan.md` file
- Run the implementation workflow (single task or all tasks)
- Ship code from a FACTS-qualified plan via pull requests
- Apply quality gates (build → lint → test) after each task
- Handle implementation failures and escalations

---

## Input Artifact

**Location**: `rpi/[problem-short-name]/plan.md`
**Requirement**: Must have passed FACTS Scale validation (Mean >= 3.00)

If the plan file does not exist or has not passed FACTS validation, stop and report the blocker. Do not proceed with an unvalidated plan.

---

## Output Artifact

**Location**: `rpi/[problem-short-name]/implementation.md`

Upon completion, produce `implementation.md` with this exact structure:

```markdown
# [Problem Statement] Implementation

## Summary
[What was built, scope delivered vs planned, overall status]

## Pull Requests
| PR | Description | Status |
|----|-------------|--------|
| [#NNN](url) | Brief description | Merged |

## Unexpected Discoveries
[Code patterns, constraints, surprising behaviors found during implementation that the plan could not anticipate]

## Plan Deviations
[Tasks that differed from the plan, with rationale: what changed, why, impact]

## Architecture Decision Records
[Key design decisions made during implementation: context, options considered, decision, rationale, consequences]

## Handoff Notes

### Known Limitations
- {limitation 1}
- {limitation 2}

### Data Models
- {TypeName}: {fields, purpose}

### API Surface
- {METHOD} {path}: {description}

## Reference
- {related file 1}
- {related file 2}
```

**Rules**: No frontmatter. All headings are H2 under an H1 title. H1 title pattern: `[Problem Statement] Implementation`. Each Architecture Decision Record follows a strict parseable format:

```markdown
### {ADR name}

context:
{description}

options:
- {option 1}
- {option 2}

decision:
{description}

rationale:
{description}

consequences:
- {consequence 1}
- {consequence 2}
```

Fields must appear in exact order: context, options, decision, rationale, consequences. Field headers on their own line only. `options` and `consequences` use bullet points.

---

## Implementation Workflow

### 1. Pre-Execution Validation

1. Verify `plan.md` exists at the expected location
2. Confirm FACTS Scale validation passed (Mean >= 3.00)
3. Parse the task breakdown and identify phases
4. Validate all referenced files are accessible in the codebase
5. Verify current git branch matches `implement/[problem-short-name]` using `scripts/check_branch.py problem-short-name`

### 2. Branch Setup

1. If not already on the implement branch, create it from main: `implement/[problem-short-name]`
2. Verify with `scripts/check_branch.py problem-short-name` before writing any code
3. This branch lives until all tasks are merged, then deleted (Branch Cleanup Discipline)

### 3. Task Execution Loop

For each phase in `plan.md`:

1. **Identify tasks** in the current phase
2. **Execute tasks sequentially** — no parallel execution; each task builds on the previous one
3. **For each task**:
   - Read the task description and any referenced code
   - Implement the specified changes following existing codebase patterns
   - Run self-validation: **build → lint → test** (Agent Self-Validation)
   - If self-validation passes, commit the changes
4. **After completing a logical group of tasks** (or after each task for large changes):
   - Push the branch
   - Open a **draft pull request** (Draft Pull Requests for Agents)
   - Mark the PR as ready for review once all self-validation gates pass
5. **Proceed to next phase** only when current changes are reviewed and merged

### 4. Review and Iteration

1. When a PR receives review comments, the agent **iterates on the PR** (Agent Iteration on Review Comments)
2. Push updates to the same branch; the PR auto-updates
3. Repeat until the reviewer approves or explicitly requests changes
4. Do NOT merge your own PRs — merge is always the human's responsibility (No Auto-Merge for Agents)

### 5. Merge Strategy

1. After human approval, the human merges the PR
2. Use **squash merge** to collapse agent commits into a single clean commit (Merge Strategies for Agent PRs)
3. After merge, the branch is automatically deleted (Branch Cleanup Discipline)

### 6. Quality Gates

After each task completion (except TDD "Red" phase where test failure is expected):

1. **Build** — project compiles/runs without errors
2. **Lint** — code passes linting with no violations
3. **Tests** — all tests pass (unit, integration, e2e as specified)

If any gate fails:
- Fix the issue and retry
- Do NOT mark the task complete until all gates pass
- If unresolved after reasonable attempts, classify per failure levels below

### 7. Completion Tracking

- Real-time updates to `plan.md` checkboxes
- Successful task: `- [ ]` → `- [x]`
- Failed tasks remain unchecked with noted blockers
- Preserve all plan metadata (FACTS scores, sections, etc.)

---

## Failure Classification

| Level | Trigger | Response |
|---|---|---|
| **Minor** | Single task quality gate failure (build/lint/test) | Fix and retry immediately |
| **Major** | 2+ consecutive task failures or critical path blocked with no clear resolution | Stop execution, report to user |
| **Critical** | 3+ consecutive failures, security issues, or fundamental design flaw | Stop execution, generate postmortem, escalate |

**On any failure:**
- Stop execution immediately
- Report complete error output and system state
- Do NOT mark the task as complete
- Do NOT proceed to the next task

**Postmortem (Critical failures):**
Generate `rpi/[problem-short-name]/implement-postmortem.md` with:
- Failed task and phase context
- Complete error details and system state at failure
- Root cause analysis
- Recommended solutions and recovery paths
- Context needed to restart the RPI Strategy process if required

---

## Task Execution Process

For each task to implement:

**a) Gather context**
- Read the task description from `plan.md`
- Read any referenced files, code locations, or components
- Understand the existing codebase patterns and conventions

**b) Implement the task**
- Make the necessary code changes
- Follow existing code patterns and conventions
- Ensure the implementation matches the task requirements
- Handle edge cases and error conditions
- Respect phase boundaries (complete all tasks in a phase before proceeding)

**c) Run self-validation** (build → lint → test)

**d) Commit and open draft PR** when a logical group of changes is ready

**e) Update plan file**
- Mark the completed task as done: `- [ ]` → `- [x]`
- Preserve all plan metadata

---

## Special Considerations

- **Phase boundaries**: Complete all tasks in a phase before starting the next phase
- **Task dependencies**: If a task depends on another unchecked task, implement the dependency first or report the blocker
- **Ambiguous tasks**: Document assumptions and note the ambiguity in the report
- **No parallel execution**: Execute tasks sequentially; short-lived branches pair with sequential delivery (Merge Strategies for Agent PRs)
- **Draft PR first**: Always open a draft PR for CI validation before marking ready for review
- **Small PRs**: Keep pull requests focused and small for faster review (Small Pull Requests)
- **File not found**: If the plan file doesn't exist, report error clearly
- **No tasks**: If all tasks are already complete, report this and take no action
- **Human merge only**: Never merge your own PR; the merge button is always the human's (Human Merge Decision)

---

## Deterministic Validation

Implementation artifacts can be programmatically validated using `scripts/validate_implement.py`. See `references/deterministic-checks.md` for the full list of enforced rules (H1 title ending in "Implementation", 6 required H2 sections in order, strict ADR format, no blank lines between consecutive bullets, Handoff Notes subsections, Reference section, plan reference resolution, ASCII-only prose, PR table present in output).

```bash
python3 .agents/skills/rpi-implement/scripts/validate_implement.py rpi/[problem-short-name]/implementation.md
```

Expected output on success: `VALID: implementation.md passes all deterministic checks`

---

## Anti-Goals

- Don't refactor beyond what's specified in the tasks
- Don't add features not in the task list
- Don't modify the Plan document except for checking off completed tasks
- Don't complete multiple phases unless explicitly told to do so
- Don't skip quality gates even under time pressure
- Don't leave tasks partially implemented
- Don't proceed with failing tests
- Don't merge your own pull requests
- Don't use parallel execution for tasks in the same phase
- Don't open PRs without self-validation passing first

---

## Handoff Contract Upon Completion

Implement phase completes when all of the following are true:
- All tasks in `plan.md` marked complete with `[x]`
- All tests passing (unit, integration, e2e)
- Build successful with no errors
- Lint passing with no violations
- Code ready for deployment
- `implementation.md` artifact produced and validated
- All PRs merged and branches deleted
- No uncommitted changes unless specified
- Documentation updates completed if specified in tasks

---

## References

- Full Implement phase spec: `docs/phases/Implement.md` in rpi-strategy repo
- Plan artifact (input): `rpi/[problem-short-name]/plan.md`
- FACTS Scale validation: `docs/scales/facts-scale.md` in rpi-strategy repo
- Previous phase (input): Research phase output via `rpi-research` skill
- PR workflow: Draft PRs, self-validation, review iteration, squash merge, branch cleanup
