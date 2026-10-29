# Sanitation Problem — Verification Results

**Status:** VERIFIED — ACCEPT (proposed artifact-based solution is viable)

This document records the codebase verification of the artifact-based sanitation proposal from `sanitation_problem_statement.md`.

## Verdict and Reasoning

**Verdict: ACCEPT.** The proposed artifact-based deletion is implementable and correct.

Reasoning:

1. **Artifact content format** — JSON with `source_dir`, `unit_id`, `script_name` is fully parseable via Go's `encoding/json`. The `MediaUnit.ID` is `int64` (confirmed in `internal/domain/ingest/unit.go`), matching the proposed JSON schema.

2. **Path consistency** — `fso.Scan` normalizes all paths via `filepath.Abs(root)`, so scanned directories (returned by `TopDirsAllFilesWritable()`) are absolute paths (e.g. `/home/galdoba/.../incoming/simple-unit`). Storing this same absolute path as `source_dir` in the artifact guarantees an exact match at scan start.

3. **Architecture separation** — `fso.Scan` scans only and returns a Snapshot; the pipeline identifies empty dirs from the snapshot (`len(snap.FilesIn(d.Path)) == 0`), correlates them with `.done` files, and passes the final list to `fso` for deletion. This is the shared entry point for both `process` (loop) and `process --once`.

4. **Rename preserves content** — `buildDoneMarker` emits `[[ -f "metadata/script.processing" ]] && mv "metadata/script.processing" "metadata/script.done"`. The script template calls `done_marker` after successful ffmpeg execution (and `set -o errexit` ensures it does not run on failure). `mv` preserves file content, so the JSON payload survives the `.processing`→`.done` transition.

5. **Metadata dir location** — Artifacts live in `filepath.Join(p.destination, "metadata")` (confirmed in `pipeline.go` line 208), which persists across scan cycles.

## Plan for Implementation (IF solution accepted)

1. **`pipeline.go` `processUnit`** — change the `.processing` artifact from empty to JSON content:
   ```go
   payload := fmt.Sprintf(`{"source_dir": "%s", "unit_id": %d, "script_name": "%s"}`, dir, unit.ID, scriptName)
   if err := os.WriteFile(processingPath, []byte(payload), 0o644); err != nil { ... }
   ```
   `dir` is already the absolute scanned path; `unit.ID` is `int64`; `scriptName` is derived from `cfg.Destination`.

2. **`fso.Snapshot`** — add a `SanitizeablePaths []string` field (paths identified as empty + safe to delete) and a `Sanitize() error` method that deletes those listed directories via fso file operations.

3. **`pipeline.go` `RunOnce`** — after scan:
   - Identify empty dirs from the snapshot (`len(snap.FilesIn(d.Path)) == 0`).
   - Correlate each with a `.done` file in `p.destination/metadata` whose JSON `source_dir` matches `d.Path` exactly.
   - Populate `snap.SanitizeablePaths` with matching paths.
   - If the list is non-empty, call `snap.Sanitize()` (pipeline's responsibility to decide when), then remove the consumed `.done` files to prevent accumulation.
   - Continue with normal ingestion/metadata gathering for non-empty dirs.

4. **Edge-case handling**:
   - Empty dir with no matching `.done` file: leave it (first cycle before processing, or dir emptied for other reasons).
   - `.done` file points to a non-matching path: skip deletion.
   - Invalid/unparseable JSON in `.done`: log warning, skip.

## What We MUST Be Aware Of

1. **`.done` file accumulation** — if the consumed `.done` file is *not* deleted after processing its `source_dir`, stale `.done` files for non-existent directories will accumulate in `destination/metadata` over many cycles. **Recommendation:** delete the `.done` file immediately after successfully deleting the `source_dir`.

2. **"Empty" definition** — a directory is only safe to delete when it has zero entries via `os.ReadDir`. This includes hidden files (e.g. leftover sidecars). If `prepareFiles` leaves any file behind (files not in `callNames`), the dir is not empty and must NOT be deleted.

3. **External execution dependency** — the external executor must successfully rename `.processing`→`.done` and preserve content. If the executor fails to do so, `.done` files will not be created and deletion will not trigger.

4. **No cross-cycle collision** — `scriptName` includes a PRT timestamp (`Klan_soprano_s05e02_PRT261028165749`), so `.processing`/`.done` filenames are unique per unit per cycle. No overwrite collisions expected.

## Cost of Implementation

- **Low.** Modifications span two modules:
  - `fso.Snapshot`: add `SanitizeablePaths []string` field and `Sanitize() error` method (file-system deletion operations).
  - `pipeline.go processUnit`: JSON payload for `.processing` artifact (lines ~210-215).
  - `pipeline.go RunOnce`: empty-dir correlation with `.done` files, populate `SanitizeablePaths`, call `snap.Sanitize()`, remove consumed `.done` files.
- No domain-layer changes required; `buildDoneMarker` already handles the rename.
- Standard library only: `encoding/json`, `os.RemoveAll` (inside fso.Sanitize), metadata-dir file listing.

## Benefits

- **Eliminates scan overhead** — deleted directories never appear in the scan snapshot, so the scan loop iterates fewer (or zero) entries per unit.
- **Keeps test ground clean** — `incoming/<unit>/` directories are removed once processed, preventing accumulation of empty dirs.
- **Self-contained** — state travels with the artifact; no lazyam storage tracking or `status.json` coupling needed for this feature.
- **Reproducible** — works identically in `process` (loop) and `process --once` since both route through `RunOnce`.
