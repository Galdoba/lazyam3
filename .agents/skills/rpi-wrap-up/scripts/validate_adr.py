#!/usr/bin/env python3
"""Validate ADR files against the rpi-wrap-up format specification.

Usage:
  validate_adr.py <adr-file>              # Validate single ADR
  validate_adr.py --all <adr-dir>         # Validate all ADRs in directory + cross-references
"""

import argparse
import re
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("ERROR: PyYAML required. Install with: pip install pyyaml", file=sys.stderr)
    sys.exit(1)

# ── Required frontmatter fields ──────────────────────────────────────
REQUIRED_YAML_FIELDS = {"id", "title", "description", "status", "date", "scope"}
VALID_STATUSES = {"accepted", "deprecated", "superseded"}
VALID_SCOPES = {"cross-package", "cross-system", "architectural"}
ADR_SCOPES = {"cross-package", "cross-system", "architectural"}  # scope threshold for ADRs

# ── Required markdown sections ───────────────────────────────────────
REQUIRED_SECTIONS = ["Context", "Decision", "Rationale", "Consequences"]
OPTIONAL_SECTIONS = ["Options", "Related"]
ALL_SECTIONS = REQUIRED_SECTIONS + OPTIONAL_SECTIONS

# ── Regex patterns ───────────────────────────────────────────────────
ID_PATTERN = re.compile(r"^\d{4}-[a-z0-9]+(-[a-z0-9]+)*$")
DATE_PATTERN = re.compile(r"^\d{4}-\d{2}-\d{2}$")
H1_PATTERN = re.compile(r"^# (.+)$", re.MULTILINE)
H2_PATTERN = re.compile(r"^## (.+)$", re.MULTILINE)
CODE_BLOCK_PATTERN = re.compile(r"```[\s\S]*?```", re.MULTILINE)


def parse_adr(filepath: Path) -> dict:
    """Parse an ADR file into frontmatter YAML and markdown body."""
    content = filepath.read_text(encoding="utf-8")
    lines = content.split("\n")

    # Extract frontmatter (between first --- markers)
    if not lines or lines[0].strip() != "---":
        return {"error": "Missing YAML frontmatter (no opening ---)"}

    fm_end = None
    for i, line in enumerate(lines[1:], 1):
        if line.strip() == "---":
            fm_end = i
            break

    if fm_end is None:
        return {"error": "Missing closing --- for YAML frontmatter"}

    yaml_text = "\n".join(lines[1:fm_end])
    body_text = "\n".join(lines[fm_end + 1:]).strip()

    try:
        frontmatter = yaml.safe_load(yaml_text) or {}
    except yaml.YAMLError as e:
        return {"error": f"YAML parse error: {e}"}

    if not isinstance(frontmatter, dict):
        return {"error": "YAML frontmatter must be a mapping"}

    return {"frontmatter": frontmatter, "body": body_text}


def check_ascii_only(text: str) -> list[str]:
    """Check that text contains only ASCII characters (outside code blocks)."""
    non_ascii = []
    for i, line in enumerate(text.split("\n"), 1):
        stripped = line.strip()
        if stripped and any(ord(c) > 127 for c in stripped):
            non_ascii.append(f"Line {i}: non-ASCII characters found")
    return non_ascii


def validate_frontmatter(fm: dict) -> list[str]:
    """Validate YAML frontmatter fields."""
    errors = []

    # Check required fields
    for field in REQUIRED_YAML_FIELDS:
        if field not in fm:
            errors.append(f"Missing required frontmatter field: {field}")

    if errors:
        return errors

    # Validate id
    fm_id = str(fm.get("id", ""))
    if not ID_PATTERN.match(fm_id):
        errors.append(f"Invalid id format '{fm_id}': must match NNNN-title-slug (4 digits, lowercase alphanumeric + hyphens)")

    # Validate status
    status = str(fm.get("status", ""))
    if status not in VALID_STATUSES:
        errors.append(f"Invalid status '{status}': must be one of {sorted(VALID_STATUSES)}")

    # Validate date
    date_val = str(fm.get("date", ""))
    if not DATE_PATTERN.match(date_val):
        errors.append(f"Invalid date format '{date_val}': must be YYYY-MM-DD")

    # Validate scope
    scope = str(fm.get("scope", ""))
    if scope not in VALID_SCOPES:
        errors.append(f"Invalid scope '{scope}': must be one of {sorted(VALID_SCOPES)}")

    # Validate description is 1-2 sentences
    desc = str(fm.get("description", ""))
    if not desc.strip():
        errors.append("Description must not be empty")
    else:
        # Count sentence boundaries: period followed by space + uppercase letter or number
        sentence_count = len(re.findall(r"\.\s+[A-Z0-9]", desc))
        if sentence_count > 2:
            errors.append(f"Description should be 1-2 sentences, got {sentence_count}")

    return errors


