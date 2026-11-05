---
name: rpi-research
description: How to execute the Research phase of the RPI Strategy (Research > Plan > Implement) for agentic software engineering. Use this skill whenever the user wants to research a problem, investigate a bug, gather code context, or produce a research document before planning or implementing. Trigger whenever the user mentions debugging, problem investigation, code analysis, gathering context before coding, building a research document, or wants to apply FAR scoring to validate research findings. Also trigger when asked to launch parallel research agents or produce a research.md artifact.
---

# RPI Research Phase

Establish a high-fidelity, action-ready understanding of a problem by clarifying scope, mapping affected code surfaces, gathering internal and external technical context, and synthesizing it into a single durable Markdown artifact for the Plan phase.

**Goal**: Remove ambiguity before any solutioning. Leave with a crisply stated problem, success definition, affected code surfaces, option tradeoffs, and a recommended direction (NOT a detailed build plan).

## When to Use This Skill

Use whenever the user wants to:
- Research a bug, feature, or technical problem before coding or planning
- Investigate what's happening in the codebase around a specific issue
- Gather code context and affected file locations
- Produce a structured research document (`research.md`)
- Validate research findings with the FAR Scale
- Launch parallel research agents for comprehensive investigation

---

## Output Artifact

Create `rpi/[problem-short-name]/research.md` with this exact structure:

```markdown
# [Problem Statement] Research

## Problem Context
- Restated, clarified problem statement
- Business/functional intent
- Current vs desired behavior
- Constraints (time, performance, compliance, environment)

## Affected Files
```
path/to/file.ext:LINE
path/to/file.ext:START-END
```
[Explanatory prose as bullet points keyed by path]

## Code Examples
[Minimal, relevant snippets showing current vs proposed shape]

## External Research Findings
[Web research: official docs, best practices, known issues — with sources]

## FAR Scale Output
### Factual Score: [0-5]
### Actionable Score: [0-5]
### Relevant Score: [0-5]
### Mean: [X.XX]  --> PASS/FAIL

## Testing Strategy
[Unit test touchpoints, integration surfaces, observability, repro steps]

## Potential Design Pattern Recommendations
[Candidate patterns with rationale and internal exemplars]

## Assumptions
[Enumerated, falsifiable statements]

## Out of Scope
[Explicit exclusions to prevent scope creep]
```

**Rules**: No frontmatter. All headings are H2 under an H1 title. H1 title pattern: `[TICKET-ID] Research` (or `Use Case Research` if no ticket ID).

---

## Code Reference Syntax

Use fenced code blocks (```) with one reference per line:

- Single location: `path/to/file.ext:LINE` or `path/to/file.ext:LINE:COL`
- Range: `path/to/file.ext:START-END`
- Range with columns: `path/to/file.ext:START_LINE:START_COL-END_LINE:END_COL`
- Multiple ranges: `path/to/file.ext:RANGE1;RANGE2`
- Entire file: `path/to/file.ext` (no line number)
- Order by research/anticipated diff sequence, **not** alphabetically

Provide explanatory prose as bullet points beneath the code block, keyed by the same path (keep references machine-parseable).

---

## Research Workflow

1. **Clarify the problem** — restate precisely, identify what's known vs unknown, define success criteria
2. **Analyze the codebase** — trace actual code paths, entry points, data flow, state changes
3. **Fetch external context** — framework docs, best practices, known issues, benchmarks
4. **Synthesize findings** — compile into the research.md structure above
5. **Apply FAR scoring** — validate with the FAR Scale rubric
6. **Loop if needed** — any FAR dimension failing threshold → restart at step 1 (retaining previously gathered artifacts unless invalidated)

### Parallel Research Agents

For complex problems, launch sub-agents simultaneously (single message, multiple Task calls):

| Agent | Purpose |
|-------|---------|
| **codebase-locator** | Find WHERE relevant code lives (files, directories, components) |
| **codebase-analyzer** | Understand HOW the code works (entry points, data flow, patterns) |
| **codebase-pattern-finder** | Discover similar implementations and existing patterns |
| **web-search-researcher** | Find external information (docs, best practices, known issues) |

Launch all agents with the same problem statement. Wait for all to complete before synthesizing.

---

## FAR Scale (Research Validation)

Score each dimension 0-5. Research must pass all thresholds to proceed to Plan.

| Dimension | Pass Threshold | Question |
|-----------|---------------|----------|
| **Factual** (F) | ≥ 4 | Is this evidenced in the code/system? |
| **Actionable** (A) | ≥ 3 | Can I move forward now? |
| **Relevant** (R) | ≥ 3 | Does it matter to this problem now? |
| **Mean** | ≥ 4.00 | Overall quality |

Scoring format:
```
F: [score]  A: [score]  R: [score]  Mean: [X.XX]  --> PASS/FAIL
```

See `references/far-scale.md` for the full 0-5 rubric per dimension, including scoring cues and examples.

### Failure Handling (Research)

| Severity | FAR Mean | Action |
|----------|---------|--------|
| Minor | 3.5–3.9 | Single iteration refinement |
| Major | 2.0–3.4 | Restart Research phase |
| Critical | < 2.0 | Leadership escalation, problem may be ill-defined |

---

## Quality Gates Before Completion

- Every code reference resolves (file exists + line within bounds)
- No snippet exceeds ~40 lines unless justified
- All assumptions have a planned validation step or explicit deferral note
- FAR table present with numeric values + computed mean
- Failure handling procedures documented if FAR validation fails

---

## Anti-Goals

- Don't propose final solution architecture
- Don't write production code
- Don't perform refactors beyond evidentiary inspection
- Don't estimate implementation timelines
- Don't produce a detailed build plan (that's the Plan phase)

---

## References

- Full FAR Scale rubric: `references/far-scale.md`

## Deterministic Validation

After producing `research.md`, verify it against deterministic checks using:

```bash
python3 .agents/skills/rpi-research/scripts/validate_research.py rpi/[problem-short-name]/research.md
```

The script enforces: no frontmatter, H1 title pattern, H2 heading hierarchy, FAR table presence with numeric scores, FAR thresholds (F>=4, A>=3, R>=3, Mean>=4.00), correct mean calculation, code reference resolution, snippet length limits, and ASCII-only text outside code blocks/inline code.
