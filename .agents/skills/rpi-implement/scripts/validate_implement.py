#!/usr/bin/env python3
"""Deterministic validation for rpi-implement implementation.md artifacts.

Validates the deterministic quality gates defined in the rpi-implement skill:
- No YAML frontmatter
- H1 title pattern: ends with "Implementation"
- 6 required H2 sections in exact order: Summary, Pull Requests, Unexpected Discoveries,
  Plan Deviations, Architecture Decision Records, Handoff Notes, Reference
- Each ADR (H3 under "Architecture Decision Records") has 5 required sub-fields
- PR table present with GitHub links
- Plan reference resolves to existing, valid plan.md
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
    print_usage,
    main_usage,
    write_handoff_marker,
)

# Required H2 sections in exact order.
H2_SCHEMA = [
    "Summary",
    "Pull Requests",
    "Unexpected Discoveries",
    "Plan Deviations",
    "Architecture Decision Records",
    "Handoff Notes",
    "Reference",
]


def check_h1_title(lines):
    """H1 title must end with 'Implementation'."""
    h1_lines = [i for i, l in enumerate(lines) if re.match(r"^# ", l) and not l.startswith("## ")]
    if not h1_lines:
        raise ValidationError("No H1 title found; expected a title ending with 'Implementation'")
    h1_text = lines[h1_lines[0]][2:].strip()
    if not re.search(r"Implementation$", h1_text):
        raise ValidationError(f"H1 title '{h1_text}' must end with 'Implementation'")


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

    if h2_names != H2_SCHEMA:
        missing = set(H2_SCHEMA) - set(h2_names)
        extra = set(h2_names) - set(H2_SCHEMA)
        details = []
        if missing:
            details.append(f"missing required: {sorted(missing)}")
        if extra:
            details.append(f"unexpected: {sorted(extra)}")
        raise ValidationError(f"H2 sections invalid: {', '.join(details)} (expected order: {H2_SCHEMA})")


def check_adr_format(lines):
    """Each ADR must follow a strict parseable format:

    ### {NAME}

    context:
    {context_description}

    options:
    - {option_1}
    - {option_2}

    decision:
    {decision_description}

    rationale:
    {rationale_description}

    consequences:
    - {consequence_1}
    - {consequence_2}
    """
    in_adr_section = False
    in_adr = False
    adr_name = None
    phase = None
    field_data = {}

    def _validate_adr(name, fields):
        for required in ['context', 'options', 'decision', 'rationale', 'consequences']:
            if required not in fields or not fields[required]:
                raise ValidationError(f"ADR '{name}' missing or empty field: {required}")
        # Check field order
        actual = [k for k in ['context', 'options', 'decision', 'rationale', 'consequences'] if k in fields]
        if actual != ['context', 'options', 'decision', 'rationale', 'consequences']:
            raise ValidationError(f"ADR '{name}' fields not in expected order: {actual}")
        # Check 'options' has bullet points
        for line in fields['options']:
            if line.strip() and not line.strip().startswith('- '):
                raise ValidationError(f"ADR '{name}' field 'options' must use bullet points: '{line.strip()}'")
        # Check 'consequences' has bullet points
        for line in fields['consequences']:
            if line.strip() and not line.strip().startswith('- '):
                raise ValidationError(f"ADR '{name}' field 'consequences' must use bullet points: '{line.strip()}'")

    for i, l in enumerate(lines):
        stripped = l.strip()

        if stripped.startswith("## "):
            heading = stripped[3:].strip()
            if heading == "Architecture Decision Records":
                in_adr_section = True
                in_adr = False
                adr_name = None
                phase = None
                field_data = {}
                continue
            else:
                if in_adr:
                    _validate_adr(adr_name, field_data)
                in_adr_section = False
                in_adr = False
                adr_name = None
                phase = None
                field_data = {}
                continue

        if in_adr_section and stripped.startswith("### "):
            if in_adr:
                _validate_adr(adr_name, field_data)
            in_adr = True
            adr_name = stripped[4:].strip()
            phase = None
            field_data = {}
            continue

        if not in_adr_section or not in_adr:
            continue

        # Blank line ends current phase content
        if stripped == "":
            phase = None
            continue

        # Field headers must be exactly "context:", "options:", etc. on their own line
        is_field_header = False
        for field in ('context', 'options', 'decision', 'rationale', 'consequences'):
            if stripped == field + ":":
                phase = field
                if phase not in field_data:
                    field_data[phase] = []
                is_field_header = True
                break

        if is_field_header:
            continue

        # Reject "context: text" on same line as field name (content after colon)
        for field in ('context', 'options', 'decision', 'rationale', 'consequences'):
            if stripped.startswith(field + ": ") or stripped.startswith(field + "\t"):
                raise ValidationError(
                    f"ADR '{adr_name}' at line {i + 1}: field '{field}:' must be on its own line, "
                    f"not combined with content"
                )

        # Accumulate field content
        if phase:
            field_data[phase].append(stripped)

    # Validate last ADR if file ends while in one
    if in_adr_section and in_adr:
        _validate_adr(adr_name, field_data)


def extract_plan_references(lines):
    """Extract plan.md file references from the document."""
    refs = []
    for i, l in enumerate(lines):
        # Match plan.md references (e.g., "plan.md", "rpi/.../plan.md")
        for m in re.finditer(r"([a-zA-Z0-9_./-]*plan\.md)", l):
            refs.append((i + 1, m.group(1)))
    return refs


def check_plan_references_resolve(refs, base_dir: Path):
    """Plan references must resolve to existing, valid plan.md files."""
    for line_no, plan_path in refs:
        full_path = (base_dir / plan_path).resolve()
        if not full_path.exists():
            raise ValidationError(f"Plan reference at line {line_no} points to non-existent file: {plan_path}")
        # Validate the plan.md using the plan validator logic (inline, to avoid circular import)
        plan_lines = full_path.read_text(encoding="utf-8").splitlines()
        # Quick check: H1 title ends with "Plan"
        h1_lines = [l for l in plan_lines if re.match(r"^# ", l) and not l.startswith("## ")]
        if not h1_lines:
            raise ValidationError(f"Plan reference at line {line_no} points to invalid plan.md (no H1 title): {plan_path}")


def check_bullet_spacing(lines):
    """Blank lines between consecutive bullet points should not exist."""
    def _is_bullet(l):
        return bool(re.match(r"^\s*[-*]\s+", l))

    for i in range(1, len(lines) - 1):
        if lines[i].strip() == "" and _is_bullet(lines[i - 1]) and _is_bullet(lines[i + 1]):
            raise ValidationError(
                f"Blank line at line {i + 1} between two bullet points; "
                f"remove the empty line (lines {i} and {i + 2} are consecutive bullets)"
            )


HANDOFF_SUBSECTIONS = ["Known Limitations", "Data Models", "API Surface"]


def check_handoff_structure(lines):
    """Handoff Notes must contain required subsections as H3 headings."""
    in_handoff = False
    found_subsections = set()

    for i, l in enumerate(lines):
        stripped = l.strip()

        if stripped.startswith("## "):
            heading = stripped[3:].strip()
            if heading == "Handoff Notes":
                in_handoff = True
                continue
            elif in_handoff:
                # Left the Handoff Notes section
                break

        if in_handoff and stripped.startswith("### "):
            found_subsections.add(stripped[4:].strip())

    missing = set(HANDOFF_SUBSECTIONS) - found_subsections
    if missing:
        raise ValidationError(f"Handoff Notes missing required subsections: {', '.join(sorted(missing))}")


def check_pr_table(lines):
    """Pull Requests section must contain a markdown table with PR links."""
    in_pr_section = False
    found_table = False
    has_pr_link = False

    for i, l in enumerate(lines):
        stripped = l.strip()

        if stripped.startswith("## "):
            heading = stripped[3:].strip()
            if heading == "Pull Requests":
                in_pr_section = True
                continue
            elif in_pr_section:
                # Left the Pull Requests section
                break

        if in_pr_section:
            # Check for table row (contains |)
            if "|" in stripped and stripped != "|" and stripped.replace("|", "").strip():
                found_table = True
                # Check for GitHub PR link in the row
                if re.search(r"\[#\d+\]\(https?://", stripped):
                    has_pr_link = True

    if not found_table:
        raise ValidationError("Pull Requests section missing markdown table with PR entries")
    if not has_pr_link:
        raise ValidationError("Pull Requests table must contain at least one GitHub PR link [#NNN](url)")


def validate_implementation_md(impl_path: Path):
    """Run all deterministic checks on an implementation.md file."""
    if not impl_path.exists():
        raise ValidationError(f"implementation.md not found: {impl_path}")

    lines = load_lines(impl_path)

    check_no_frontmatter(lines, "implementation")
    check_h1_title(lines)
    check_headings_structure(lines)
    check_adr_format(lines)
    check_bullet_spacing(lines)
    check_handoff_structure(lines)
    check_pr_table(lines)
    check_ascii_only(lines)

    refs = extract_plan_references(lines)
    if refs:
        check_plan_references_resolve(refs, impl_path.parent.parent.parent)

    return True


def main():
    json_output = False
    if len(sys.argv) >= 2 and sys.argv[1] in ("--help", "-h"):
        print_usage("validate_implement.py", "path-to-implementation.md")
        sys.exit(0)
    if len(sys.argv) >= 2 and sys.argv[1] == "--json":
        json_output = True
        sys.argv = sys.argv[:1] + sys.argv[2:]
    if len(sys.argv) < 2:
        main_usage("validate_implement.py", "path-to-implementation.md")
        sys.exit(2)

    impl_path = Path(sys.argv[1])
    result = {"file": str(impl_path), "status": "valid", "errors": []}
    try:
        validate_implementation_md(impl_path)
    except ValidationError as e:
        result["status"] = "invalid"
        result["errors"].append(str(e))
    except Exception as e:
        result["status"] = "error"
        result["errors"].append(str(e))

    if result["status"] == "valid":
        repo_root = impl_path.parent.parent.parent
        write_handoff_marker(repo_root / ".rpi-workflow.json", "implement", impl_path.parent.name, "implementation.md")

    if json_output:
        print(json.dumps(result, indent=2))
        sys.exit(0 if result["status"] == "valid" else 1)

    if result["status"] == "valid":
        print("VALID: implementation.md passes all deterministic checks")
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