def validate_markdown(body: str) -> list[str]:
    """Validate markdown body structure."""
    errors = []

    # Check H1
    h1_match = H1_PATTERN.search(body)
    if not h1_match:
        errors.append("Missing H1 heading (title)")
    else:
        h1_text = h1_match.group(1)
        if not h1_text.strip():
            errors.append("H1 heading is empty")

    # Check H2 sections
    h2_matches = H2_PATTERN.findall(body)
    if not h2_matches:
        errors.append("No H2 sections found")
        return errors

    # Check required sections present
    for section in REQUIRED_SECTIONS:
        if section not in h2_matches:
            errors.append(f"Missing required section: ## {section}")

    # Check section order
    section_indices = []
    for section in ALL_SECTIONS:
        if section in h2_matches:
            section_indices.append(h2_matches.index(section))

    # Verify required sections are in correct order
    for i in range(len(REQUIRED_SECTIONS) - 1):
        if REQUIRED_SECTIONS[i] in h2_matches and REQUIRED_SECTIONS[i + 1] in h2_matches:
            idx_a = h2_matches.index(REQUIRED_SECTIONS[i])
            idx_b = h2_matches.index(REQUIRED_SECTIONS[i + 1])
            if idx_a > idx_b:
                errors.append(f"Section order violation: ## {REQUIRED_SECTIONS[i]} must come before ## {REQUIRED_SECTIONS[i + 1]}")

    # Check Consequences is a bullet list
    consequences_match = re.search(r"## Consequences\s*\n((?:[^\n].*\n)*)", body)
    if consequences_match:
        consequences_body = consequences_match.group(1)
        bullets = [l for l in consequences_body.strip().split("\n") if l.strip().startswith("- ")]
        if not bullets:
            errors.append("## Consequences must contain bullet points (lines starting with '- ')")

    return errors


def validate_ascii(body: str) -> list[str]:
    """Validate that prose is ASCII-only."""
    # Remove code blocks before checking
    clean = CODE_BLOCK_PATTERN.sub("", body)
    return check_ascii_only(clean)


def validate_index(adrs_dir: Path, all_parsed: dict[str, dict]) -> list[str]:
    """Validate INDEX.md completeness and consistency."""
    errors = []
    index_path = adrs_dir / "INDEX.md"

    if not index_path.exists():
        errors.append("INDEX.md missing in ADR directory")
        return errors

    index_content = index_path.read_text(encoding="utf-8")

    # Extract referenced ADR IDs from INDEX.md links (format: [NNNN](filename.md))
    referenced_ids: set[str] = set()
    for match in re.finditer(r"\[(\d{4})\]", index_content):
        referenced_ids.add(match.group(1))

    # Build set of known IDs from parsed ADRs
    known_ids: set[str] = set()
    for adr_id, parsed in all_parsed.items():
        fm = parsed.get("frontmatter", {})
        if fm and "id" in fm:
            known_ids.add(str(fm["id"]).split("-", 1)[0])

    # Check: every known ADR should be in INDEX.md
    for known_id in sorted(known_ids):
        if known_id not in referenced_ids:
            errors.append(f"INDEX.md missing entry for ADR {known_id}")

    # Check: INDEX.md should not reference non-existent ADRs
    for ref_id in sorted(referenced_ids):
        if ref_id not in known_ids:
            errors.append(f"INDEX.md references non-existent ADR {ref_id}")

    return errors


