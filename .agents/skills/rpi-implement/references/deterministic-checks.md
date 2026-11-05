# Deterministic Checks for rpi-implement

These checks are programmatically verifiable and can be enforced via the `scripts/validate_implement.py` script. They complement the qualitative quality gates and ensure the implementation artifact is structurally sound.

## Enforced Deterministic Rules

### 1. No YAML Frontmatter
- The `implementation.md` must not begin with a `---` frontmatter block.

### 2. H1 Title Pattern
- H1 title must end with "Implementation" (e.g. `[Problem Statement] Implementation`).

### 3. Heading Hierarchy
- All headings under the H1 must be H2 (exactly `## `) or H3 subsections (e.g. ADR entries, "### Phase N:") under their parent H2 section. No H4+ headings.

### 4. H2 Section Order
The following 6 sections must appear in exact order:
1. Summary
2. Unexpected Discoveries
3. Plan Deviations
4. Architecture Decision Records
5. Handoff Notes
6. Reference

### 5. Architecture Decision Record (ADR) Strict Format
Each ADR (H3 subsection under "Architecture Decision Records") must follow this exact parseable structure:

```markdown
### {ADR name}

context:
{single-line description}

options:
- {option 1}
- {option 2}

decision:
{single-line description}

rationale:
{single-line description}

consequences:
- {consequence 1}
- {consequence 2}
```

Rules:
- Field headers (`context:`, `options:`, `decision:`, `rationale:`, `consequences:`) must be on their own line, never combined with content
- Fields must appear in exact order: context, options, decision, rationale, consequences
- `options` and `consequences` must contain bullet points (`- `)
- `context`, `decision`, and `rationale` are single-line descriptions

### 6. Plan Reference Resolution
- Any reference to `plan.md` in the document must resolve to an existing, valid plan.md file (with an H1 title ending in "Plan").

### 7. ASCII-Only Text
- Document text must contain ONLY ASCII characters (no non-English text, en-dashes, em-dashes, or other non-ASCII characters).
- This check ignores content inside fenced code blocks (```) and inline code spans (`).

### 8. No Blank Lines Between Consecutive Bullets
- A blank line between two consecutive bullet points (lines N-1 and N+1 are bullets, line N is blank) must be removed. Blank lines between list items create separate markdown list items, which is visually inconsistent.

### 9. Handoff Notes Subsections
The "Handoff Notes" section must contain exactly these 3 H3 subsections:
1. **Known Limitations** — what doesn't work, what's incomplete, what to avoid
2. **Data Models** — key types/structs with fields and purpose
3. **API Surface** — HTTP endpoints with method, path, and description

### 10. Reference Section
- A "Reference" H2 section must exist as the last section.
- Contains a bulleted list of related files (research.md, plan.md, other artifacts).

### 11. Pull Requests Table
- A "Pull Requests" H2 section must exist (after Summary, before Unexpected Discoveries).
- Contains a markdown table with columns: PR, Description, Status.
- Each row must have a GitHub PR link in the format `[#NNN](url)`.

## Verification

Run the deterministic validator:

```bash
python3 .agents/skills/rpi-implement/scripts/validate_implement.py rpi/[problem-short-name]/implementation.md
```

Expected output on success:
```
VALID: implementation.md passes all deterministic checks
```

On failure, the script prints the specific violation and exits with a non-zero status code.

## Pre-Execution Branch Check

Before starting implementation, verify the current git branch matches `implement/[problem-short-name]`:

```bash
python3 .agents/skills/rpi-implement/scripts/check_branch.py [problem-short-name]
```

Expected output on success:
```
OK: on branch 'implement/[problem-short-name]'
```

On failure (wrong branch or not in a git repo), the script prints a BRANCH CHECK FAILED message and exits with status 1 or 3. This deterministic check prevents implementation work from occurring on the main branch or an incorrectly named branch.
