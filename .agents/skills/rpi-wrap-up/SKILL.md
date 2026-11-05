---
name: rpi-wrap-up
description: Finalize an RPI cycle after implementation — extract Architecture Decision Records from implementation.md, validate format, persist to doc/adr/, archive intermediate documents. Use when the user wants to wrap up an RPI cycle, extract ADRs, validate them, or archive research/plan/implement documents after implementation. Trigger when the user mentions wrapping up, finalizing, archiving, or extracting ADRs from an RPI cycle.
---

# RPI Wrap-Up Phase

Finalize an RPI cycle after implementation. The **agent** evaluates, extracts, and writes ADRs. The **script** validates format.

**Goal**: Leave with all ADRs validated in `doc/adr/`, all intermediate docs archived in `rpi/archive/[name]/`, and `doc/adr/INDEX.md` updated.

## When to Use This Skill

Use when the user wants to:
- Wrap up an RPI cycle after implementation
- Extract and persist ADRs from `implementation.md`
- Archive `research.md`, `plan.md`, `implementation.md`

## Prerequisites

- `rpi/[name]/implementation.md` exists and has been produced by `rpi-implement`
- The `implement/[name]` branch has NOT been merged to main yet
- `doc/adr/` directory exists (create if not)

## Agent Workflow

### 1. Read implementation.md

Read `rpi/[name]/implementation.md`. Find the `## Architecture Decision Records` section. Extract each ADR block (H3 heading + fields: `context:`, `decision:`, `rationale:`, `consequences:`, `options:`).

### 2. Evaluate each ADR candidate

For each extracted ADR, the agent must decide: **is this a real ADR?**

**ADR threshold** — qualifies only if the decision:
- Spans ≥2 packages or systems
- Sets a precedent for future work
- Changes interfaces, contracts, or data models
- Has cross-cutting consequences

**Not an ADR** — implementation details, single-package changes, bug fixes, refactors, configuration tweaks. Skip these.

For each qualifying ADR, the agent determines:
- **scope**: `cross-package` (≥2 packages), `cross-system` (system boundaries), `architectural` (project-wide precedent)
- **description**: 1-2 sentences summarizing the decision in human-readable form
- **tags**: relevant categorization tags (`logging`, `architecture`, `infra`, etc.)

### 3. Read existing ADRs

Read all files in `doc/adr/` (skip `INDEX.md`). For each new ADR, check if any existing ADR has overlapping scope or duplicates the decision.

### 4. Mark superseded ADRs

If a new ADR supersedes an existing one:
1. Read the old ADR file from disk
2. Update `status: superseded`, add `superseded_by: "new-id"`
3. Write the updated file back
4. Set `supersedes: "old-id"` on the new ADR

### 5. Assign IDs

Find the highest existing ADR number in `doc/adr/` (parse `NNNN-` from filenames). Assign sequential IDs starting from `NNNN+1`.

### 6. Write ADR files

For each new ADR, create `doc/adr/NNNN-title.md`:

```yaml
---
id: "NNNN-title-slug"
title: "Title"
description: "1-2 sentence summary of the decision."
status: accepted
date: "YYYY-MM-DD"
scope: cross-package
---

# Title

## Context
[Agent-written context from the implementation.md context field]

## Decision
[Agent-written decision from the implementation.md decision field]

## Rationale
[Agent-written rationale from the implementation.md rationale field]

## Consequences
- [Agent-written consequences as bullet list]

## Options
- [If present in implementation.md]
```

**Agent writes the content** — do not copy verbatim from implementation.md. Refine, restructure, and ensure clarity. The ADR should be readable by someone who was not part of the implementation.

### 7. Validate

Run validation script on all ADRs:

```bash
python3 .agents/skills/rpi-wrap-up/scripts/validate_adr.py --all doc/adr/
```

If validation fails, fix the ADR files and re-run. Do not proceed until all ADRs pass.

### 8. Regenerate INDEX.md

Create `doc/adr/INDEX.md` with a table of all ADRs:

| ID | Title | Status | Date | Scope |
|----|-------|--------|------|-------|

### 9. Archive intermediate documents

Move `rpi/[name]/research.md`, `rpi/[name]/plan.md`, `rpi/[name]/implementation.md` to `rpi/archive/[name]/`.

### 10. Clean up

If `rpi/[name]/` is now empty, remove the directory.

## ADR Format Specification

### YAML Frontmatter (required fields)

| Field | Required | Description |
|-------|:---:|-------------|
| `id` | Yes | Unique `NNNN-title-slug`, 4 digits |
| `title` | Yes | Matches H1 in markdown |
| `description` | Yes | 1-2 sentence summary of the decision |
| `status` | Yes | `accepted` \| `deprecated` \| `superseded` |
| `date` | Yes | `YYYY-MM-DD` |
| `scope` | Yes | Impact scope: `cross-package` \| `cross-system` \| `architectural` |

**ADR threshold**: only decisions with `scope` of `cross-package` or higher qualify as ADRs. Implementation-level details are not ADRs.

### Markdown Body (required sections)

| Section | Description |
|---------|-------------|
| `## Context` | Situation, constraints, what existed before the decision |
| `## Decision` | Full detailed explanation of what was decided |
| `## Rationale` | Why this decision, comparison with alternatives |
| `## Consequences` | Positive and negative consequences (bullet list) |

### Markdown Body (optional sections)

| Section | When needed |
|---------|-------------|
| `## Options` | When there were alternatives considered |
| `## Related` | When linking to other ADRs or docs |

### Section order (fixed)

`## Context` → `## Decision` → `## Rationale` → `## Consequences` → `## Options` → `## Related`

## Conflict Resolution

When a new ADR supersedes an existing one:
1. Update old ADR file on disk: set `status: superseded`, add `superseded_by: "new-id"`
2. Set `supersedes: "old-id"` on the new ADR
3. Both files must pass validation after update

## Deterministic Validation

```bash
python3 .agents/skills/rpi-wrap-up/scripts/validate_adr.py <adr-file>
# Expected: VALID: ADR passes all checks
```

```bash
python3 .agents/skills/rpi-wrap-up/scripts/validate_adr.py --all <adr-dir>
# Validates all ADRs, cross-references, and no cycles
# Expected: VALID: all ADRs pass checks
```
