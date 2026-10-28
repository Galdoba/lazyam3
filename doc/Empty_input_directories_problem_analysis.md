# Empty Input Directories — Problem Analysis

## Root Problem Statement

When the pipeline processes a media unit:
1. The pipeline **scans** a directory in `sourceDir` for writable media files.
2. `prepareFiles()` **moves** the source media files out of the input directory (into the output/destination).
3. On the **next scan cycle**, the same directory is empty.
4. The scanner detects an empty (or invalid) directory and emits a `ScanError` — logging a repeated, noisy error on every pass.

## Current Behavior (Reproduces the Bug)

```
Pass 1:  Scan `/home/user/lazyam3/input/`
  → finds 3 media units, processes them
  → moves 3 × 5 files = 15 files out of the input dir
  → logs: "scan: 3 writable directories found"
  → logs: "processing: unit 63729.mp4"
  → logs: "UnitDone(63729)"

Pass 2:  Scan `/home/user/lazyam3/input/`
  → no writable directories (empty now)
  → emits ScanError
  → logs: "scan failed: [Errno 1] no such file or directory: /home/user/lazyam3/input"
```

Each additional pass repeats the empty-directory scan error. This is visible in `PROGRESS.md` under the "gaps" section: a related logging issue about empty placeholder separators exists, but the root cause is the directory emptiness.

## Why Files Move Out of the Input Directory

`processUnit()` in `internal/app/pipeline/pipeline.go` calls `p.prepareFiles(dir, cfg.CallNames, p.destination)`, which copies/moves files from the input directory into `destination/buffer/` and `destination/in_progress/`. This is intentional — the destination is also where scripts land (`destination/scripts/<id>.sh`), so the pipeline assumes the processed media stays in a single destination.

## Impact

| Symptom | Description | Severity |
|---|---|---|
| Repeated `ScanError` per cycle | Empty directory triggers a new scan error each poll interval | Medium (noisy, repetitive) |
| Wasted polling cycles | The pipeline polls for `defaultPollInterval` (1s) even though nothing can change | Low |
| Confusing log noise | Scans show `dirs found: 0` or report errors instead of clean "nothing to do" | Medium |

## Proposed Solutions (Backlog)

### Solution A — Defer Deletion (Delete-after-Success)

**Idea:** Don't remove input files until the script has verified successful execution.

**Mechanism:**
- Add a temporary marker file per unit inside the input directory.
- After `script.Render()` succeeds, the marker stays (or the script itself serves as the marker).
- On the next scan cycle, skip directories that contain a marker (file exists, but it's a "done" artifact).
- Only purge old markers during a dedicated cleanup pass.

**Artifacts:**
- The script artifact itself (e.g., `/destination/scripts/63729.sh`) can serve as the success signal — but the pipeline doesn't track where scripts are archived.
- Alternatively: a small `.done` file placed by the script into a known marker directory under the input path.

**Trade-offs:**
- **Pro:** Simple to reason about; keeps source files until done.
- **Con:** Input directories accumulate "done" markers indefinitely; need a strategy to prune them.

### Solution B — Marker Artifact in Output Directory

**Idea:** The script writes a marker file into the **output** directory, and the pipeline checks it on the next pass.

**Mechanism:**
- `processUnit()` creates a marker file: `destination/markers/<unit_id>.done` (or `destination/metadata/unit_<unit_id>.done`).
- On the next scan, the pipeline first checks `destination/markers/` for the presence of `<id>.done`.
- If present, skip that unit directory — treat it as "already processed."
- This way, the pipeline knows exactly which units have completed, and can safely ignore them.

**Trade-offs:**
- **Pro:** Marker lives alongside the scripts; easy to query; clean separation from input directories.
- **Con:** The marker directory must be maintained; must handle marker staleness/cleanup.

### Solution C — Archive Scripts to Known Location

**Idea:** Scripts are archived to a known path, and the pipeline uses that archive to determine which units are complete.

**Options:**
- **Option 1 — Local path for scripts:** Store scripts at a fixed local path (e.g., `destination/scripts/`) and query that directory to determine which units are done. The pipeline knows this path (it's configured as `paths.generator.script_directory` or similar).
- **Option 2 — Infuse a signal via marker file (report?):** The script itself writes a "report" or marker file into `destination/` that encodes which unit it processed. The pipeline reads this report on the next scan.

**Trade-offs:**
- **Pro:** Centralized tracking; easy to implement (just list files in a known directory).
- **Con:** Requires coordination between the script (infrastructure) and the pipeline (application) to agree on the marker format.

## Architecture Implications

The pipeline's current design assumes that files **move** from input to output, and that input directories are "consumed" during processing. This is consistent with a single-pass or batch workflow, but conflicts with the polling model that repeatedly scans the same `sourceDir` for new or changed units.

**Key tension:**
- Single-pass → files are consumed (moved out) → input dirs become empty → polling errors on next pass.
- Polling → input dirs must stay intact (or be marked) → conflict with "move files out" design.

## Proposed Resolution (Recommendation)

**Combine B + C — Marker-based tracking in output:**

1. On `processUnit()` completion, write a marker file:
   ```
   destination/metadata/<unit_id>.done
   ```

2. On the next `RunOnce()` scan, before calling `ingest.Organizer.Ingest(dir)`:
   ```
   if exists(destination/metadata/<unit_id>.done) → skip dir (unit already processed)
   ```

3. The script artifact (`destination/scripts/<unit_id>.sh`) is the primary output; the marker is the tracking signal.

4. Optionally: add a **dedicated cleanup task** that purges markers older than N days, to avoid unbounded growth.

## Open Questions

- **Where should the marker live?**
  - Option A: `destination/markers/<unit_id>.done` — clean namespace, easy to prune.
  - Option B: `destination/metadata/<unit_id>.done` — co-located with metadata lookups.
  - Option C: `destination/scripts/<unit_id>.done` — co-located with scripts.
- **How to handle script failures?** If `build` fails, the marker should NOT be written. The directory remains unprocessed and is retried on the next cycle.
- **What about dry-run mode?** In `dryRun=true`, files are not moved and scripts are not written. The marker should also be skipped — or a dry-run marker should be used to prevent double-processing.

## Next Steps

1. **Add marker file writing** to `processUnit()` after successful script rendering.
2. **Add marker check** to the scan loop — skip directories with a completed marker.
3. **Update `RunOnce`** to handle the case where all directories are marked (emit a clean "nothing to do" event instead of a scan error).
4. **Add tests** for:
   - Marker is written after successful processing.
   - Marker is NOT written after failed processing.
   - Second scan skips already-marked directories.
   - Dry-run mode does not create markers.
