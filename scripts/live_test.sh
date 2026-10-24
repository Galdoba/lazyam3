#!/usr/bin/env bash
set -euo pipefail

# ── paths ──────────────────────────────────────────────────────────────
TEST_GROUND="$HOME/workspace/lazyam3"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SYNTHETIC="$REPO_ROOT/testdata/synthetic"

# ── 1. wipe files (keep directories) ──────────────────────────────────
echo "=== wiping test ground ==="
find "$TEST_GROUND" -type f -delete
echo "done."

# ── 2. copy synthetic test data ───────────────────────────────────────
echo "=== copying synthetic data → incoming ==="
mkdir -p "$TEST_GROUND/incoming"
cp -r "$SYNTHETIC/"* "$TEST_GROUND/incoming/"
echo "done."

# ── 3. build & run ────────────────────────────────────────────────────
echo "=== building lazyam3 ==="
go build -o /tmp/lazyam3_bin "$REPO_ROOT/cmd/lazyam3"
echo "=== running lazyam3 process --once ==="
/tmp/lazyam3_bin process --once

# ── 4. print file tree ────────────────────────────────────────────────
echo ""
echo "=== files in test ground ==="
find "$TEST_GROUND" -type f | sort
