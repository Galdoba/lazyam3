#!/usr/bin/env bash
# preflight.sh - go/no-go gates for solution-search skill.
# Usage: ./preflight.sh <problem_statement_path>
# Exit 0 = all gates pass, exit 1 = one or more gates failed.

set -euo pipefail

PS_FILE="${1:-}"

if [[ -z "$PS_FILE" ]]; then
  echo "FAIL: no Problem Statement path provided"
  echo "Usage: $0 <path/to/problem_statement.md>"
  exit 1
fi

if [[ ! -f "$PS_FILE" ]]; then
  echo "FAIL: Problem Statement file does not exist: $PS_FILE"
  exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
FAILED=0

# ── 1. Git working tree must be clean ─────────────────────────────
echo "=== Git clean check ==="
if git diff --quiet 2>/dev/null; then
  echo "PASS: git working tree is clean (stable starting point)"
else
  echo "FAIL: git working tree has uncommitted changes"
  git status --short
  FAILED=1
fi

# ── 2. Status must be CONFIRMED ───────────────────────────────────
echo ""
echo "=== Status check ==="
FRONTMATTER=$(sed -n '/^---$/,/^---$/p' "$PS_FILE" | head -n -1 | tail -n +2)
ST=$(echo "$FRONTMATTER" | grep -E "^status:" | head -1 | sed 's/^status:[[:space:]]*//')
if [[ -n "$ST" && "$ST" == "CONFIRMED" ]]; then
  echo "PASS: status is CONFIRMED"
else
  echo "FAIL: status must be CONFIRMED for solution-search (got: ${ST:-<empty>})"
  FAILED=1
fi

# ── 3. Codebase accessible ────────────────────────────────────────
echo ""
echo "=== Codebase check ==="
if [[ -d "$REPO_ROOT/internal" ]]; then
  echo "PASS: codebase accessible ($REPO_ROOT/internal exists)"
else
  echo "FAIL: codebase structure unclear (no internal/ dir at $REPO_ROOT)"
  FAILED=1
fi

# ── Result ────────────────────────────────────────────────────────
echo ""
if [[ "$FAILED" -eq 1 ]]; then
  echo "RESULT: GO/NO-GO - NO (fix issues before starting solution-search)"
  exit 1
fi

echo "RESULT: GO/NO-GO - YES (all gates passed, start solution-search)"

# ── Output Problem Statement content ──────────────────────────────
# Pass the full PS document to the agent so it doesn't need to read it again.
echo ""
echo "=== Problem Statement ==="
cat "$PS_FILE"
exit 0
