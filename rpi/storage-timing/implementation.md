# Storage Timing: Data Loss Risk in Pipeline Loop Implementation

## Summary
Determined that `storage.json` should be updated when `.processing` is renamed to `.done` (execution complete), not at script-build time. This avoids a false-positive where `storage.json` marks work "done" while the external executor is still processing (minutes to hours). The store is updated via `captureProcessingHistory` when `.done` files are detected and the source directory is empty, and the status totals file was separated from `storage.json` (`status.json`) so `writeStatus` no longer corrupts store data. Build, lint, and tests pass.

## Pull Requests
| PR | Description | Status |
|----|-------------|--------|
| [#1](https://github.com/Galdoba/lazyam3/pull/1) | Storage timing: write-through persistence in pipeline loop | Draft (PR to be opened) |

## Unexpected Discoveries
The status totals were being written to the same `storage.json` path as the processing-history store, so `writeStatus(total)` overwrote the store data on every pass. The `.processing` to `.done` rename happens in the generated bash script, not Go code, so Go detects completion when `.done` files are found at the start of the next cycle.

## Plan Deviations
The plan specified adding `store.Save()` at the end of `RunOnce()`, but storing at build time would create a false-positive for the inter-process pipeline (work isn't done until execution completes). The correct approach relies on the existing `captureProcessingHistory` to persist when `.done` files are found, plus separating the status file path to prevent corruption.

## Architecture Decision Records
### Store on .done Detection (Business Logic)

context:
Work has two stages: build script (cheap, ~1s) and process media (external executor, minutes to hours). From the stakeholder's perspective, work is not done until the script finishes processing.

options:
- Store in `processUnit` at build time (false-positive risk).
- Store in `captureProcessingHistory` when `.done` files are found (execution complete).

decision:
Rely on `captureProcessingHistory` to persist processing history when `.done` files are detected and the source directory is empty.

rationale:
Storing at `.done` detection avoids false-positives for the inter-process pipeline: `storage.json` only records work that has actually completed. A missed signal is less dangerous than a false positive for the larger pipeline of which lazyam3 is an element.

consequences:
- `storage.json` is updated at the start of the cycle after `.done` appears (~1s poll interval).
- Build-time data is not recorded, avoiding false positives.

### Separate Status Path from Store Path

context:
`NewPipeline` set `statusPath` to `opts.CFG.Storage.Path`, and the store also uses `Storage.Path`, so both wrote to `storage.json` and collided.

options:
- Write status to a separate `status.json` file.
- Write store to a subdirectory.

decision:
Set `statusPath` to `filepath.Join(filepath.Dir(Storage.Path), "status.json")`.

rationale:
Keeps status totals and processing-history data in the same directory but distinct files, avoiding data corruption.

consequences:
- Status totals are now in `status.json`, processing history in `storage.json`.

## Handoff Notes

### Known Limitations
- `storage.json` is updated at the start of the cycle after `.done` files appear, not at the exact rename moment.
- Store save failures are logged as warnings and do not abort processing.

### Data Models
- `jsonstore.Store`: in-memory key-value map (`data: map[string]any`) backed by a JSON file at `path`, guarded by `sync.RWMutex`, with `Set`, `Get`, `Delete`, `Save`, `Close` methods.

### API Surface
- `Store.Save() error`: flushes the current in-memory contents to the JSON file atomically via snapshot and `os.WriteFile`.

## Reference
- internal/app/pipeline/pipeline.go
- internal/app/pipeline/pipeline_test.go
- internal/infra/persistence/jsonstore/jsonstore.go
