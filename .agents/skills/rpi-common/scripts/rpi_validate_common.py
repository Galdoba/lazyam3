"""Common deterministic validation utilities for rpi-* skills.

Shared across rpi-plan, rpi-research, and rpi-implement validation scripts:
- ValidationError class
- load_lines helper
- check_no_frontmatter
- check_ascii_only
- extract_code_references
- check_code_references_resolve
- write_handoff_marker for workflow handoff markers
"""

import json
import re
import sys
from datetime import datetime, timezone
from pathlib import Path


class ValidationError(Exception):
    pass


def write_handoff_marker(status_path, stage, problem, document, facts_mean=None, tasks_count=None):
    """Write the RPI workflow status marker JSON file at repo root.

    status_path: Path to .rpi-workflow.json at repo root
    stage: current completed stage ('research', 'plan', 'implement')
    problem: problem directory name (e.g. 'phase-1-notification')
    document: relative document path that just passed (e.g. 'plan.md')
    """
    next_stage_map = {"research": "plan", "plan": "implement", "implement": "none"}
    next_doc_map = {"research": "plan.md", "plan": "implementation.md", "implement": "none"}
    next_stage = next_stage_map.get(stage, "none")
    next_document = next_doc_map.get(stage, "none")

    marker = {
        "current_problem": problem,
        "last_completed_stage": stage,
        "last_completed_document": document,
        "proposed_next_stage": next_stage,
        "updated_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    }
    if next_stage != "none":
        marker["proposed_next_document"] = next_document
    if facts_mean is not None:
        marker["facts_mean"] = facts_mean
    if tasks_count is not None:
        marker["tasks_count"] = tasks_count

    status_path.parent.mkdir(parents=True, exist_ok=True)
    status_path.write_text(json.dumps(marker, indent=2) + "\n", encoding="utf-8")


def load_lines(path: Path):
    return path.read_text(encoding="utf-8").splitlines()


def check_no_frontmatter(lines, artifact_name: str):
    """Frontmatter must not be present (no leading `---`)."""
    stripped = [l.strip() for l in lines]
    if stripped and stripped[0] == "---":
        raise ValidationError(
            f"YAML frontmatter detected at line 1; {artifact_name}.md must not have frontmatter"
        )


def check_ascii_only(lines):
    """Document text (outside code blocks and inline code) must be ASCII only."""
    in_code_block = False
    for i, l in enumerate(lines):
        if l.strip().startswith("```"):
            in_code_block = not in_code_block
            continue
        if in_code_block:
            continue
        text_without_inline = re.sub(r"`[^`]*`", "", l)
        for j, ch in enumerate(text_without_inline):
            if ord(ch) > 127:
                raise ValidationError(
                    f"Non-ASCII character {ch!r} (U+{ord(ch):04X}) at line {i+1}, "
                    f"column {j+1}; document must be ASCII only (outside code blocks)"
                )


def extract_code_references(lines):
    """Extract code references: path.ext:LINE, path.ext:START-END, or path.ext (entire file)."""
    refs = []
    pattern = re.compile(r"([a-zA-Z0-9_./-]+\.(?:go|md|py|js|ts|html|json|sql|txt))(:[0-9]+(?:-[0-9]+)?)?")
    for i, l in enumerate(lines):
        for m in pattern.finditer(l):
            path = m.group(1)
            line_part = m.group(2)[1:] if m.group(2) else ""
            refs.append((i + 1, f"{path}:{line_part}" if line_part else path))
    return refs


def check_code_references_resolve(refs, base_dir: Path, context: str):
    """Code references with line numbers must resolve to existing files with lines in bounds.
    Entire-file references (no line number) are skipped (may be [NEW] files)."""
    for line_no, ref in refs:
        colon_idx = ref.find(":")
        if colon_idx == -1:
            continue  # entire-file reference, skip existence check
        rel_path = ref[:colon_idx]
        parts = ref[colon_idx + 1:].split(";")
        file_path = (base_dir / rel_path).resolve()
        if not file_path.exists():
            raise ValidationError(f"Code reference at {context} line {line_no} points to non-existent file: {rel_path}")
        file_lines = file_path.read_text(encoding="utf-8", errors="ignore").splitlines()
        for p in parts:
            if not p:
                continue
            if "-" in p:
                start, end = p.split("-")
                if not (1 <= int(start) <= len(file_lines) and 1 <= int(end) <= len(file_lines)):
                    raise ValidationError(f"Code reference line range out of bounds in {rel_path}: {p}")
            else:
                ln = int(p)
                if not (1 <= ln <= len(file_lines)):
                    raise ValidationError(f"Code reference line {ln} out of bounds in {rel_path}")


def print_usage(script_name: str, path_arg: str):
    """Print usage to stdout (for --help)."""
    print(f"usage: {script_name} <{path_arg}>")
    print(f"       {script_name} --help")
    print(f"       {script_name} --json <{path_arg}>")


def main_usage(script_name: str, path_arg: str):
    print(f"usage: {script_name} <{path_arg}>", file=sys.stderr)