def validate_cross_references(all_adrs: dict[str, dict]) -> list[str]:
    """Validate cross-references between ADRs."""
    errors = []
    all_ids = set(all_adrs.keys())

    for adr_id, parsed in all_adrs.items():
        fm = parsed.get("frontmatter", {})
        if not fm or "error" in parsed:
            continue

        superseded_by = str(fm.get("superseded_by", "")).strip()
        supersedes = str(fm.get("supersedes", "")).strip()

        if superseded_by and superseded_by not in all_ids:
            errors.append(f"{adr_id}: superseded_by references non-existent ADR '{superseded_by}'")

        if supersedes and supersedes not in all_ids:
            errors.append(f"{adr_id}: supersedes references non-existent ADR '{supersedes}'")

    # Check for status consistency: superseded_by implies old ADR is superseded
    for adr_id, parsed in all_adrs.items():
        fm = parsed.get("frontmatter", {})
        if not fm or "error" in parsed:
            continue
        superseded_by = str(fm.get("superseded_by", "")).strip()
        if superseded_by and superseded_by in all_ids:
            old_fm = all_adrs.get(superseded_by, {}).get("frontmatter", {})
            if old_fm and "error" not in all_adrs.get(superseded_by, {}):
                old_status = str(old_fm.get("status", "")).strip()
                if old_status != "superseded":
                    errors.append(f"{adr_id}: superseded_by '{superseded_by}' exists but has status '{old_status}', expected 'superseded'")

    # Check for circular dependencies
    def follow_chain(start: str, visited: set, depth: int) -> str | None:
        if depth > 10:
            return f"Circular dependency detected (depth > 10) involving {start}"
        fm = all_adrs.get(start, {}).get("frontmatter", {})
        if not fm or "error" in fm:
            return None
        next_id = str(fm.get("superseded_by", "")).strip()
        if not next_id:
            return None
        if next_id in visited:
            return f"Circular dependency: {' → '.join(visited)} → {next_id}"
        visited.add(next_id)
        return follow_chain(next_id, visited, depth + 1)

    for adr_id in all_ids:
        result = follow_chain(adr_id, {adr_id}, 1)
        if result:
            errors.append(result)

    return errors


def validate_single(filepath: Path) -> tuple[bool, list[str]]:
    """Validate a single ADR file. Returns (passed, errors)."""
    errors = []

    if not filepath.exists():
        return False, [f"File not found: {filepath}"]

    parsed = parse_adr(filepath)
    if "error" in parsed:
        return False, [f"Parse error: {parsed['error']}"]

    fm = parsed.get("frontmatter", {})
    body = parsed.get("body", "")

    errors.extend(validate_frontmatter(fm))
    errors.extend(validate_markdown(body))
    errors.extend(validate_ascii(body))

    return len(errors) == 0, errors


def main():
    parser = argparse.ArgumentParser(description="Validate ADR files")
    parser.add_argument("target", help="ADR file path or directory")
    parser.add_argument("--all", action="store_true", help="Validate all ADRs in directory with cross-references")
    args = parser.parse_args()

    target = Path(args.target)

    if args.all and target.is_dir():
        # Validate all ADRs in directory
        adr_files = sorted(f for f in target.glob("*.md") if f.name != "INDEX.md")
        if not adr_files:
            print(f"ERROR: No .md files found in {target}", file=sys.stderr)
            sys.exit(1)

        all_parsed: dict[str, dict] = {}
        all_errors: list[str] = []

        for adr_file in adr_files:
            parsed = parse_adr(adr_file)
            fm = parsed.get("frontmatter", {})
            adr_id = fm.get("id", adr_file.stem) if fm else adr_file.stem
            all_parsed[str(adr_id)] = parsed

        # Validate each ADR individually
        for adr_id, parsed in all_parsed.items():
            if "error" in parsed:
                all_errors.append(f"{adr_id}: {parsed['error']}")
                continue

            fm = parsed.get("frontmatter", {})
            body = parsed.get("body", "")

            fm_errors = validate_frontmatter(fm)
            md_errors = validate_markdown(body)
            ascii_errors = validate_ascii(body)

            for err in fm_errors + md_errors + ascii_errors:
                all_errors.append(f"{adr_id}: {err}")

        # Cross-reference validation
        all_errors.extend(validate_cross_references(all_parsed))

        # INDEX.md completeness validation
        all_errors.extend(validate_index(target, all_parsed))

        if all_errors:
            for err in all_errors:
                print(f"ERROR: {err}", file=sys.stderr)
            print(f"INVALID: {len(all_errors)} error(s) found", file=sys.stderr)
            sys.exit(1)

        print(f"VALID: all {len(all_parsed)} ADRs pass checks")
        sys.exit(0)

    elif target.is_file():
        passed, errors = validate_single(target)
        if errors:
            for err in errors:
                print(f"ERROR: {err}", file=sys.stderr)
            print(f"INVALID: {len(errors)} error(s) found", file=sys.stderr)
            sys.exit(1)
        print(f"VALID: {target.name} passes all checks")
        sys.exit(0)

    else:
        print(f"ERROR: {target} not found", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
