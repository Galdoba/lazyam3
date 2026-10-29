#!/usr/bin/env bash
# landing.sh - validate a Problem Statement document as output of discovery-formalizer.
# Usage: ./landing.sh <problem_statement_path>
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

if [[ ! -s "$PS_FILE" ]]; then
  echo "FAIL: Problem Statement file is empty: $PS_FILE"
  exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)

FAILED=0
WARNINGS=0

# ASCII-only check
echo "=== ASCII-only ==="
NON_ASCII=$(grep -Pn '[^\x09\x0A\x0D\x20-\x7E]' "$PS_FILE" 2>/dev/null || true)
if [[ -n "$NON_ASCII" ]]; then
  echo "FAIL: non-ASCII characters found:"
  echo "$NON_ASCII" | head -20
  FAILED=1
else
  echo "PASS: all characters are ASCII"
fi

# ── 1. Frontmatter fields ─────────────────────────────────────────
echo ""
echo "=== Frontmatter ==="
FRONTMATTER=$(sed -n '/^---$/,/^---$/p' "$PS_FILE" | head -n -1 | tail -n +2)

# Strict field order check
EXPECTED_ORDER="problem_id|title|repository|baseline_commit|problem_type|status"
ACTUAL_ORDER=$(echo "$FRONTMATTER" | grep -E '^[a-z_]+:' | sed 's/:.*//')
EXPECTED_ORDER_STR=$(echo "$EXPECTED_ORDER" | tr '|' '\n' | tr '\n' '|')
ACTUAL_ORDER_STR=$(echo "$ACTUAL_ORDER" | tr '\n' '|')

if [[ "$ACTUAL_ORDER_STR" == "$EXPECTED_ORDER_STR" ]]; then
  echo "PASS: frontmatter field order correct"
else
  echo "FAIL: frontmatter field order incorrect"
  echo "  expected: $(echo "$EXPECTED_ORDER" | tr '|' ' | ')"
  echo "  got:      $(echo "$ACTUAL_ORDER" | tr '\n' ' | ')"
  FAILED=1
fi

for field in problem_id title repository baseline_commit problem_type status; do
  value=$(echo "$FRONTMATTER" | grep -E "^${field}:" | head -1 | sed "s/^${field}:[[:space:]]*//")
  if [[ -z "$value" ]]; then
    echo "FAIL: missing frontmatter field: $field"
    FAILED=1
  else
    echo "PASS: $field = $value"
  fi
done

# problem_type validation
PT=$(echo "$FRONTMATTER" | grep -E "^problem_type:" | head -1 | sed 's/^problem_type:[[:space:]]*//')
if [[ -n "$PT" ]]; then
  if echo "$PT" | grep -qE "^(feature|bug|improvement)$"; then
    echo "PASS: problem_type valid ($PT)"
  else
    echo "FAIL: problem_type must be feature|bug|improvement (got: $PT)"
    FAILED=1
  fi
fi

# status must be DRAFT
ST=$(echo "$FRONTMATTER" | grep -E "^status:" | head -1 | sed 's/^status:[[:space:]]*//')
if [[ -n "$ST" && "$ST" == "DRAFT" ]]; then
  echo "PASS: status is DRAFT"
else
  echo "FAIL: status must be DRAFT (got: ${ST:-<empty>})"
  FAILED=1
fi

# baseline_commit must match current HEAD
BC=$(echo "$FRONTMATTER" | grep -E "^baseline_commit:" | head -1 | sed 's/^baseline_commit:[[:space:]]*//')
CURRENT_HEAD=$(git rev-parse HEAD 2>/dev/null || echo "")
if [[ -z "$BC" ]]; then
  echo "FAIL: baseline_commit is empty"
  FAILED=1
elif [[ "$BC" == "$CURRENT_HEAD" ]]; then
  echo "PASS: baseline_commit matches HEAD ($BC)"
else
  echo "FAIL: baseline_commit ($BC) does not match HEAD ($CURRENT_HEAD)"
  FAILED=1
fi

