#!/usr/bin/env bash
# repo-scanner.sh - Repository scanner for agent orientation.
# Auto-detects repo, delegates to language-specific scanners.
# Usage: ./repo-scanner.sh [--format json|markdown]
#
# For Go repos: delegates to scan.sh (if available).
# For other languages: built-in multi-language scanner.

set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
OUTPUT_FORMAT="markdown"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --format) OUTPUT_FORMAT="$2"; shift 2;;
    --help)
      echo "Usage: $0 [--format json|markdown]"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1;;
  esac
done

# ── Check for Go ────────────────────────────────────────────────────

has_go_files() {
  local count
  count=$(find "$REPO_ROOT" -type f -name "*.go" \
    -not -path '*/.git/*' \
    -not -path '*/node_modules/*' \
    -not -path '*/vendor/*' \
    2>/dev/null | wc -l)
  [[ "$count" -gt 0 ]]
}

# ── Delegate to scan.sh for Go ─────────────────────────────────────

if has_go_files; then
  SCAN_SH="$REPO_ROOT/scan.sh"
  if [[ -f "$SCAN_SH" && -x "$SCAN_SH" ]]; then
    echo "Go repository detected, delegating to scan.sh" >&2
    exec bash "$SCAN_SH" "$REPO_ROOT"
  else
    echo "Warning: scan.sh not found at $SCAN_SH, falling back to built-in scanner" >&2
  fi
fi

# ── Built-in multi-language scanner (non-Go fallback) ──────────────

# Get files by extension.
files_by_ext() {
  local ext="$1"
  find "$REPO_ROOT" -type f -name "*${ext}" \
    -not -path '*/.git/*' \
    -not -path '*/node_modules/*' \
    -not -path '*/vendor/*' \
    2>/dev/null | sort
}

# ── Go exports (built-in fallback) ─────────────────────────────────

extract_go_exports() {
  local file="$1"
  local rel="${file#$REPO_ROOT/}"

  local types functions methods

  types=$(grep -E '^\s*type\s+[A-Z][a-zA-Z0-9]*\s+(struct|interface|\(|map\[|chan\s+)' "$file" 2>/dev/null \
    | sed -E 's/^\s*type\s+([A-Z][a-zA-Z0-9]*)\s+.*/\1/' || true)

  functions=$(grep -E '^\s*func\s+[A-Z][a-zA-Z0-9]*\s*\(' "$file" 2>/dev/null \
    | sed -E 's/^\s*func\s+([A-Z][a-zA-Z0-9]*)\s*.*/\1/' || true)

  methods=$(grep -E '^\s*func\s*\(' "$file" 2>/dev/null \
    | grep -v '^\s*func\s\+[A-Z]' \
    | sed -E 's/^\s*func\s+\([^)]+\)\s+([A-Z][a-zA-Z0-9]*)\s*(\(.*|$)/\1/' || true)

  local exports="[]"
  local count=0

  while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    exports=$(echo "$exports" | jq --arg n "$name" --arg t "type" '. + [{"name": $n, "kind": $t}]')
    count=$((count + 1))
  done <<< "$types"

  while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    exports=$(echo "$exports" | jq --arg n "$name" --arg t "func" '. + [{"name": $n, "kind": $t}]')
    count=$((count + 1))
  done <<< "$functions"

  while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    exports=$(echo "$exports" | jq --arg n "$name" --arg t "method" '. + [{"name": $n, "kind": $t}]')
    count=$((count + 1))
  done <<< "$methods"

  jq -n \
    --arg path "$rel" \
    --argjson exports "$exports" \
    --argjson count "$count" \
    '{path: $path, language: "go", exports: $exports, export_count: $count}'
}

# ── Rust exports ────────────────────────────────────────────────────

extract_rust_exports() {
  local file="$1"
  local rel="${file#$REPO_ROOT/}"

  local types functions

  types=$(grep -E '^\s*pub\s+(struct|enum|union)\s+[A-Z]' "$file" 2>/dev/null \
    | sed -E 's/^\s*pub\s+(struct|enum|union)\s+([A-Z][a-zA-Z0-9_]*).*/\2/' || true)

  functions=$(grep -E '^\s*pub\s+fn\s+[a-z]' "$file" 2>/dev/null \
    | sed -E 's/^\s*pub\s+fn\s+([a-zA-Z0-9_]*).*/\1/' || true)

  local exports="[]"

  while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    exports=$(echo "$exports" | jq --arg n "$name" --arg t "type" '. + [{"name": $n, "kind": $t}]')
  done <<< "$types"

  while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    exports=$(echo "$exports" | jq --arg n "$name" --arg t "fn" '. + [{"name": $n, "kind": $t}]')
  done <<< "$functions"

  jq -n \
    --arg path "$rel" \
    --argjson exports "$exports" \
    '{path: $path, language: "rust", exports: $exports}'
}

