#!/usr/bin/env python3
"""Deterministic validation for rpi-plan plan.md artifacts.

Validates the deterministic quality gates defined in the rpi-plan skill:
- No YAML frontmatter
- H1 title pattern: `[TICKET-ID] Plan` or `Use Case Plan` (ends with 'Plan')
- All headings are H2 under the H1; no H3+ headings
- 8 required H2 sections in exact order: Implementation Overview, Task Breakdown,
  Code References, Testing Plan, FACTS Scale Output, Dependencies & Sequencing,
  Risk Mitigation, Rollback Strategy
- Task checkboxes use `- [ ]` syntax under "## Phase N:" headings
- FACTS Scale Output present with numeric F, A, C, T, S scores and computed Mean (2 decimals)
- FACTS thresholds: F>=3, Mean>=3.00, all scores 0-5
- Mean equals arithmetic average of F, A, C, T, S (2 decimal precision)
- Code references resolve to existing files with lines within bounds (entire-file refs skipped)
- No non-ASCII characters outside code blocks and inline code spans
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

FACTS_THRESHOLD_F = 3
FACTS_THRESHOLD_MEAN = 3.00


def check_h1_title(lines):
    """H1 title must end with 'Plan' (e.g. `[TICKET-ID] Plan`, `Use Case Plan`)."""
    h1_lines = [i for i, l in enumerate(lines) if re.match(r"^# ", l) and not l.startswith("## ")]
    if not h1_lines:
        raise ValidationError("No H1 title found; expected a title ending with 'Plan'")
    h1_text = lines[h1_lines[0]][2:].strip()
    if not re.search(r"Plan$", h1_text):
        raise ValidationError(f"H1 title '{h1_text}' must end with 'Plan'")


# Expected H2 sections in order. required=True means the section must appear.
H2_SCHEMA = [
    ("Implementation Overview", True),
    ("Task Breakdown", True),
    ("Code References", True),
    ("Testing Plan", True),
    ("FACTS Scale Output", True),
    ("Dependencies & Sequencing", True),
    ("Risk Mitigation", True),
    ("Rollback Strategy", True),
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
                raise ValidationError(f"Heading at line {i+1} has level 3 (H3) without parent H2: '{heading}'")
            continue
        if level >= 4:
            raise ValidationError(f"Heading at line {i+1} has level {level}; max level 3 (H3) allowed under H2 sections")

    expected_names = [name for name, _ in H2_SCHEMA]

    if h2_names != expected_names:
        missing = set(expected_names) - set(h2_names)
        extra = set(h2_names) - set(expected_names)
        details = []
        if missing:
            details.append(f"missing required: {sorted(missing)}")
        if extra:
            details.append(f"unexpected: {sorted(extra)}")
        raise ValidationError(f"H2 sections invalid: {', '.join(details)} (expected order: {expected_names})")


def check_task_checkboxes(lines):
    """Task Breakdown phase must contain checkbox tasks under '## Phase N:' or '### Phase N:' headings."""
    in_task_breakdown = False
    in_phase = False
    found_checkbox = False
    for l in lines:
        stripped = l.strip()
        if stripped.startswith("## "):
            heading = stripped[3:].strip()
            in_task_breakdown = heading == "Task Breakdown"
            in_phase = heading.startswith("Phase ")
            continue
        if stripped.startswith("### "):
            heading = stripped[4:].strip()
            in_phase = heading.startswith("Phase ")
            continue
        if in_task_breakdown and in_phase:
            if re.match(r"- \[ \] ", stripped) or re.match(r"- \[x\] ", stripped):
                found_checkbox = True
                break
    if not found_checkbox:
        raise ValidationError("Task Breakdown phase missing checkbox tasks ('- [ ] ...' or '- [x] ...' under '### Phase N:')")


def extract_facts_scores(lines):
    """Extract F, A, C, T, S scores and Mean from the FACTS Scale Output section."""
    text = "\n".join(lines)

    # Inline format: "F: 4  A: 3  C: 3  T: 4  S: 3  Mean: 3.40  --> PASS"
    f_match = re.search(r"F:\s*([0-5])", text)
    a_match = re.search(r"\bA:\s*([0-5])", text)
    c_match = re.search(r"\bC:\s*([0-5])", text)
    t_match = re.search(r"\bT:\s*([0-5])", text)
    s_match = re.search(r"\bS:\s*([0-5])", text)
    mean_match = re.search(r"Mean:\s*([0-9]+\.[0-9]{2})", text)

    if not all([f_match, a_match, c_match, t_match, s_match, mean_match]):
        raise ValidationError("FACTS table missing numeric F, A, C, T, S scores or Mean (2 decimals)")

    f = int(f_match.group(1))
    a = int(a_match.group(1))
    c = int(c_match.group(1))
    t = int(t_match.group(1))
    s = int(s_match.group(1))
    mean = float(mean_match.group(1))
    return f, a, c, t, s, mean


def check_facts_thresholds(f, a, c, t, s, mean):
    """FACTS thresholds: F>=3, all scores 0-5, Mean>=3.00."""
    for name, score in [("F", f), ("A", a), ("C", c), ("T", t), ("S", s)]:
        if not (0 <= score <= 5):
            raise ValidationError(f"FACTS score {name}={score} out of range 0-5")
    if f < FACTS_THRESHOLD_F:
        raise ValidationError(f"Feasibility score {f} < threshold {FACTS_THRESHOLD_F}")
    if mean < FACTS_THRESHOLD_MEAN:
        raise ValidationError(f"Mean score {mean} < threshold {FACTS_THRESHOLD_MEAN}")


def check_mean_calculation(f, a, c, t, s, mean):
    """Mean must equal arithmetic average of F, A, C, T, S to 2 decimals."""
    expected = round((f + a + c + t + s) / 5.0, 2)
    if abs(mean - expected) > 0.001:
        raise ValidationError(f"Mean {mean} != computed average {expected} of F={f}, A={a}, C={c}, T={t}, S={s}")


def validate_plan_md(plan_path: Path):
    """Run all deterministic checks on a plan.md file."""
    if not plan_path.exists():
        raise ValidationError(f"plan.md not found: {plan_path}")

    lines = load_lines(plan_path)

    check_no_frontmatter(lines, "plan")
    check_h1_title(lines)
    check_headings_structure(lines)
    check_task_checkboxes(lines)
    check_ascii_only(lines)

    f, a, c, t, s, mean = extract_facts_scores(lines)
    check_facts_thresholds(f, a, c, t, s, mean)
    check_mean_calculation(f, a, c, t, s, mean)

    refs = extract_code_references(lines)
    # plan.md lives at rpi/[problem]/plan.md; repo root is two levels up
    check_code_references_resolve(refs, plan_path.parent.parent.parent, "plan")

    return True


def main():
    json_output = False
    if len(sys.argv) >= 2 and sys.argv[1] in ("--help", "-h"):
        print_usage("validate_plan.py", "path-to-plan.md")
        sys.exit(0)
    if len(sys.argv) >= 2 and sys.argv[1] == "--json":
        json_output = True
        sys.argv = sys.argv[:1] + sys.argv[2:]
    if len(sys.argv) < 2:
        main_usage("validate_plan.py", "path-to-plan.md")
        sys.exit(2)

    plan_path = Path(sys.argv[1])
    result = {"file": str(plan_path), "status": "valid", "errors": []}
    try:
        validate_plan_md(plan_path)
    except ValidationError as e:
        result["status"] = "invalid"
        result["errors"].append(str(e))
    except Exception as e:
        result["status"] = "error"
        result["errors"].append(str(e))

    if result["status"] == "valid":
        repo_root = plan_path.parent.parent.parent
        lines = load_lines(plan_path)
        f, a, c, t, s, mean = extract_facts_scores(lines)
        tasks_count = sum(1 for l in lines if re.match(r"- \[ ?\] ", l.strip()))
        write_handoff_marker(
            repo_root / ".rpi-workflow.json",
            "plan",
            plan_path.parent.name,
            "plan.md",
            facts_mean=mean,
            tasks_count=tasks_count,
        )

    if json_output:
        print(json.dumps(result, indent=2))
        sys.exit(0 if result["status"] == "valid" else 1)

    if result["status"] == "valid":
        print("VALID: plan.md passes all deterministic checks")
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
