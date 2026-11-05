---
name: rpi-plan
description: How to execute the Plan phase of the RPI Strategy (Research > Plan > Implement) for agentic software engineering. Use this skill whenever the user wants to turn research findings into a concrete, executable technical implementation plan with atomic tasks. Trigger whenever the user mentions planning, task breakdown, implementation plan, task decomposition, breaking down a solution, creating a plan from research, or wants to produce a plan.md artifact. Also trigger when asked to validate a plan with the FACTS Scale.
---

# RPI Plan Phase

Transform Research phase findings into a concrete, executable technical implementation plan with atomic tasks that can be performed by AI coding agents.

**Goal**: Create a validated sequence of actionable development tasks that bridge problem understanding to production-ready code delivery. Leave with bounded scope, milestone map, dependency ordering, risk register, and measurable success criteria.

## When to Use This Skill

Use whenever the user wants to:
- Turn research findings into a concrete implementation plan
- Break down a solution into atomic, independently executable tasks
- Create a task breakdown with explicit dependencies and sequencing
- Validate a technical plan with the FACTS Scale
- Produce a structured plan document (`plan.md`)

---

## Input Artifact

**Location**: `rpi/[problem-short-name]/research.md`
**Requirement**: Must have passed FAR Scale validation (F >= 4, A >= 3, R >= 3, Mean >= 4.00)

If the research artifact does not exist or has not passed FAR validation, stop and prompt the user to complete the Research phase first.

---

## Output Artifact

Create `rpi/[problem-short-name]/plan.md` with this exact structure:

```markdown
# [Problem Statement] Plan

## Implementation Overview
- High-level approach derived from Research findings
- Key architectural decisions and patterns to be implemented
- Integration points with existing systems
- Performance and scalability considerations

## Task Breakdown
Organized into logical phases with atomic tasks using checkboxes.
Each task must be:
- Single responsibility (one clear objective)
- Independently completable by an AI agent
- Verifiable upon completion
- Appropriately sized (atomic units like single file edits or command calls)

Format:
## Phase 1: [Phase Name]
- [ ] Task description with specific file paths and changes
- [ ] Next atomic task in sequence

## Phase 2: [Next Phase Name]
- [ ] First task of next phase

## Code References
Comprehensive list of all files and locations that will be modified or created.
Representation: fenced code block using code reference syntax below.

## Testing Plan
- Unit tests for new functionality
- Integration tests for system interactions
- End-to-end test scenarios
- Performance test requirements
- Regression test considerations

## FACTS Scale Output
F: [score]  A: [score]  C: [score]  T: [score]  S: [score]  Mean: [X.XX]  --> PASS/FAIL

## Dependencies & Sequencing
- Inter-task dependencies within and across phases
- Critical path identification
- External system dependencies

## Risk Mitigation
- Technical risks identified during planning
- Mitigation strategies for each risk
- Contingency plans for high-impact scenarios
- Early warning indicators
- Escalation criteria

## Rollback Strategy
- Rollback points for each phase
- Data migration reversal procedures
- Feature flag configurations
- Service deployment rollback plans
```

**Rules**: No frontmatter. All headings are H2 under an H1 title. H1 title pattern: `[TICKET-ID] Plan` (or `Use Case Plan` if no ticket ID).

---

## Code Reference Syntax

Use fenced code blocks (```) with one reference per line:

- Single location: `path/to/file.ext:LINE` or `path/to/file.ext:LINE:COL`
- Range: `path/to/file.ext:START-END`
- Range with columns: `path/to/file.ext:START_LINE:START_COL-END_LINE:END_COL`
- Multiple ranges: `path/to/file.ext:RANGE1;RANGE2`
- Entire file: `path/to/file.ext` (no line number)
- New files (to be created): `path/to/new/file.ext [NEW]`

Order by **implementation sequence**, not alphabetically.

---

## Planning Workflow

1. **Analyze requirements** — re-read research.md, confirm FAR pass, identify solution direction
2. **Define implementation phases** — group work into logical, sequential phases
3. **Break down into atomic tasks** — each task: single responsibility, independently completable, verifiable
4. **Map code references** — list all files to be created or modified
5. **Plan testing strategy** — unit, integration, e2e, performance, regression
6. **Apply FACTS scoring** — validate the plan against all five dimensions
7. **Document dependencies, risks, and rollback** — complete the remaining sections
8. **Loop if needed** — any FACTS dimension failing threshold → restart at step 2 (retaining research findings)

### Task Design Principles

- **Single Responsibility**: Each task accomplishes exactly one objective
- **Atomic**: Each task is an indivisible unit (single file edit, single command call, etc.)
- **Verifiable**: Each task has clear pass/fail completion criteria
- **Context-Rich**: Include enough context for an AI agent to execute independently
- **Balanced**: Phases should have roughly similar complexity
- **Sequential**: Tasks within a phase must form a linear dependency chain; no parallel execution paths (aligns with Implement phase constraint)

---

## FACTS Scale (Plan Validation)

Score each dimension 0-5. Plan must pass all thresholds to proceed to Implement.

| Dimension | Question |
|-----------|----------|
| **Feasibility** (F) | Can this be implemented with available resources and skills? |
| **Atomicity** (A) | Is this task focused on a single responsibility? |
| **Clarity** (C) | Are execution order and dependencies unambiguous? |
| **Testability** (T) | Can completion be verified? |
| **Size** (S) | Is the task appropriately scoped? |

**Pass criteria**: Mean >= 3.00 across all five dimensions.

Scoring format:
```
F: [score]  A: [score]  C: [score]  T: [score]  S: [score]  Mean: [X.XX]  --> PASS/FAIL
```

See `references/facts-scale.md` for the full 0-5 rubric per dimension, including scoring cues and examples.

### Failure Handling (Plan)

| Severity | FACTS Mean | Action |
|----------|-----------|--------|
| Minor | 2.8–2.9 | Single iteration task refinement |
| Major | 2.0–2.7 | Return to Research phase for problem decomposition |
| Critical | < 2.0 | Leadership escalation, plan may be fundamentally unsound |

---

## Quality Gates Before Completion

- Every task maps to specific code locations or clear creation instructions
- All tasks are atomic and independently verifiable
- Dependencies are explicitly defined and non-circular
- Testing strategy covers all new functionality
- FACTS table present with numeric values + computed mean
- Rollback strategy defined for each implementation phase
- Failure handling procedures documented if FACTS validation fails

---

## Deterministic Validation

Plan artifacts can be programmatically validated using `scripts/validate_plan.py`. See `references/deterministic-checks.md` for the full list of enforced rules (H1 title ending in "Plan", 8 H2 sections in order, task checkboxes, FACTS thresholds/calculation, code reference resolution, ASCII-only prose).

```bash
python3 .agents/skills/rpi-plan/scripts/validate_plan.py rpi/[problem-short-name]/plan.md
```

Expected output on success: `VALID: plan.md passes all deterministic checks`

---

## Anti-Goals

- Don't write production code (leave to Implement)
- Don't make implementation decisions better left to coding agents
- Don't over-specify solutions that constrain implementation creativity
- Don't create tasks too large for a single AI agent session
- Don't skip FACTS validation — it's the gate to Implement

---

## Handoff Contract to Implement Phase

Implement phase receives:
- Final plan.md with FACTS pass indicators
- Complete task breakdown with checkboxes
- Normalized code references for all changes
- Testing validation criteria for each task
- Rollback procedures and checkpoints
- Dependency ordering for parallel execution

---

## References

- Full FACTS Scale rubric: `references/facts-scale.md`
- Research phase (input): `../rpi-research/SKILL.md`