# ── Python exports ──────────────────────────────────────────────────

extract_python_exports() {
  local file="$1"
  local rel="${file#$REPO_ROOT/}"

  local classes functions

  classes=$(grep -E '^\s*class\s+[A-Z][a-zA-Z0-9_]*' "$file" 2>/dev/null \
    | sed -E 's/^\s*class\s+([A-Z][a-zA-Z0-9_]*).*/\1/' || true)

  functions=$(grep -E '^\s*def\s+[a-z_][a-zA-Z0-9_]*\s*\(' "$file" 2>/dev/null \
    | grep -E '^def\s' \
    | sed -E 's/^\s*def\s+([a-zA-Z0-9_]*).*/\1/' || true)

  local exports="[]"

  while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    exports=$(echo "$exports" | jq --arg n "$name" --arg t "class" '. + [{"name": $n, "kind": $t}]')
  done <<< "$classes"

  while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    exports=$(echo "$exports" | jq --arg n "$name" --arg t "func" '. + [{"name": $n, "kind": $t}]')
  done <<< "$functions"

  jq -n \
    --arg path "$rel" \
    --argjson exports "$exports" \
    '{path: $path, language: "python", exports: $exports}'
}

# ── TS/JS exports ───────────────────────────────────────────────────

extract_ts_exports() {
  local file="$1"
  local rel="${file#$REPO_ROOT/}"

  local classes functions types

  classes=$(grep -E '^\s*export\s+(default\s+)?class\s+[A-Z]' "$file" 2>/dev/null \
    | sed -E 's/^\s*export\s+(default\s+)?class\s+([A-Z][a-zA-Z0-9_]*).*/\2/' || true)

  functions=$(grep -E '^\s*export\s+(const|let|var|function|async\s+function)\s+[A-Za-z_][A-Za-z0-9_]*' "$file" 2>/dev/null \
    | sed -E 's/^\s*export\s+(const|let|var|function|async\s+function)\s+([A-Za-z_][A-Za-z0-9_]*).*/\2/' || true)

  types=$(grep -E '^\s*export\s+(type|interface|enum)\s+[A-Z]' "$file" 2>/dev/null \
    | sed -E 's/^\s*export\s+(type|interface|enum)\s+([A-Z][A-Za-z0-9_]*).*/\2/' || true)

  local exports="[]"

  while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    exports=$(echo "$exports" | jq --arg n "$name" --arg t "class" '. + [{"name": $n, "kind": $t}]')
  done <<< "$classes"

  while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    exports=$(echo "$exports" | jq --arg n "$name" --arg t "func" '. + [{"name": $n, "kind": $t}]')
  done <<< "$functions"

  while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    exports=$(echo "$exports" | jq --arg n "$name" --arg t "type" '. + [{"name": $n, "kind": $t}]')
  done <<< "$types"

  jq -n \
    --arg path "$rel" \
    --argjson exports "$exports" \
    '{path: $path, language: "typescript", exports: $exports}'
}

# ── Build tree ──────────────────────────────────────────────────────

build_tree() {
  local lines=()
  while IFS= read -r line; do
    lines+=("$line")
  done < <(
    if command -v tree &>/dev/null; then
      tree --noreport --dirsfirst -L 4 --prune \
        --exclude '.git' \
        --exclude 'node_modules' \
        --exclude 'vendor' \
        --exclude '.cache' \
        --exclude '_build' \
        --exclude '__pycache__' \
        --exclude '.target' \
        --exclude '.cargo' \
        "$REPO_ROOT" 2>/dev/null || \
      find "$REPO_ROOT" -maxdepth 4 \
        -not -path '*/.git/*' \
        -not -path '*/node_modules/*' \
        -not -path '*/vendor/*' \
        -not -path '*/.cache/*' \
        -type f -o -type d \
        2>/dev/null | sed "s|$REPO_ROOT/||" | sort
    else
      find "$REPO_ROOT" -maxdepth 4 \
        -not -path '*/.git/*' \
        -not -path '*/node_modules/*' \
        -not -path '*/vendor/*' \
        -not -path '*/.cache/*' \
        -type f -o -type d \
        2>/dev/null | sed "s|$REPO_ROOT/||" | sort
    fi
  )

  printf '%s\n' "${lines[@]}" | jq -R . | jq -s .
}

