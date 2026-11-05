# Deterministic Checks for rpi-research

These checks are programmatically verifiable and can be enforced via the `scripts/validate_research.py` script. They complement the qualitative quality gates and FAR Scale scoring.

## Enforced Deterministic Rules

### 1. No YAML Frontmatter
- The research.md must not begin with a `---` frontmatter block.

### 2. H1 Title Pattern
- H1 title must match `[TICKET-ID] Research` (e.g. `# ISSUE-123 Research`) or `# Use Case Research`.

### 3. Heading Hierarchy
- All headings under the H1 must be H2 (exactly `## `). No H3+ headings in the main structure.

### 4. FAR Table Presence and Format
- A "FAR Scale Output" section must exist with numeric F, A, R scores (0-5 each) and a computed Mean to exactly 2 decimal places, plus a PASS/FAIL indicator.

### 5. FAR Thresholds
- Factual (F) >= 4
- Actionable (A) >= 3
- Relevant (R) >= 3
- Mean >= 4.00

### 6. Mean Calculation Correctness
- Mean must equal the arithmetic average of F, A, R rounded to 2 decimal places.

### 7. Code Reference Resolution
- Every code reference (`path/to/file.ext:LINE`, `path:START-END`, or entire file) must resolve to an existing file within the repo, with line numbers within bounds.

### 8. Snippet Length
- No code snippet exceeds 40 lines unless explicitly justified.

### 9. ASCII-Only Text
- Document text must contain ONLY ASCII characters (no non-English text, en-dashes, em-dashes, or other non-ASCII characters).
- This check ignores content inside fenced code blocks (```) and inline code spans (`).

## Verification

Run the deterministic validator:

```bash
python3 .agents/skills/rpi-research/scripts/validate_research.py rpi/[problem-short-name]/research.md
```

Expected output on success:
```
VALID: research.md passes all deterministic checks
```

On failure, the script prints the specific violation and exits with a non-zero status code.
