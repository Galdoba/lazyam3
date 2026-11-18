#!/usr/bin/env bash
set -euo pipefail

# Full test gate: compile, vet, unit suite, lint, then both live pipeline runs.
# Each step keeps its own log, so a failure can be read without re-running.

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LOG_DIR="$(mktemp -d /tmp/lazyam3-full-test.XXXXXX)"
FAILURES=0

step() {
  local name="$1"
  shift
  printf '=== %s ===\n' "$name"
  if "$@" >"$LOG_DIR/$name.log" 2>&1; then
    echo "PASS: $name"
  else
    tail -n 40 "$LOG_DIR/$name.log" >&2
    echo "FAIL: $name"
    FAILURES=$((FAILURES + 1))
  fi
}

cd "$REPO_ROOT"

mkdir -p "$HOME/workspace/lazyam3/incoming"

step go-build go build ./...
step go-vet go vet ./...
step go-test go test -race -cover ./...
step lint golangci-lint run ./...
step live-test "$REPO_ROOT/scripts/live_test.sh"
step live-test-executor "$REPO_ROOT/scripts/live_test_2.sh"

echo ""
echo "=== full test summary ==="
if [ "$FAILURES" -eq 0 ]; then
  echo "FULL TEST PASSED"
  echo "logs: $LOG_DIR"
else
  echo "$FAILURES step(s) failed"
  echo "logs: $LOG_DIR"
  exit 1
fi
