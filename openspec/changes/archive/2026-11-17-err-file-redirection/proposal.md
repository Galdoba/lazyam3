# Proposal

## Why

The operator's render pipeline writes a non-fatal error log (`<SOURCE>.#err`, collected by ffmpeg/fflite during an otherwise successful render) next to the media in the in-progress directory. The operator's notification handler picks that log up and announces it under the source file's name, but no lazyam3-generated script ever puts it beside the finished render, so the log cannot be correlated with the deliverables it describes, and the storage record gives no trace that the render had problems. A render that produced visible defects looks identical to a clean one.

## What Changes

- Generated scripts copy `.ready` and `.#err` (if present) from the in-progress directory into the render output (target) directory, naming both files after the primary render output's base name (`<OUTBASE>.ready`, `<OUTBASE>.#err`). The original files in the in-progress directory are left untouched; the operator's notification handler owns their cleanup.
- Generated scripts also copy `.#err` into the metadata directory as `<SCRIPT_NAME>.err`, containing the full text of the ffmpeg/fflite error log. This sidecar is the communication channel between the script and the pipeline.
- lazyam3 maintains a separate error log store (JSONL, analogous to the main processing history store). Each entry records the script name, source directory, and the full error log text. Only renders that actually produced an error log get an entry, so the store holds nothing but real failures. lazyam3 only appends to this store — it does not analyze or correlate error logs.
- The error store entry is written by the pipeline after a render completes and its processing history entry is committed, keyed by the script name and source directory, so any downstream consumer can correlate an error with its render and source.
- The notification directory is no longer a required executor path. Generated scripts no longer write completion markers into it. The operator's notification handler continues to clean up `.ready` and `.#err` files from the in-progress directory on its own schedule.
- The two naming families become explicit policy: script-name for anything lazyam3 reads back (`<SCRIPT_NAME>.sh`, `.processing`, `.done`, `.err`), render-outbase for anything lazyam3 places for the operator (`<OUTBASE>.ready`, `<OUTBASE>.#err` in the target directory). The exception is the announcement name the operator's handler gives a log it collected itself, which lazyam3 leaves alone.

Not in scope: the operator's notification handler, which is not modified; hard render failures (non-zero ffmpeg exit still leaves `.processing` dangling); and error log analysis or correlation logic.

## Capabilities

### New Capabilities

- `script-generation`: what a generated rendering script emits in the render output directory, and the naming it uses for those artifacts.
- `error-store`: a separate, append-only JSONL store for error log records, keyed by script name and source directory.
- `execution-signals`: the marker channel between a generated script and the pipeline (processing, done, err sidecar) and the record the pipeline commits to storage from it.

### Modified Capabilities

None. No specs exist yet in this project; `script-generation` and `error-store` are declared new, with `execution-signals` capturing the marker contract this change extends.

## Impact

- `internal/domain/script/template.go` - new placeholder for the `.ready`/`.#err` copy block from in-progress to target, including sidecar copy to metadata directory.
- `internal/domain/script/script.go` - render-outbase derivation factored out of `buildReadyFile`, shared by the copy builder. New `buildCopyFromInProgress` helper that copies both target and sidecar.
- `internal/app/pipeline/pipeline.go` - after a completed render, reads `<SCRIPT_NAME>.err` sidecar from metadata directory; when it exists, writes an entry to the error store with the full error text; removes sidecar alongside the done artifact.
- `internal/infra/persistence/jsonstore/` - second store instance for error logs.
- `internal/infra/config/config.go` - mandatory error store path and status path in config. No new executor paths.
- `internal/domain/script/config.go` - `Environment` no longer requires `Notifications` for script generation; `MetadataDir` already present and used for sidecar.
- Tests: `internal/domain/script/script_test.go`, `internal/app/script/build_test.go`, `internal/app/script/build_e2e_test.go`, `internal/app/pipeline/pipeline_test.go`; end-to-end check via `scripts/live_test.sh`.
- Generated scripts on existing executor hosts change shape: post-render block copies to target + metadata sidecar, no notification writes.
- Downstream: no change is required in the operator's notification handler; its existing behavior already covers the announcement and cleanup.