# ── 2. Body sections present and non-empty ─────────────────────────
echo ""
echo "=== Body sections ==="
for section in "current_behavior" "desired_behavior" "rationale"; do
  if grep -q "^## $section" "$PS_FILE"; then
    content=$(awk "/^## $section/{found=1; next} /^## /{found=0} found" "$PS_FILE" | head -10)
    if [[ -n "$(echo "$content" | tr -d '[:space:]')" ]]; then
      echo "PASS: ## $section present and non-empty"
    else
      echo "FAIL: ## $section is empty"
      FAILED=1
    fi
  else
    echo "FAIL: ## $section not found"
    FAILED=1
  fi
done

# ── 3. acceptance_criteria: each entry must have criterion + verification_method ──
echo ""
echo "=== acceptance_criteria ==="
AC_SECTION=$(awk '/^## acceptance_criteria/{found=1; next} /^## /{found=0} found' "$PS_FILE")
if [[ -z "$AC_SECTION" ]]; then
  echo "FAIL: ## acceptance_criteria not found"
  FAILED=1
else
  CRIT_COUNT=$(echo "$AC_SECTION" | grep -c "^  - criterion:" || true)
  VMETHOD_COUNT=$(echo "$AC_SECTION" | grep -c "^    verification_method:" || true)
  if [[ "$CRIT_COUNT" -eq 0 ]]; then
    echo "FAIL: no acceptance criteria found"
    FAILED=1
  else
    echo "PASS: $CRIT_COUNT acceptance criteria found"
  fi
  if [[ "$VMETHOD_COUNT" -eq 0 ]]; then
    echo "FAIL: no verification_method fields in acceptance_criteria"
    FAILED=1
  elif [[ "$VMETHOD_COUNT" -lt "$CRIT_COUNT" ]]; then
    echo "WARN: $CRIT_COUNT criteria but only $VMETHOD_COUNT verification_methods"
    WARNINGS=1
  else
    echo "PASS: all criteria have verification_method"
  fi
fi

# ── 4. evidence: each entry must have type + reference ─────────────
echo ""
echo "=== evidence ==="
EVID_SECTION=$(awk '/^## evidence:/{found=1; next} /^## /{found=0} found' "$PS_FILE")
if [[ -z "$EVID_SECTION" ]]; then
  echo "FAIL: ## evidence not found"
  FAILED=1
else
  EVID_COUNT=$(echo "$EVID_SECTION" | grep -c "^  - type:" || true)
  REF_COUNT=$(echo "$EVID_SECTION" | grep -c "^    reference:" || true)
  if [[ "$EVID_COUNT" -eq 0 ]]; then
    echo "FAIL: no evidence entries found"
    FAILED=1
  else
    echo "PASS: $EVID_COUNT evidence entries found"
  fi
  if [[ "$REF_COUNT" -eq 0 ]]; then
    echo "FAIL: no reference fields in evidence"
    FAILED=1
  elif [[ "$REF_COUNT" -lt "$EVID_COUNT" ]]; then
    echo "WARN: $EVID_COUNT evidence entries but only $REF_COUNT references"
    WARNINGS=1
  else
    echo "PASS: all evidence entries have reference"
  fi

  # 7. Verify evidence references point to real files
  echo ""
  echo "=== Evidence file references ==="
  while IFS= read -r ref_path; do
    # Strip leading/trailing whitespace, quotes, and parenthetical descriptions
    ref_path=$(echo "$ref_path" | sed -E 's/^[[:space:]]*//;s/[[:space:]]*$//;s/^"//;s/"$//;s/[[:space:]]+\(.*\)//')
    if [[ -z "$ref_path" ]]; then
      continue
    fi
    # Extract file path - strip line numbers, em-dashes, and descriptions
    file_path=$(echo "$ref_path" | sed -E 's/:[0-9]+(-[0-9]+)?//;s/[[:space:]]+[---].*//')
    if [[ -f "$REPO_ROOT/$file_path" ]]; then
      echo "PASS: evidence references existing file: $file_path"
    else
      echo "WARN: evidence references non-existing file: $file_path"
      WARNINGS=1
    fi
  done < <(echo "$EVID_SECTION" | grep "^    reference:" | sed 's/^    reference:[[:space:]]*//')
