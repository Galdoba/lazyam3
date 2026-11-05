#!/usr/bin/env python3
"""Pre-execution branch check for rpi-implement.

Verifies the current git branch matches the expected implement branch
`implement/[problem-short-name]` before implementation begins.

Usage:
    python3 .agents/skills/rpi-implement/scripts/check_branch.py [problem-short-name]
"""

import subprocess
import sys


def current_branch():
    result = subprocess.run(
        ["git", "branch", "--show-current"],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        return None
    return result.stdout.strip()


def main():
    if len(sys.argv) < 2:
        print("usage: check_branch.py <problem-short-name>", file=sys.stderr)
        sys.exit(2)

    problem = sys.argv[1]
    expected_branch = f"implement/{problem}"
    current = current_branch()

    if current is None:
        print("ERROR: not in a git repository or git branch --show-current failed", file=sys.stderr)
        sys.exit(3)

    if current != expected_branch:
        print(
            f"BRANCH CHECK FAILED: current branch '{current}', "
            f"expected '{expected_branch}'. Create the implement branch before starting.",
            file=sys.stderr,
        )
        sys.exit(1)

    print(f"OK: on branch '{current}'")
    sys.exit(0)


if __name__ == "__main__":
    main()
