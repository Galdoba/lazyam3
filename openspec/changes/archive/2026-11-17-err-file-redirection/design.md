# Design

## Context

Motivation is in proposal.md. The state that constrains the approach:

- A generated script runs under `set -o errexit` with its post-render steps chained by `&&` (`internal/domain/script/template.go`). An error log appears only after a *successful* ffmpeg/fflite exit, so it can live inside that chain rather than needing a failure path.
- The marker channel already exists. The pipeline writes `<SCRIPT_NAME>.processing` holding a single marshalled JSON payload just before handing the script over (`internal/app/pipeline/pipeline.go:337`). The script renames it to `.done` as its last act (`internal/domain/script/script.go:280`). On a later pass the pipeline matches `.done` by `source_dir`, then appends that same payload to the JSONL history store (`pipeline.go:394` through `pipeline.go:247`).
- The generating host and the executing host are different machines. Every path in `executorPaths` is embedded into the script as a string and is never resolved by the generator (`internal/app/pipeline/config.go:93`). The render output directory is therefore not reliably observable from the generator process.
- `Environment.Notifications` is already plumbed into the script environment and the template declares the variable, but no line in the template writes to it (`internal/domain/script/config.go:90`, `template.go:32`).
- The ready marker name is derived inline from the first video prediction, extension stripped, in `buildReadyFile` (`internal/domain/script/script.go:299`).
- The metadata directory (`MetadataDir`) is accessible from the generator host, making it the reliable channel for the script to communicate with the pipeline.

The operator's notification handler is fixed: it serves renders beyond lazyam3's, and will not be changed for this feature. Its behavior:

- Every five seconds it scans the in-progress directory for files whose extension is `#err`, copies each one into an error archive directory, and *moves* the in-progress copy into the notification directory.
- It also scans the in-progress directory for `.ready` files and moves them into the notification directory.
- It then scans the notification directory: for every file it reads the file content as the message body and derives the destination chat from the file's extension, so `.ready` and `#err` are routing keys. For `#err` it discards the content and announces the filename instead, and archives a copy. It then deletes the notification file.
- Net effect: any `.#err` or `.ready` file that appears in the in-progress directory is picked up by the handler, announced to the operator, and removed from the in-progress directory. The handler is the sole owner of in-progress cleanup.

## Goals / Non-Goals

**Goals:**

- One base name shared by every artifact lazyam3 itself produces, derived in exactly one place.
- The error log and ready marker reach the render output directory alongside the deliverables.
- A separate error store captures the full text of ffmpeg/fflite error logs, keyed by script name and source directory.
- The operator is not told twice about one error.
- No new executor paths required; the notification directory is not needed for script generation.

**Non-Goals:**

- Changing the operator's notification handler in any way.
- Failure handling. A non-zero ffmpeg exit still stops the chain and leaves `.processing` in place. The marker naming for a future failure signal is not decided here.
- Changing the ready marker's name or location, the `.processing`/`.done` contract, or the store's append-only semantics.
- Cleaning up the log's original location in the in-progress directory; the operator's notification handler owns that.
- Error log analysis or correlation logic.

## Decisions

**1. The script copies `.#err` from the in-progress directory into the target directory and touches the ready marker there.**

Both artifacts lazyam3 places for the operator live in the target directory: the ready marker is created there by the script, and the error log, if ffmpeg/fflite produced one, is copied from the in-progress directory where the render left it. Both are named after the primary render output's base name.

```bash
&& OUTBASE="<OUTBASE>" \
   if [ -f "${IN_PROGRESS}/${SOURCE}.#err" ]; then \
       cp -f "${IN_PROGRESS}/${SOURCE}.#err" "<TARGET_DIR>/${OUTBASE}.#err"; \
       cp -f "${IN_PROGRESS}/${SOURCE}.#err" "${METADATA_DIR}/${SCRIPT_NAME}.err"; \
   fi \
&& touch "<TARGET_DIR>/${OUTBASE}.ready"
```

`OUTBASE` is derived once from the first video prediction by a single helper, shared by the ready marker and the copied error log name. A clean render (no `.#err`) skips the copy and continues; the `&&` chain survives because the `if` block exits zero. The ready marker is touched unconditionally after the copy block. `OUTBASE` / `SCRIPT_NAME` variables are initialized because the script runs under `set -o nounset`.

The sidecar copy (`${METADATA_DIR}/${SCRIPT_NAME}.err`) contains the full text of the ffmpeg/fflite error log — not just a filename reference. This is the communication channel between the script and the pipeline.