# ── Count stats ─────────────────────────────────────────────────────

count_by_ext() {
  local total=0
  local entries=""

  for ext in go rs ts tsx js jsx py; do
    local count
    count=$(find "$REPO_ROOT" -type f -name "*.${ext}" \
      -not -path '*/.git/*' \
      -not -path '*/node_modules/*' \
      -not -path '*/vendor/*' \
      2>/dev/null | wc -l)
    if [[ "$count" -gt 0 ]]; then
      total=$((total + count))
      if [[ -n "$entries" ]]; then
        entries="${entries},"
      fi
      entries="${entries}{\"extension\":\"${ext}\",\"count\":${count}}"
    fi
  done

  jq -n \
    --argjson total "$total" \
    --argjson by_ext "[${entries}]" \
    '{total_files: $total, by_extension: $by_ext}'
}

# ── Main ────────────────────────────────────────────────────────────

main() {
  echo "Scanning repository: $REPO_ROOT" >&2

  local tree
  tree=$(build_tree)

  local stats
  stats=$(count_by_ext)

  local go_files rust_files ts_files py_files

  go_files=$(files_by_ext ".go")
  rust_files=$(files_by_ext ".rs")
  ts_files=$(find "$REPO_ROOT" -type f \( -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx" \) \
    -not -path '*/node_modules/*' -not -path '*/.git/*' \
    2>/dev/null | sort)
  py_files=$(files_by_ext ".py")

  local all_exports="[]"

  while IFS= read -r file; do
    [[ -z "$file" ]] && continue
    local export_json
    export_json=$(extract_go_exports "$file")
    all_exports=$(echo "$all_exports" | jq --argjson e "$export_json" '. + [$e]')
  done <<< "$go_files"

  while IFS= read -r file; do
    [[ -z "$file" ]] && continue
    local export_json
    export_json=$(extract_rust_exports "$file")
    all_exports=$(echo "$all_exports" | jq --argjson e "$export_json" '. + [$e]')
  done <<< "$rust_files"

  while IFS= read -r file; do
    [[ -z "$file" ]] && continue
    local export_json
    export_json=$(extract_ts_exports "$file")
    all_exports=$(echo "$all_exports" | jq --argjson e "$export_json" '. + [$e]')
  done <<< "$ts_files"

  while IFS= read -r file; do
    [[ -z "$file" ]] && continue
    local export_json
    export_json=$(extract_python_exports "$file")
    all_exports=$(echo "$all_exports" | jq --argjson e "$export_json" '. + [$e]')
  done <<< "$py_files"

  local git_info
  git_info=$(jq -n \
    --arg branch "$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo 'unknown')" \
    --arg latest "$(git log -1 --format='%h %s' 2>/dev/null || echo 'unknown')" \
    --argjson commits "$(git rev-list --count HEAD 2>/dev/null || echo 0)" \
    '{branch: $branch, latest_commit: $latest, total_commits: $commits}')

  local json_output
  json_output=$(jq -n \
    --arg root "$REPO_ROOT" \
    --argjson tree "$tree" \
    --argjson stats "$stats" \
    --argjson exports "$all_exports" \
    --argjson git "$git_info" \
    '{
      repository: $root,
      tree: $tree,
      stats: $stats,
      exports: $exports,
      git: $git
    }')

  if [[ "$OUTPUT_FORMAT" == "json" ]]; then
    echo "$json_output"
  fi

  if [[ "$OUTPUT_FORMAT" == "markdown" ]]; then
    local branch latest
    branch=$(echo "$git_info" | jq -r '.branch')
    latest=$(echo "$git_info" | jq -r '.latest_commit')
    local total_files
    total_files=$(echo "$stats" | jq -r '.total_files')
    echo "## Repository Map"
    echo ""
    echo "**Path:** \`$REPO_ROOT\`"
    echo "**Branch:** \`$branch\`"
    echo "**Latest:** \`$latest\`"
    echo ""
    echo "### Files"
    echo ""
    echo "Total source files: \`$total_files\`"
    echo ""
    echo "| Language | Count |"
    echo "|----------|-------|"
    echo "$stats" | jq -r '.by_extension[] | "| \(.extension) | \(.count) |"'
    echo ""
    echo "### Exports by file"
    echo ""
    echo "$all_exports" | jq -r '.[] | select(.export_count > 0) | "\(.path): \(.export_count) exports (\(.language))"'
  fi
}

main
