---
name: solution-search
description: Help users find technical solutions for problems defined in a Problem Statement. Use when the user asks "how to implement", "what's the best approach", "find solutions", or needs technical options for a feature. Trigger whenever the user is in "solution search" mode, wants to brainstorm approaches for a defined problem, or needs to evaluate tradeoffs before implementation. Also trigger when the user references a Problem Statement file and asks how to build it.
---

# Solution Search

This skill guides systematic technical solution discovery for problems defined in a **Problem Statement** document. It produces a **Solution Verdict** document with evaluated approaches and a go/no-go decision.

## Workflow

### Pre-flight: Go/No-Go Gate

**MUST run the pre-flight gate script before starting solution analysis. Do NOT skip it.**

```bash
bash .agents/skills/solution-search/scripts/preflight.sh doc/<id>_problem_statement.md
```

The pre-flight gate checks:
- **Git clean**: working tree has no uncommitted changes (stable starting point)
- **Status CONFIRMED**: Problem Statement must be confirmed, not DRAFT
- **Codebase accessible**: the project root must be accessible (source directories exist)

Only proceed to Phase 1 when pre-flight returns PASS.

### Pre-flight: Document Sanity Check

After the script gate passes, do a quick scan of the Problem Statement itself:

1. **Required frontmatter fields** (YAML): `problem_id`, `problem_type`, `status` must be present.
2. **Required body sections** (Markdown headings): `## current_behavior`, `## desired_behavior`, `## rationale`, `## acceptance_criteria` must be present.

If fields or sections are missing, note them as open questions — do NOT block. The script gate is the hard gate; document issues are soft warnings.

### Phase 0: Repository Scan (mandatory)

Generate a repository map before doing any code exploration:

```bash
bash .agents/skills/solution-search/scripts/repo-scanner.sh --format json --output /tmp/repo-scan.json
```

This walks the repo tree, counts files by language, and extracts exports (Go: types/functions/methods, Rust: pub struct/enum/fn, TS/JS: export class/func/type, Python: class/func). The JSON output is your map — use it to navigate to relevant files instead of guessing.

### Phase 0b: Anti-patterns (mandatory)

Extract anti-patterns from the Problem Statement before Phase 2. These are constraints that narrow the search by ruling out obviously wrong paths:

- **`non_goals`**: explicit exclusions from the PS. If the PS says "no real database", don't propose PostgreSQL/MongoDB solutions.
- **`constraints`**: technical/business constraints that rule out approaches violating them.
- **`risks`**: known risks that should make you avoid similar patterns.

Anti-patterns are not goals — they're guardrails. They prevent wasted effort on paths that can't work.

### Phase 1: Read Problem Statement

Read the Problem Statement document in full. Do NOT summarize or extract — the document is already structured and verified on pre-flight. The entire content matters: `evidence` gives file:line entry points, `constraints` define hard boundaries, `non_goals` prevent wasted effort, `open_questions` need addressing, `risks` flag what to watch for.

**Pass the full document content as context for all subsequent phases.** The agent should have the complete PS available when exploring code, brainstorming solutions, and evaluating tradeoffs. Nothing in the document is optional.

### Phase 2: Code Exploration

Trace the actual code paths involved. Do NOT guess — read the code.

1. **Find entry points**: search for keywords from the problem (feature ID, key terms). Look at entry points, orchestration code, and public APIs.
2. **Trace data flow**: follow variables/structs through the system. Read the relevant files.
3. **Find existing patterns**: look for similar features already implemented. Follow the same patterns.
4. **Identify integration points**: where does new code need to hook in? What interfaces need changing?

For each file you read, note the file path and relevant line numbers. This information will be used in the solution proposals.

### Phase 3: Brainstorm Solutions

Propose 2-4 distinct approaches, ordered from simplest to most complex. For each:

1. **Name**: short, descriptive label (e.g., "Direct wiring", "Options passthrough", "Repository pattern")
2. **Description**: общее направление, какие файлы меняются, как данные текут
3. **Pros**: что делает этот подход хорошим
4. **Cons**: что делает его плохим или рискованным
5. **Effort**: rough estimate (e.g., "1-2 hours", "half day")

**Always propose at least one minimal solution first.** The simplest approach that works is usually the best.

### Phase 4: Evaluate Tradeoffs

For each approach, evaluate:

| Dimension | What to check |
|---|---|
| **Domain purity** | Does it introduce external deps in a dependency-free layer? (skip if project has no such layer) |
| **Breaking changes** | Does it change public APIs, config format, or existing behavior? |
| **Testability** | Can it be unit-tested without integration setup? (adapt to project's test conventions) |
| **Future extensibility** | Can more features be added on top without touching this code again? |
| **Risk** | What could go wrong? Which code paths are most fragile? |

### Phase 5: Decide Verdict

The verdict is NOT "pick the best approach" — it's a **go/no-go decision** for the problem. Three possible outcomes:

- **APPROVED**: The problem is solvable. Recommend the best approach with rationale. This is the default when at least one approach is viable.
- **REJECTED**: The problem cannot be solved given current constraints. Document the blocker(s) — architectural, technical, or business. The problem should be re-evaluated when conditions change.
- **DEFERRED**: The problem is theoretically solvable but not right now. Document what needs to happen first (e.g., "need to refactor X before this can work cleanly").

**The verdict answers: "Can this be done, and if so, how (at a high level)?"** It does NOT contain implementation steps — that's a future planning phase.

### Phase 6: Output

Write the Solution Verdict document to a project-appropriate location (e.g., `doc/<feature-id>_solution_options.md`).

```markdown
---
problem_id: <DOMAIN>-<NNN>
title: <short title matching the problem>
generated_at: <ISO 8601 timestamp>
verdict: APPROVED | REJECTED | DEFERRED
---

## approaches:

### Approach 1: <name>
**Description:** <общее направление, какие файлы, как данные текут>

**Pros:**
- <reason>

**Cons:**
- <reason>

**Effort:** <estimate>

### Approach 2: <name>
...

## verdict: <APPROVED | REJECTED | DEFERRED>

**Rationale:** <почему этот вердикт, сравнение подходов>

## open_questions:
- unresolved items (или "none")
```

**Document rules:**
- Use YAML literal block scalars (`|`) for multi-paragraph fields
- Always include at least 2 distinct approaches
- Be specific about file paths and line numbers from code exploration
- If an approach introduces external dependencies in a layer that should be dependency-free, flag it explicitly
- `open_questions` should address items that prevent a clean APPROVED verdict

### Post-flight: Landing Gate

After writing the document, validate it before presenting to the user:

```bash
bash .agents/skills/solution-search/scripts/landing-gate.sh doc/<feature-id>_solution_options.md
```

The landing gate checks:
- **HARD**: file exists, frontmatter fields (problem_id, title, generated_at, verdict), verdict ∈ {APPROVED, REJECTED, DEFERRED}, ≥ 2 approaches, approach structure (Description, Pros, Cons), verdict section with rationale
- **SOFT**: open_questions section present (warning only, non-blocking)

Only present the document to the user when landing gate returns PASS.

## Tips

- **Read before you write.** Never propose changes to code you haven't read. If the code path is unclear, search more or ask.
- **Respect the architecture.** Follow existing patterns for dependency wiring, interface definitions, and data flow. Match the project's conventions.
- **One feature, one document.** This skill produces a single artifact. If the user wants multiple solution documents, run this skill once per feature.
- **Be honest about gaps.** If you can't find the code path, note it as an open question rather than guessing.
