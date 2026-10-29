# Sanitation Problem — Artifact-Based Solution Proposal

## Problem

After `lazyam3 process`, source files are moved from `incoming/<unit>/` to `buffer/`. On the next scan cycle, `incoming/<unit>/` is empty, causing scan overhead and noisy logs. The pipeline currently **silently skips** empty directories, but the directories are never deleted, so they accumulate as empty dirs.

We want lazyam3 to **delete empty `incoming/<unit>/` directories** once the unit's script has executed successfully — eliminating scan overhead and keeping the test ground clean. This must be reproducible in both `lazyam3 process` (loop) and repeated `lazyam3 process --once`.

## Proposed Solution: Artifact-Based Information Passing

Rather than storing state in lazyam's storage (which is written at the end of each cycle), pass information via the **processing artifact itself**.

### Strategy

1. **At render moment** (after script string is ready, before external execution):
   - Create `.processing` artifact with **content** (JSON), not empty:
     ```json
     {"source_dir": "/home/galdoba/workspace/lazyam3/incoming/simple-unit", "unit_id": 12187, "script_name": "Klan_soprano_s05e02_PRT261028165749"}
     ```
   - Written to `<metadata_dir>/<script_name>.processing`

2. **Script execution** (external executor) renames `.processing` → `.done`, preserving content.

3. **At scan start** (beginning of `RunOnce`, before scan):
   - Read all `.done` files in `<metadata_dir>/`
   - Parse `source_dir` from each
   - If that `source_dir` exists and is empty → **safe to delete**
   - Delete it *before* scanning, so scan never encounters it

### Why This Works

- **Self-contained**: info travels with the artifact, no lazyam storage tracking needed.
- **Safe**: only deletes when `.done` exists (OK execution) AND dir is empty.
- **Reproducible**: works in both `process` (loop) and `process --once` — both go through `RunOnce`.
- **Eliminates scan overhead**: deleted dirs don't exist during scan.

### What the Next Agent Needs to Verify

1. **Artifact content format**: JSON with `source_dir`, `unit_id`, `script_name`. Must be parseable.
2. **Path consistency**: `source_dir` in artifact must match scanned dir path (absolute path, consistent format).
3. **Timing**: deletion must happen at scan start, before scan traversal, after reading `.done` files.
4. **Edge cases**: `.done` file for a dir that no longer exists; dir not empty (files still present); multiple `.done` files.

### Files to Touch

- `internal/domain/script/script.go`: `buildDoneMarker` or artifact content writer.
- `internal/app/pipeline/pipeline.go`: write `.processing` with JSON content; scan-start deletion logic reading `.done` files.
- Storage/config: metadata dir path for artifacts.
