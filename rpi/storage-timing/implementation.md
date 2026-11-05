# Storage Timing: Data Loss Risk in Pipeline Loop Implementation

## Summary
Implemented write-through persistence by storing processing history in the store immediately after `processUnit` successfully builds a script (`storeProcessingHistory`), and by separating the status file from the store storage path (`status.json` vs `storage.json`) so `writeStatus` no longer overwrites processing history. This closes the data-loss window that existed when data only reached `storage.json` on the next cycle's `captureProcessingHistory` call, and when the status file overwrote store data. Build, lint, and tests pass.

## Pull Requests
| PR | Description | Status |
|----|-------------|--------|
| [#1](https://github.com/Galdoba/lazyam3/pull/1) | Storage timing: write-through persistence in pipeline loop | Draft (PR to be opened) |

## Unexpected Discoveries
The status totals were being written to the same `storage.json` path as the processing-history store, so `writeStatus(total)` overwrote the store data on every pass. The `.processing` to `.done` rename happens in the generated bash script, not Go code, so processing history must be stored in `processUnit` right after the script builds, not in `captureProcessingHistory`.

## Plan Deviations
The plan specified adding `store.Save()` at the end of `RunOnce()`, but that only saves whatever was already in the store (processing history is added in `processUnit`). The actual fix stores history in `processUnit` via `storeProcessingHistory`, and additionally fixed a path collision where status and store shared `storage.json`.

## Architecture Decision Records
### Store Processing History in processUnit

context:
Processing history (script metadata) needs to reach `storage.json` immediately after a unit is processed, before the program can be stopped or crash. Previously it was only stored in `captureProcessingHistory` from `.done` files of the previous cycle.

options:
- Store history in `processUnit` after successful script build (write-through).
- Store history in `captureProcessingHistory` only (plan, ineffective).

decision:
Added `storeProcessingHistory(scriptName, payload)` called after the `.processing` artifact is written in `processUnit`, persisting to the store immediately.

rationale:
This closes the data-loss window: data is in `storage.json` as soon as the script builds, regardless of when the next cycle's `captureProcessingHistory` runs or whether the program stops.

consequences:
- Each successful unit triggers a `store.Save()`, adding disk writes proportional to processed units.
- The store and status now use separate files (`storage.json` vs `status.json`).

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
- Each successfully processed unit triggers a `store.Save()`, adding disk writes. For large batches this is acceptable given the small JSON file size.
- Store save failures are logged as warnings and do not abort processing.

### Data Models
- `jsonstore.Store`: in-memory key-value map (`data: map[string]any`) backed by a JSON file at `path`, guarded by `sync.RWMutex`, with `Set`, `Get`, `Delete`, `Save`, `Close` methods.

### API Surface
- `Store.Save() error`: flushes the current in-memory contents to the JSON file atomically via snapshot and `os.WriteFile`.

## Reference
- internal/app/pipeline/pipeline.go
- internal/app/pipeline/pipeline_test.go
- internal/infra/persistence/jsonstore/jsonstore.go
