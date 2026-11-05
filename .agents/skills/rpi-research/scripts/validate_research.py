#!/usr/bin/env python3
"""Deterministic validation for rpi-research research.md artifacts.

Validates the deterministic quality gates defined in the rpi-research skill:
- No YAML frontmatter
- H1 title pattern: `[TICKET-ID] Research` or `Use Case Research`
- All headings are H2 under the H1
- FAR table present with numeric F, A, R scores and computed Mean (2 decimals)
- FAR thresholds: F>=4, A>=3, R>=3, Mean>=4.00
- Mean equals arithmetic average of F, A, R (2 decimal precision)
- Code references resolve to existing files with lines within bounds
- No code snippet exceeds ~40 lines unless justified
"""

import json
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent.parent / "rpi-common" / "scripts"))

from rpi_validate_common import (
    ValidationError,
    load_lines,
    check_no_frontmatter,
    check_ascii_only,
    extract_code_references,
    check_code_references_resolve,
    print_usage,
    main_usage,
    write_handoff_marker,
)

FAR_THRESHOLD_F = 4
FAR_THRESHOLD_A = 3
FAR_THRESHOLD_R = 3
FAR_THRESHOLD_MEAN = 4.00
MAX_SNIPPET_LINES = 40


def check_h1_title(lines):
    """H1 title must end with 'Research' (e.g. `[TICKET-ID] Research`, `Use Case Research`, or descriptive)."""
    h1_lines = [i for i, l in enumerate(lines) if re.match(r"^# ", l) and not l.startswith("## ")]
    if not h1_lines:
        raise ValidationError("No H1 title found; expected a title ending with 'Research'")
    h1_text = lines[h1_lines[0]][2:].strip()
    if not re.search(r"Research$", h1_text):
        raise ValidationError(f"H1 title '{h1_text}' must end with 'Research'")


# Expected H2 sections in order. required=True means the section must appear.
H2_SCHEMA = [
    ("Problem Context", True),
    ("Affected Files", True),
    ("Code Examples", True),
    ("External Research Findings", True),
    ("FAR Scale Output", True),
    ("Testing Strategy", True),
    ("Potential Design Pattern Recommendations", True),
    ("Assumptions", True),
    ("Out of Scope", True),
]


def check_headings_structure(lines):
    """Validate H2 section order/presence; allow H3 subsections under any H2."""
    h2_names = []
    under_h2 = None
    for i, l in enumerate(lines):
        m = re.match(r"^(#+)\s+(.+)$", l)
        if not m:
            continue
        level = len(m.group(1))
        heading = m.group(2).strip()
        if level == 1:
            under_h2 = None
            continue
        if level == 2:
            h2_names.append(heading)
            under_h2 = heading
            continue
        if level == 3:
            if under_h2 is None:
                raise ValidationError(f"H3 heading at line {i+1} without parent H2: '{heading}'")
            continue
        if level >= 4:
            raise ValidationError(f"Heading at line {i+1} has level {level}; max level 3 (H3) allowed under H2 sections")

    # Validate H2 schema: order and required presence
    expected_names = [name for name, _ in H2_SCHEMA]
    required_names = {name for name, req in H2_SCHEMA if req}

    if h2_names != expected_names:
        missing = required_names - set(h2_names)
        extra = set(h2_names) - set(expected_names)
        details = []
        if missing:
            details.append(f"missing required: {sorted(missing)}")
        if extra:
            details.append(f"unexpected: {sorted(extra)}")
        raise ValidationError(f"H2 sections invalid: {', '.join(details)} (expected order: {expected_names})")


def extract_far_scores(lines):
    """Extract F, A, R scores and Mean from the FAR Scale Output section."""
    text = "\n".join(lines)

    # Format 1: "### Factual Score: 4", "### Actionable Score: 4", etc.
    f_match = re.search(r"###\s*(?:Factual|F)\s*Score?:\s*(\d)", text)
    a_match = re.search(r"###\s*(?:Actionable|A)\s*Score?:\s*(\d)", text)
    r_match = re.search(r"###\s*(?:Relevant|R)\s*Score?:\s*(\d)", text)
    mean_match = re.search(r"###\s*Mean:\s*([0-9]+\.[0-9]{2})", text)

    # Format 2: "F: 4  A: 4  R: 4  Mean: 4.00" inline
    if not f_match:
        f_match = re.search(r"^F:\s*(\d)", text, re.MULTILINE)
    if not a_match:
        a_match = re.search(r"^A:\s*(\d)", text, re.MULTILINE)
    if not r_match:
        r_match = re.search(r"^R:\s*(\d)", text, re.MULTILINE)
    if not mean_match:
        mean_match = re.search(r"Mean:\s*([0-9]+\.[0-9]{2})", text)

    if not all([f_match, a_match, r_match, mean_match]):
        raise ValidationError("FAR table missing numeric F, A, R scores or Mean (2 decimals)")

    f = int(f_match.group(1))
    a = int(a_match.group(1))
    r = int(r_match.group(1))
    mean = float(mean_match.group(1))
    return f, a, r, mean