Rejected alternatives: have the pipeline copy the file after the fact (the paths are executor-side, the generator host cannot resolve them); have the script move files (the operator's tooling owns the in-progress directory).

**2. No error-archive lookup, no `errors` executor path.**

The script only reads from the in-progress directory. If the operator's notifier has already swept the log, the copy is missed for this render — but the notifier's archive copy still exists on the executor, and no error store entry is written. This is acceptable because: the operator's archive is the durable copy; and short renders (where the log is most likely complete) are unaffected.

**3. Sidecar marker carries the full error log text.**

The script writes `<SCRIPT_NAME>.err` in the metadata directory containing the complete content of the ffmpeg/fflite error log. This file is the sole communication channel between the script and the pipeline for error reporting.

The sidecar uses the metadata directory because it is accessible from both the executor (where the script runs) and the generator (where the pipeline reads it). The `.done` marker already uses this directory, establishing the trust boundary.

Clean renders do not write a sidecar. The pipeline treats the absence of `<SCRIPT_NAME>.err` as "clean render" and writes no error store entry at all, so every entry in the store describes a render that actually produced an error log.

**4. Separate error log store.**

lazyam3 maintains a dedicated JSONL store for error log records, analogous to the main processing history store. Each entry contains:

```json
{
  "script_name": "Show_s01e06_PRT261023120000",
  "source_dir": "/path/to/unit/dir",
  "error_text": "ffmpeg version ... [h264 @ ...] non-monotonic timestamps ...\n...",
  "timestamp": "2026-11-17T12:00:00Z"
}
```

- `error_text` is the full text of the ffmpeg/fflite error log. An entry exists only because an error log was found, so the field is never written as an empty placeholder.
- The store is append-only. lazyam3 writes one entry per render that produced an error log, keyed by the script name.
- lazyam3 does not analyze, aggregate, or correlate error logs. It only records them.
- The store path is a mandatory configuration field, like the main storage path; lazyam3 opens it at startup and fails fast if it cannot be opened for writing.

The error store is written by the pipeline after a `.done` marker is consumed and the main processing history entry has been committed. The pipeline reads `<SCRIPT_NAME>.err` from the metadata directory; only when that sidecar exists does it take its content as `error_text` and append the entry. The sidecar is removed after a successful commit.

**5. Notification handler takes `.ready` and `.#err` from in-progress.**

The script never creates, updates, or removes anything inside the in-progress directory beyond reading the `.#err` it may find there. Its only `touch` is the ready marker in the target directory (`<TARGET_DIR>/${OUTBASE}.ready`), and an error log found next to the source is copied, never moved, so whatever the executor left stays exactly where it was. The operator's notification handler scans in-progress every five seconds, moves both `.#err` and `.ready` files out of in-progress into the notification directory, announces them, and deletes the notification copies. lazyam3 is agnostic to this cleanup; the handler is the sole owner of in-progress directory state.

**6. One derivation of `<OUTBASE>`.** Extract the base-name logic previously inlined in `buildReadyFile` into a single helper over the predictions, and have both operator-facing names that depend on it (the ready marker and the copied error log in the target directory) come from it. Parity between those two names is then structural, not something tests have to police. The error store entry is keyed by the script name instead, so it does not take part in this derivation (see decision 7).

**7. Two naming families, made explicit, with one documented exception.** Script-name for pipeline-read artifacts (`.processing`, `.done`, `.err`, the script itself), render-outbase for artifacts lazyam3 places for the operator (`<OUTBASE>.ready`, `<OUTBASE>.#err` in the target directory). The exception: the *announcement* of an error log keeps whatever name the operator's notifier chose for it, which lazyam3 leaves alone.

Alternative - one uniform name everywhere - was rejected: keying operator-facing files on the PRT-bearing script name would force the handler to learn lazyam3's internal naming, and keying markers on the outbase would break the correlation between a record and its script when the same product is rendered more than once.

**8. `cp`, never `mv`, and never a delete (in in-progress).** The in-progress directory belongs to the operator's tooling, which collects files on its own schedule.

**9. Sidecar and `.done` are cleaned up together.** The pipeline removes `.done` and `<SCRIPT_NAME>.err` in the same step that commits the processing history entry (together with the error store entry, when the render left an error log). If any store write fails the transaction aborts, so both markers survive for the next pass.

## Risks / Trade-offs

|- **The in-progress log may be missing if the notifier sweeps it mid-render.** The script finds nothing, writes no sidecar, and no error store entry is written, so the failure is invisible in the store. The operator's archive copy is still available on the executor for manual inspection.
|- A stale `<SCRIPT_NAME>.err` if a render wrote the sidecar but died before renaming `.processing` → the sidecar is only read alongside the matching `.done` for the same script name, and removed together with it, so an orphan sidecar is never read.
|- Clean renders produce no error store entry, so every entry is a real failure. The trade-off: the store no longer proves that a render was examined, and a render whose sidecar was swept away before the pipeline read it is indistinguishable from a clean one.
|- Nothing in this change helps a render that exits non-zero: `.processing` still hangs forever with no `.done`, no record, and no error store entry. Called out in proposal.md as deliberately out of scope.
|- The error store is a new file on the generator host. If the generator host runs out of disk space, error entries are lost. This is a separate concern from the main store and can be addressed independently.

## Migration Plan

No executor path keys are added or removed. The notification directory remains optional. Configuration gains `storage.error_store_path` and `storage.status_path` as mandatory fields (the status file path is no longer derived from the storage path); existing configs must add them or startup fails with a validation error. No stored-data migration: only newly generated scripts gain the new copy block, and history records written before the change lack the error store entry. Scripts already generated by earlier builds run unchanged under the old shape. Rollback is a revert; the operator's handler was never touched.

## Open Questions

- ~~Should the error store path default to a sibling of the main storage path (e.g., `storage.json` → `errors.json`), or be fully user-configured?~~ Resolved: fully user-configured and mandatory for both the error store and the status file; an unopenable path is an infrastructure error with an early return.
