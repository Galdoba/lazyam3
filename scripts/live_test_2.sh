#!/usr/bin/env bash
set -euo pipefail

# Executor simulation end-to-end, single consumption cycle (mixed pass):
# pass 1 generates scripts; every generated script is then executed, with an
# error log planted next to the in-progress source of the first three of them;
# one further `lazyam3 process --once` consumes every execution marker at once,
# so the same cycle commits both errored and clean renders.

TEST_GROUND="$HOME/workspace/lazyam3"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SYNTHETIC="$REPO_ROOT/testdata/synthetic"
BUFFER="$TEST_GROUND/buffer"
ERR_STORE="$BUFFER/errors.jsonl"
HISTORY="$BUFFER/history.jsonl"
STATUS="$BUFFER/status.json"
METADATA="$BUFFER/metadata"
ERROR_COUNT=3
PLANTED_MARK="ffmpeg error: simulated decode failure"

fail() { echo "FAIL: $*" >&2; exit 1; }

# ── 1. wipe files (keep directories) ──────────────────────────────────
echo "=== wiping test ground ==="
find "$TEST_GROUND" -type f -delete
echo "done."

# ── 2. copy synthetic test data ───────────────────────────────────────
echo "=== copying synthetic data → incoming ==="
mkdir -p "$TEST_GROUND/incoming"
cp -r "$SYNTHETIC/"* "$TEST_GROUND/incoming/"
echo "done."

# ── 3. build & pass 1 (generate scripts) ──────────────────────────────
echo "=== building lazyam3 ==="
go build -o /tmp/lazyam3_bin "$REPO_ROOT/cmd/lazyam3"
echo "=== pass 1: lazyam3 process --once ==="
/tmp/lazyam3_bin process --once

# ── 4. executor cycle: run every generated script ─────────────────────
mapfile -t SCRIPTS < <(find "$BUFFER" -maxdepth 1 -name '*.sh' | sort)
[ "${#SCRIPTS[@]}" -gt "$ERROR_COUNT" ] || fail "expected more than $ERROR_COUNT scripts, got ${#SCRIPTS[@]}"

errored_scripts=()
errored_sources=()
errored_outbases=()
errored_target_dirs=()
errored_texts=()
errored_in_progress=()
clean_scripts=()
clean_outbases=()
clean_target_dirs=()

echo "=== executor simulation: ${#SCRIPTS[@]} script(s), $ERROR_COUNT with error logs ==="
idx=0
for s in "${SCRIPTS[@]}"; do
  name=$(basename "$s" .sh)
  SOURCE=$(grep -m1 '^SOURCE="' "$s" | cut -d'"' -f2)
  in_progress=$(grep -m1 '^IN_PROGRESS="' "$s" | cut -d'"' -f2)
  OUTBASE=$(grep -m1 '^OUTBASE="' "$s" | cut -d'"' -f2)
  TARGET_DIR=$(grep -m1 '^TARGET_DIR="' "$s" | cut -d'"' -f2)

  if [ "$idx" -lt "$ERROR_COUNT" ]; then
    mkdir -p "$in_progress"
    printf '%s in %s\n' "$PLANTED_MARK" "$name" > "$in_progress/${SOURCE}.#err"
    errored_scripts+=("$name")
    errored_sources+=("$SOURCE")
    errored_outbases+=("$OUTBASE")
    errored_target_dirs+=("$TARGET_DIR")
    errored_texts+=("$PLANTED_MARK in $name")
    errored_in_progress+=("$in_progress")
    echo "planted error log for $name"
  else
    clean_scripts+=("$name")
    clean_outbases+=("$OUTBASE")
    clean_target_dirs+=("$TARGET_DIR")
  fi
  bash "$s"
  idx=$((idx + 1))
done

# ── 5. pass 2: consume every execution marker in one cycle ────────────
echo "=== pass 2: lazyam3 process --once ==="
/tmp/lazyam3_bin process --once

# ── 6. assertions ─────────────────────────────────────────────────────
echo "=== assertions ==="
[ -f "$ERR_STORE" ] || fail "error store not found: $ERR_STORE"
[ -f "$HISTORY" ] || fail "history store not found: $HISTORY"

[ "$(wc -l < "$ERR_STORE")" -eq "$ERROR_COUNT" ] ||
  fail "error store must hold exactly $ERROR_COUNT entries (one per errored render), got $(wc -l < "$ERR_STORE")"

errored_count_in_store=0
for i in "${!errored_scripts[@]}"; do
  name="${errored_scripts[$i]}"
  text="${errored_texts[$i]}"
  outbase="${errored_outbases[$i]}"
  target_dir="${errored_target_dirs[$i]}"
  source="${errored_sources[$i]}"
  in_progress="${errored_in_progress[$i]}"

  grep "\"script_name\":\"$name\"" "$ERR_STORE" | grep -q "$text" ||
    fail "error store has no entry for $name carrying its own error text"
  errored_count_in_store=$((errored_count_in_store + 1))

  err_copy="$target_dir/${outbase}.#err"
  ready="$target_dir/${outbase}.ready"
  [ -f "$err_copy" ] || fail "target missing error copy: $err_copy"
  grep -q "$text" "$err_copy" || fail "target error copy has wrong content: $err_copy"
  [ -f "$ready" ] || fail "target missing ready marker: $ready"

  [ ! -e "$METADATA/${name}.err" ] || fail "metadata sidecar not removed after commit: ${name}.err"
  [ ! -e "$METADATA/${name}.done" ] || fail "done marker not removed after commit: ${name}.done"

  [ -f "$in_progress/${source}.#err" ] || fail "in-progress .#err must remain (copy, never move): $source"
  [ -f "$TEST_GROUND/done/${source}" ] || fail "in-progress source must move to done: $source"
done
[ "$errored_count_in_store" -eq "$ERROR_COUNT" ] ||
  fail "expected $ERROR_COUNT errored scripts to be checked, got $errored_count_in_store"

[ "$(find "$TEST_GROUND/target" -name '*.#err' | wc -l)" -eq "$ERROR_COUNT" ] ||
  fail "expected exactly $ERROR_COUNT .#err files in target, got $(find "$TEST_GROUND/target" -name '*.#err' | wc -l)"

[ "${#clean_scripts[@]}" -gt 0 ] || fail "expected at least one clean render in this fixture"
for i in "${!clean_scripts[@]}"; do
  name="${clean_scripts[$i]}"
  outbase="${clean_outbases[$i]}"
  target_dir="${clean_target_dirs[$i]}"
  if grep -q "\"script_name\":\"$name\"" "$ERR_STORE"; then
    fail "clean render $name must not appear in the error store"
  fi
  if grep -q "\"script_name\":\"$name\"" "$HISTORY"; then
    [ -f "$target_dir/${outbase}.ready" ] || fail "clean render $name lost its ready marker"
    [ ! -e "$target_dir/${outbase}.#err" ] || fail "clean render $name must not have a .#err in target"
    [ ! -e "$METADATA/${name}.err" ] || fail "clean render $name must not leave a metadata sidecar"
  fi
done

[ "$(wc -l < "$HISTORY")" -gt "$ERROR_COUNT" ] ||
  fail "history store must also record the clean renders, got $(wc -l < "$HISTORY") entries"

[ -z "$(find "$TEST_GROUND/notifications" -type f)" ] || fail "notification directory must stay empty"

[ -f "$STATUS" ] || fail "status file not written: $STATUS"

echo "ALL ASSERTIONS PASSED"

# ── 7. print file tree ────────────────────────────────────────────────
echo ""
echo "=== files in test ground ==="
find "$TEST_GROUND" -type f | sort
