---
name: discovery-formalizer
description: Help users formalize rough feature ideas into structured Problem Statement documents. Use when the user says they're in "Discovery mode", mentions a rough idea for a new feature, or asks to document a problem/requirement. This skill is the first step in the discovery-to-implementation pipeline — it turns fuzzy thoughts into a well-structured, evidence-backed document ready for review. Trigger whenever the user says "I have an idea", "let's figure out X", "write up a problem statement", or explicitly declares discovery mode in AGENTS.md or the conversation.
---

# Discovery Formalizer

This skill helps turn a rough feature idea into a well-structured **Problem Statement** document following the project's conventions.

## Workflow

### Phase 1: Intake

1. Read the user's rough notes, description, or idea dump.
2. Identify the **core problem** or **desired change** the user is describing.
3. If the user has not mentioned a feature ID, create one: `<DOMAIN>-<NNN>` (e.g., `INGEST-003`, `PIPELINE-002`). Use a domain abbreviation relevant to the problem. Search the codebase for existing feature IDs to avoid collisions (grep `doc/` for `problem_id:`).
4. Confirm with the user: "Here's what I understand — the problem is **X**, affecting **Y**. Is that right?"

### Phase 2: Elicitation

Explore the codebase to fill gaps the user might have missed. Look at:
- **Relevant source files**: grep for keywords, trace data flow, find the code paths involved.
- **Existing docs**: check `doc/INDEX.md`, `doc/feature_list.json`, `AGENTS.md` for context.
- **Related problems**: are there similar issues already documented?

Ask the user targeted questions about:
- **Current behavior**: how does the system handle this now? (find code if possible)
- **Desired behavior**: what should happen instead?
- **Impact**: who/what is affected? what breaks if this isn't done?
- **Constraints**: technical or business constraints (performance, backward compatibility, domain purity, etc.)
- **Risks**: what could go wrong?
- **Non-goals**: what's explicitly out of scope?
- **Evidence**: code references, logs, user reports, architecture notes

Be proactive — don't wait for the user to answer everything. Fill in as much as you can from code analysis first, then ask only about gaps.

### Phase 3: Draft

Write the Problem Statement document to `doc/<feature-id>_problem_statement.md`.

### Document conventions

- **ASCII-only**: all content must use only ASCII characters (0x20-0x7E, plus tab/newline). No UTF-8 — no em-dashes, no ellipsis, no accented letters. Use `--` instead of em-dashes, `...` instead of ellipsis.
- **Strict frontmatter field order** (do not reorder): `problem_id` → `title` → `repository` → `baseline_commit` → `problem_type` → `status`
- **YAML literal block scalars** (`|`) for multi-paragraph fields (`current_behavior`, `desired_behavior`, `rationale`)
- **Markdown bullet lists** with `  - ` (2 spaces + dash)
- **`##` headings** with exactly one space after `##`
- `problem_type` must be one of: `feature`, `bug`, `improvement`
- `status` must be `DRAFT`
- `baseline_commit` must be the current HEAD commit hash (not "unspecified")

Use the following template (YAML frontmatter + markdown body):

```markdown
---
problem_id: <DOMAIN>-<NNN>
title: <short descriptive title>
repository: github.com/Galdoba/lazyam3
baseline_commit: <most recent relevant commit hash, or "unspecified">
problem_type: feature|bug|improvement
status: DRAFT
---

## current_behavior: |
  <Describe what the system does today. Include specific file paths and line numbers where possible.>

## desired_behavior: |
  <Describe the desired state after the change. Be specific and actionable.>

## rationale: |
  <Why does this matter? What happens if it's not done?>

## evidence:
  - type: <user_report|code_review|architecture|log_analysis>
    reference: <file:line or doc reference>
  # add more as needed

## acceptance_criteria:
  - criterion: <verifiable statement>
    verification_method: <how to test/verify>
  # one per criterion

## non_goals:
  - <what's explicitly out of scope>

## constraints:
  - <technical or business constraints>

## risks:
  - <potential issues or failure modes>

## open_questions:
  - <unresolved items, or "none" if all answered>
```

**Template rules:**
- Use YAML literal block scalars (`|`) for multi-paragraph fields (`current_behavior`, `desired_behavior`, `rationale`).
- Each `acceptance_criteria` entry must have both `criterion` and `verification_method`.
- `evidence` entries must have `type` and `reference`.
- `status` starts as `DRAFT`.

### Phase 4: Review

1. Present the draft to the user.
2. Ask: "Does this accurately capture the problem? Anything missing or wrong?"
3. Iterate until the user is satisfied.
4. The document is now ready for review/approval.

### Landing Gate (output validation)

**MUST run the landing gate script after the document is finalized. Do NOT skip it. Do NOT do manual checks.**

```bash
bash .agents/skills/discovery-formalizer/scripts/landing.sh doc/<id>_problem_statement.md
```

If landing returns FAIL, fix the reported issues and rerun.
If landing returns WARN, review the warnings, fix if needed, and rerun.
Only proceed to solution-search when landing returns PASS.

## Discovery Mode Integration

When the user declares "discovery mode" (either by saying so or by setting it in `AGENTS.md`):
1. Load this skill.
2. Ask: "What idea or problem would you like to formalize?"
3. Proceed through the workflow above.

## Tips

- **Be proactive with code exploration.** Don't just ask questions — search the codebase first. If you can find the relevant code path, cite it in the document.
- **Be honest about gaps.** If you can't find evidence for something, note it as an open question rather than guessing.
- **Keep it lean.** The document should be comprehensive but not padded. Every section should add value.
- **Use the user's language.** Don't reword their problem into jargon they wouldn't recognize.
- **One document at a time.** This skill produces a single artifact. If the user wants a Solution Options doc or Decision Record next, that's a separate skill invocation.