def check_far_thresholds(f, a, r, mean):
    """FAR thresholds: F>=4, A>=3, R>=3, Mean>=4.00."""
    if f < FAR_THRESHOLD_F:
        raise ValidationError(f"Factual score {f} < threshold {FAR_THRESHOLD_F}")
    if a < FAR_THRESHOLD_A:
        raise ValidationError(f"Actionable score {a} < threshold {FAR_THRESHOLD_A}")
    if r < FAR_THRESHOLD_R:
        raise ValidationError(f"Relevant score {r} < threshold {FAR_THRESHOLD_R}")
    if mean < FAR_THRESHOLD_MEAN:
        raise ValidationError(f"Mean score {mean} < threshold {FAR_THRESHOLD_MEAN}")


def check_mean_calculation(f, a, r, mean):
    """Mean must equal arithmetic average of F, A, R to 2 decimals."""
    expected = round((f + a + r) / 3.0, 2)
    if abs(mean - expected) > 0.001:
        raise ValidationError(f"Mean {mean} != computed average {expected} of F={f}, A={a}, R={r}")


def check_snippet_lengths(lines):
    """No code snippet exceeds ~40 lines unless justified by a '**Snippet commentary:**' line after."""
    in_code_block = False
    block_lines = 0
    block_end_line = 0
    for idx, l in enumerate(lines):
        line_no = idx + 1
        if l.strip().startswith("```"):
            if in_code_block:
                if block_lines > MAX_SNIPPET_LINES:
                    # Check for justification comment after the closing ``` (skip blank lines)
                    justified = False
                    for cand in range(block_end_line + 1, min(block_end_line + 4, len(lines))):
                        if lines[cand].strip().startswith("**Snippet commentary:**"):
                            justified = True
                            break
                    if not justified:
                        raise ValidationError(f"Code snippet at line {block_end_line - block_lines + 1} exceeds {MAX_SNIPPET_LINES} lines (found {block_lines}) without '**Snippet commentary:**' justification")
                block_lines = 0
            in_code_block = not in_code_block
            continue
        if in_code_block:
            block_lines += 1
            block_end_line = line_no


def validate_research_md(research_path: Path):
    """Run all deterministic checks on a research.md file."""
    if not research_path.exists():
        raise ValidationError(f"research.md not found: {research_path}")

    lines = load_lines(research_path)

    check_no_frontmatter(lines, "research")
    check_h1_title(lines)
    check_headings_structure(lines)
    check_snippet_lengths(lines)
    check_ascii_only(lines)

    f, a, r, mean = extract_far_scores(lines)
    check_far_thresholds(f, a, r, mean)
    check_mean_calculation(f, a, r, mean)

    refs = extract_code_references(lines)
    # research.md lives at rpi/[problem]/research.md; repo root is two levels up
    check_code_references_resolve(refs, research_path.parent.parent.parent, "research")

    return True


def main():
    json_output = False
    if len(sys.argv) >= 2 and sys.argv[1] in ("--help", "-h"):
        print_usage("validate_research.py", "path-to-research.md")
        sys.exit(0)
    if len(sys.argv) >= 2 and sys.argv[1] == "--json":
        json_output = True
        sys.argv = sys.argv[:1] + sys.argv[2:]
    if len(sys.argv) < 2:
        main_usage("validate_research.py", "path-to-research.md")
        sys.exit(2)

    research_path = Path(sys.argv[1])
    result = {"file": str(research_path), "status": "valid", "errors": []}
    try:
        validate_research_md(research_path)
    except ValidationError as e:
        result["status"] = "invalid"
        result["errors"].append(str(e))
    except Exception as e:
        result["status"] = "error"
        result["errors"].append(str(e))

    if result["status"] == "valid":
        repo_root = research_path.parent.parent.parent
        lines = load_lines(research_path)
        f, a, r, mean = extract_far_scores(lines)
        write_handoff_marker(repo_root / ".rpi-workflow.json", "research", research_path.parent.name, "research.md", facts_mean=mean)

    if json_output:
        print(json.dumps(result, indent=2))
        sys.exit(0 if result["status"] == "valid" else 1)

    if result["status"] == "valid":
        print("VALID: research.md passes all deterministic checks")
        sys.exit(0)
    if result["status"] == "invalid":
        for err in result["errors"]:
            print(f"INVALID: {err}", file=sys.stderr)
        sys.exit(1)
    for err in result["errors"]:
        print(f"ERROR: {err}", file=sys.stderr)
    sys.exit(3)


if __name__ == "__main__":
    main()
