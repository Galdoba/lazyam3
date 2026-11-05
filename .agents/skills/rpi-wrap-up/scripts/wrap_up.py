#!/usr/bin/env python3
"""Execute the RPI wrap-up phase: archive intermediate documents.

Usage:
  wrap_up.py <cycle-name> [--dry-run]

Arguments:
  cycle-name        Name of the RPI cycle (e.g. 'logger-debug-level')
  --dry-run         Show what would be done without making changes

This script only archives documents. ADR extraction is done by the agent
per the rpi-wrap-up SKILL.md instructions.
"""

import argparse
import json
import shutil
import sys
from datetime import datetime, timezone
from pathlib import Path

# ── Paths ────────────────────────────────────────────────────────────
RPI_DIR = Path("rpi")


def write_handoff_marker(status_path, stage, problem, document):
    """Write the RPI workflow status marker JSON file at repo root."""
    marker = {
        "current_problem": problem,
        "last_completed_stage": stage,
        "last_completed_document": document,
        "proposed_next_stage": "none",
        "proposed_next_document": "none",
        "updated_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    }
    status_path.parent.mkdir(parents=True, exist_ok=True)
    status_path.write_text(json.dumps(marker, indent=2) + "\n", encoding="utf-8")


def archive_cycle_docs(cycle_name: str, dry_run: bool):
    """Archive intermediate documents for a cycle."""
    cycle_dir = RPI_DIR / cycle_name
    archive_dir = RPI_DIR / "archive" / cycle_name

    if not cycle_dir.exists():
        print(f"WARNING: {cycle_dir} does not exist, skipping archive")
        return

    docs_to_archive = ["research.md", "plan.md", "implementation.md"]
    existing_docs = [d for d in docs_to_archive if (cycle_dir / d).exists()]

    if not existing_docs:
        print(f"INFO: No documents to archive in {cycle_dir}")
        return

    if dry_run:
        print(f"DRY-RUN: Would archive {len(existing_docs)} docs to {archive_dir}")
        for doc in existing_docs:
            print(f"  {cycle_dir}/{doc} -> {archive_dir}/{doc}")
        return

    archive_dir.mkdir(parents=True, exist_ok=True)

    for doc in existing_docs:
        src = cycle_dir / doc
        dst = archive_dir / doc
        shutil.move(str(src), str(dst))
        print(f"  Archived: {src} -> {dst}")

    # Remove empty cycle directory
    remaining = [f for f in cycle_dir.iterdir() if f.is_file()]
    if not remaining:
        cycle_dir.rmdir()
        print(f"  Removed empty directory: {cycle_dir}")


def main():
    parser = argparse.ArgumentParser(description="RPI Wrap-Up: archive intermediate documents")
    parser.add_argument("cycle_name", help="Name of the RPI cycle (e.g. logger-debug-level)")
    parser.add_argument("--dry-run", action="store_true", help="Show what would be done")
    args = parser.parse_args()

    cycle_name = args.cycle_name
    repo_root = Path(__file__).resolve().parent.parent.parent.parent

    print(f"RPI Wrap-Up: {cycle_name}")
    print()

    archive_cycle_docs(cycle_name, args.dry_run)

    if not args.dry_run:
        write_handoff_marker(
            repo_root / ".rpi-workflow.json",
            "wrap-up",
            cycle_name,
            "archive",
        )
        print()
        print("Wrap-up complete.")


if __name__ == "__main__":
    main()