fi

# ── 5. non_goals: at least 3 ──────────────────────────────────────
echo ""
echo "=== non_goals ==="
NG_SECTION=$(awk '/^## non_goals:/{found=1; next} /^## /{found=0} found' "$PS_FILE")
if [[ -z "$NG_SECTION" ]]; then
  echo "FAIL: ## non_goals not found"
  FAILED=1
else
  NG_COUNT=$(echo "$NG_SECTION" | grep -c "^  - " || true)
  if [[ "$NG_COUNT" -lt 3 ]]; then
    echo "FAIL: only $NG_COUNT non_goals (need at least 3)"
    FAILED=1
  else
    echo "PASS: $NG_COUNT non_goals (>= 3)"
  fi
fi

# ── 6. constraints: at least 2 ────────────────────────────────────
echo ""
echo "=== constraints ==="
C_SECTION=$(awk '/^## constraints:/{found=1; next} /^## /{found=0} found' "$PS_FILE")
if [[ -z "$C_SECTION" ]]; then
  echo "FAIL: ## constraints not found"
  FAILED=1
else
  C_COUNT=$(echo "$C_SECTION" | grep -c "^  - " || true)
  if [[ "$C_COUNT" -lt 2 ]]; then
    echo "FAIL: only $C_COUNT constraints (need at least 2)"
    FAILED=1
  else
    echo "PASS: $C_COUNT constraints (>= 2)"
  fi
fi

# ── 7. open_questions: warn if empty ──────────────────────────────
echo ""
echo "=== open_questions ==="
OQ_SECTION=$(awk '/^## open_questions:/{found=1; next} /^## /{found=0} found' "$PS_FILE")
if [[ -z "$OQ_SECTION" ]]; then
  echo "WARN: ## open_questions not found (suggest adding unresolved items)"
  WARNINGS=1
else
  OQ_COUNT=$(echo "$OQ_SECTION" | grep -c "^  - " || true)
  if [[ "$OQ_COUNT" -eq 0 ]]; then
    echo "WARN: ## open_questions is empty (consider adding unresolved items)"
    WARNINGS=1
  else
    echo "PASS: $OQ_COUNT open questions"
  fi
fi

# ── 8. Feature tracking entry ─────────────────────────────────────
echo ""
echo "=== Feature tracking ==="
PID=$(echo "$FRONTMATTER" | grep -E "^problem_id:" | head -1 | sed 's/^problem_id:[[:space:]]*//')
if [[ -n "$PID" ]]; then
  FOUND_IN_FEATURES=0
  if [[ -f "$REPO_ROOT/doc/feature_list.json" ]]; then
    if grep -q "\"$PID\"" "$REPO_ROOT/doc/feature_list.json"; then
      FOUND_IN_FEATURES=1
    fi
  fi
  if [[ -f "$REPO_ROOT/doc/BACKLOG.md" ]]; then
    if grep -q "$PID" "$REPO_ROOT/doc/BACKLOG.md"; then
      FOUND_IN_FEATURES=1
    fi
  fi
  if [[ "$FOUND_IN_FEATURES" -eq 1 ]]; then
    echo "PASS: feature_id '$PID' found in feature tracking"
  else
    echo "WARN: feature_id '$PID' not found in feature_list.json or BACKLOG.md"
    WARNINGS=1
  fi
fi

# ── Result ────────────────────────────────────────────────────────
echo ""
if [[ "$FAILED" -eq 1 ]]; then
  echo "RESULT: LANDING - FAIL (fix issues before proceeding to solution-search)"
  exit 1
elif [[ "$WARNINGS" -eq 1 ]]; then
  echo "RESULT: LANDING - WARN (issues found, review before proceeding)"
  exit 0
else
  echo "RESULT: LANDING - PASS (all gates passed, ready for solution-search)"
  exit 0
fi
