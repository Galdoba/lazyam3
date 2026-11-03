#!/usr/bin/env bash
set -euo pipefail

# ── paths ──────────────────────────────────────────────────────────────
TEST_GROUND="$HOME/workspace/lazyam3"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SYNTHETIC="$REPO_ROOT/testdata/synthetic"
METADATA_SRC="$REPO_ROOT/testdata/metadata.json"

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

# ── 4. hot-reload demo: run continuous process, modify metadata, verify reload ──
echo "=== metadata hot-reload demo ==="
METADATA_COPY="$TEST_GROUND/metadata.json"
cp "$METADATA_SRC" "$METADATA_COPY"

echo "starting continuous process (dormancy 1s)..."
/tmp/lazyam3_bin process --metadata-path "$METADATA_COPY" --dormancy 1s &
PID=$!
# wait for first scan pass to complete
sleep 2
# modify metadata.json to trigger reload on next scan pass
sed -i 's/"hardsub": "auto"/"hardsub": "auto", "reload_demo": true/' "$METADATA_COPY"
touch -d "now + 10 seconds" "$METADATA_COPY"
# wait for second scan pass
sleep 2
kill "$PID" 2>/dev/null || true
wait "$PID" 2>/dev/null || true

echo "reload log lines:"
grep -i "metadata reloaded\|metadata reload failed" "$TEST_GROUND/lazyam.log" || echo "  (no reload lines)"
rm -f "$METADATA_COPY"

# ── 5. print file tree ────────────────────────────────────────────────
echo ""
echo "=== files in test ground ==="
find "$TEST_GROUND" -type f | sort
