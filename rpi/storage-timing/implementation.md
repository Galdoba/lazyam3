# Storage Timing: Data Loss Risk in Pipeline Loop Implementation

## Summary
Implemented write-through persistence by adding a guarded `p.store.Save()` call at the end of `RunOnce()`, after the unit processing loop and before the `EventScanDone` emission. This ensures processed unit data reaches `storage.json` immediately after each pass, closing the data-loss window that existed when data was only persisted on shutdown or the next pass. All tasks in the plan were completed; build, lint, and tests pass.

## Pull Requests
| PR | Description | Status |
|----|-------------|--------|
| (draft) | Storage timing: write-through persistence in pipeline loop | Draft |

## Unexpected Discoveries
The `jsonstore.Store.Save()` method is already thread-safe via `sync.RWMutex` and performs a full in-memory snapshot before JSON marshaling, so concurrent reads during the write are safe. The `captureProcessingHistory` path already calls `store.Save()` on transaction completion, so the new unconditional save after the unit loop is consistent with existing semantics.

## Plan Deviations
No deviations from the plan. The single guarded `p.store.Save()` call was added exactly as specified in Phase 1, and the unit test verifies in-memory store state matches persisted state after `RunOnce()`.

## Architecture Decision Records
### Write-Through Persistence in RunOnce

context:
The pipeline loop processes units in `RunOnce()` and previously only persisted processing history to `storage.json` via `captureProcessingHistory()` (on `.done` file matches) or on shutdown. This left a window where a crash or stop after processing a unit but before the next pass would lose that unit's data.

options:
- Add `p.store.Save()` at the end of `RunOnce()` after the unit loop (write-through per pass).
- Rely only on existing `captureProcessingHistory()` saves and shutdown persistence (no change).
- Add periodic saves in the `Run()` loop after each `RunOnce()` return.

decision:
Add a guarded `if p.store != nil { p.store.Save() }` call at the end of `RunOnce()`, after the unit processing loop and before `p.emit(EventScanDone, 0, nil)`.

rationale:
Write-through persistence after each pass directly closes the data-loss window with minimal code change. The store file is small (processed script records only), so per-pass writes are fast and acceptable. This matches the recommended pattern from the research findings.

consequences:
- Every `RunOnce()` pass results in a disk write to `storage.json`, even when no units were processed.
- If `Save()` fails, the pipeline logs a warning and continues; the next pass retries persistence.

## Handoff Notes

### Known Limitations
- `storage.json` is written on every pass, adding disk writes proportional to the number of passes. For large numbers of passes with no units, this is unnecessary but the file is small so impact is minimal.
- `Save()` failures are logged as warnings but do not abort the pipeline; the in-memory state may diverge from disk until the next successful save.

### Data Models
- `jsonstore.Store`: in-memory key-value map (`data: map[string]any`) backed by a JSON file at `path`, guarded by `sync.RWMutex`, with `Set`, `Get`, `Delete`, `Save`, `Close` methods.

### API Surface
- `Store.Save() error`: flushes the current in-memory contents to the JSON file atomically via snapshot and `os.WriteFile`.

## Reference
- internal/app/pipeline/pipeline.go
- internal/infra/persistence/jsonstore/jsonstore.go
- internal/app/pipeline/pipeline_test.go
