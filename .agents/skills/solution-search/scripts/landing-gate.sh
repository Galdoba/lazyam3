#!/usr/bin/env bash
# landing-gate.sh - Validate a Solution Verdict document.
# Usage: ./landing-gate.sh <path/to/solution_options.md>
# Exit 0 = all gates pass, exit 1 = one or more gates failed.
#
# Checks:
#   HARD (1): file exists, frontmatter fields, verdict value, approaches count, approach structure, verdict section
#   SOFT (0): open_questions section present

set -euo pipefail

DOC_FILE="${1:-}"

if [[ -z "$DOC_FILE" ]]; then
  echo "FAIL: no document path provided"
  echo "Usage: $0 <path/to/solution_options.md>"
  exit 1
fi

if [[ ! -f "$DOC_FILE" ]]; then
  echo "FAIL: document does not exist: $DOC_FILE"
  exit 1
fi

FAILED=0
WARNINGS=0
declare -a WARN_MSGS=()

# ── Helper functions ────────────────────────────────────────────────

# Extract YAML frontmatter (between first and second ---)
get_frontmatter() {
  sed -n '/^---$/,/^---$/p' "$DOC_FILE" | head -n -1 | tail -n +2
}

# Get a frontmatter field value
get_field() {
  local field="$1"
  get_frontmatter | grep -E "^${field}:" | head -1 | sed "s/^${field}:[[:space:]]*//"
}

# Check if a markdown heading exists
heading_exists() {
  local heading="$1"
  grep -q "^## ${heading}" "$DOC_FILE" 2>/dev/null
}

# Count approach sections
count_approaches() {
  grep -c "^### Approach" "$DOC_FILE" 2>/dev/null || echo "0"
}

# ── Gate checks ────────────────────────────────────────────────────

echo "=== Landing Gate: Solution Verdict Validation ==="
echo "Document: $DOC_FILE"
echo ""

# ── 1. File exists ────────────────────────────────────────────────
echo "=== Gate 1: File exists ==="
if [[ -f "$DOC_FILE" ]]; then
  echo "PASS"
else
  echo "FAIL: $DOC_FILE not found"
  FAILED=1
fi

# ── 2. Frontmatter: problem_id ────────────────────────────────────
echo "=== Gate 2: Frontmatter problem_id ==="
PID=$(get_field "problem_id")
if [[ -n "$PID" ]]; then
  echo "PASS: problem_id = $PID"
else
  echo "FAIL: missing or empty problem_id"
  FAILED=1
fi

# ── 3. Frontmatter: title ─────────────────────────────────────────
echo "=== Gate 3: Frontmatter title ==="
TITLE=$(get_field "title")
if [[ -n "$TITLE" ]]; then
  echo "PASS: title present"
else
  echo "FAIL: missing or empty title"
  FAILED=1
fi

# ── 4. Frontmatter: generated_at ──────────────────────────────────
echo "=== Gate 4: Frontmatter generated_at ==="
GEN_AT=$(get_field "generated_at")
if [[ -n "$GEN_AT" ]]; then
  echo "PASS: generated_at = $GEN_AT"
else
  echo "FAIL: missing or empty generated_at"
  FAILED=1
fi

# ── 5. Frontmatter: verdict ───────────────────────────────────────
echo "=== Gate 5: Frontmatter verdict ==="
VERDICT=$(get_field "verdict")
if [[ "$VERDICT" == "APPROVED" || "$VERDICT" == "REJECTED" || "$VERDICT" == "DEFERRED" ]]; then
  echo "PASS: verdict = $VERDICT"
else
  echo "FAIL: verdict must be APPROVED, REJECTED, or DEFERRED (got: ${VERDICT:-<empty>})"
  FAILED=1
fi

# ── 6. Approaches count (≥ 2) ─────────────────────────────────────
echo "=== Gate 6: Approaches count ==="
APPROACH_COUNT=$(count_approaches)
if [[ "$APPROACH_COUNT" -ge 2 ]]; then
  echo "PASS: $APPROACH_COUNT approaches found"
else
  echo "FAIL: need at least 2 approaches (got: $APPROACH_COUNT)"
  FAILED=1
fi

# ── 7. Approach structure ─────────────────────────────────────────
echo "=== Gate 7: Approach structure ==="
# Check that at least one approach has Description, Pros, Cons
HAS_DESCRIPTION=$(grep -c "\*\*Description:\*\*" "$DOC_FILE" 2>/dev/null || echo "0")
HAS_PROS=$(grep -c "\*\*Pros:\*\*" "$DOC_FILE" 2>/dev/null || echo "0")
HAS_CONS=$(grep -c "\*\*Cons:\*\*" "$DOC_FILE" 2>/dev/null || echo "0")

if [[ "$HAS_DESCRIPTION" -ge 2 && "$HAS_PROS" -ge 2 && "$HAS_CONS" -ge 2 ]]; then
  echo "PASS: approaches have Description, Pros, Cons"
else
  echo "FAIL: approaches missing Description ($HAS_DESCRIPTION), Pros ($HAS_PROS), or Cons ($HAS_CONS)"
  FAILED=1
fi

# ── 8. Verdict section ────────────────────────────────────────────
echo "=== Gate 8: Verdict section ==="
if heading_exists "verdict"; then
  echo "PASS: ## verdict section exists"
else
  echo "FAIL: missing ## verdict section"
  FAILED=1
fi

# ── 9. Rationale present ──────────────────────────────────────────
echo "=== Gate 9: Rationale ==="
if grep -q "\*\*Rationale:\*\*" "$DOC_FILE" 2>/dev/null; then
  echo "PASS: rationale present"
else
  echo "FAIL: missing rationale"
  FAILED=1
fi

# ── 9.5 ASCII-only ────────────────────────────────────────────────
echo "=== Gate 9.5: ASCII-only ==="
NON_ASCII=$(grep -Pn '[^\x09\x0A\x0D\x20-\x7E]' "$DOC_FILE" 2>/dev/null || true)
if [[ -n "$NON_ASCII" ]]; then
  echo "FAIL: non-ASCII characters found:"
  echo "$NON_ASCII" | head -20
  FAILED=1
else
  echo "PASS: all characters are ASCII"
fi

# ── 10. Open questions (soft) ─────────────────────────────────────
echo "=== Gate 10: Open questions (soft) ==="
if heading_exists "open_questions"; then
  echo "PASS: ## open_questions section exists"
else
  WARNINGS=$((WARNINGS + 1))
  WARN_MSGS+=("## open_questions section missing")
  echo "WARN: ## open_questions section missing (non-blocking)"
fi

# ── Result ─────────────────────────────────────────────────────────
echo ""
echo "=== RESULT ==="
if [[ "$FAILED" -eq 0 ]]; then
  echo "PASS: all hard gates passed"
  if [[ "$WARNINGS" -gt 0 ]]; then
    echo "Warnings: ${#WARN_MSGS[@]}"
    for msg in "${WARN_MSGS[@]}"; do
      echo "  - $msg"
    done
  fi
  exit 0
else
  echo "FAIL: $FAILED hard gate(s) failed"
  if [[ "$WARNINGS" -gt 0 ]]; then
    echo "Warnings: ${#WARN_MSGS[@]}"
    for msg in "${WARN_MSGS[@]}"; do
      echo "  - $msg"
    done
  fi
  exit 1
fi
