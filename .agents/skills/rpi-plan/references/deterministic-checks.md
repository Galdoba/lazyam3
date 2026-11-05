# Deterministic Checks for rpi-plan

These checks are programmatically verifiable and can be enforced via the `scripts/validate_plan.py` script. They complement the qualitative quality gates and FACTS Scale scoring.

## Enforced Deterministic Rules

### 1. No YAML Frontmatter
- The plan.md must not begin with a `---` frontmatter block.

### 2. H1 Title Pattern
- H1 title must end with "Plan" (e.g. `[TICKET-ID] Plan` or `Use Case Plan`).

### 3. Heading Hierarchy
- All headings under the H1 must be H2 (exactly `## `) or H3 subsections (e.g. "### Phase N:", "### Unit tests", "### Dependency chain") under their parent H2 section. No H4+ headings.

### 4. H2 Section Order
The following 8 sections must appear in exact order:
1. Implementation Overview
2. Task Breakdown
3. Code References
4. Testing Plan
5. FACTS Scale Output
6. Dependencies & Sequencing
7. Risk Mitigation
8. Rollback Strategy

### 5. Task Checkboxes
- The Task Breakdown phase (under "## Phase N:" headings) must contain checkbox tasks formatted as `- [ ] task description`.

### 6. FACTS Table Presence and Format
- A "FACTS Scale Output" section must exist with numeric F, A, C, T, S scores (0-5 each) and a computed Mean to exactly 2 decimal places, plus a PASS/FAIL indicator.

### 7. FACTS Thresholds
- Feasibility (F) >= 3
- Mean >= 3.00
- All scores (F, A, C, T, S) within range 0-5

### 8. Mean Calculation Correctness
- Mean must equal the arithmetic average of F, A, C, T, S rounded to 2 decimal places.

### 9. Code Reference Resolution
- Every code reference (`path/to/file.ext:LINE`, `path:START-END`) must resolve to an existing file within the repo, with line numbers within bounds.
- Entire-file references (no line number) are skipped (may be `[NEW]` files).

### 10. ASCII-Only Text
- Document text must contain ONLY ASCII characters (no non-English text, en-dashes, em-dashes, or other non-ASCII characters).
- This check ignores content inside fenced code blocks (```) and inline code spans (`).

## Verification

Run the deterministic validator:

```bash
python3 .agents/skills/rpi-plan/scripts/validate_plan.py rpi/[problem-short-name]/plan.md
```

Expected output on success:
```
VALID: plan.md passes all deterministic checks
```

On failure, the script prints the specific violation and exits with a non-zero status code.
