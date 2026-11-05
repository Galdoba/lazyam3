# Storage Timing: Data Loss Risk in Pipeline Loop Plan

## Implementation Overview
- High-level approach: Implement write-through persistence by calling `store.Save()` at the end of each `RunOnce()` pass, ensuring processed unit data reaches `storage.json` immediately after processing, before the program can be stopped or crash.
- Key architectural decisions: The fix is a single guarded call to `p.store.Save()` after the unit processing loop in `RunOnce()`. The `jsonstore.Store.Save()` method is already thread-safe and persists the in-memory map to disk.
- Integration points: `internal/app/pipeline/pipeline.go` (RunOnce) calls into `internal/infra/persistence/jsonstore/jsonstore.go` (Save). No changes to the store implementation are required.
- Performance and scalability considerations: The `storage.json` file is small and contains only processed script records. Saving after each pass is acceptable and aligns with the write-through persistence pattern recommended in the research findings.

## Task Breakdown
Organized into logical phases with atomic tasks using checkboxes.

### Phase 1: Persist processed units after each pass
- [x] In `internal/app/pipeline/pipeline.go`, at the end of `RunOnce()` after the unit processing loop (line 220) and before `p.emit(EventScanDone, 0, nil)` (line 222), add a call to `p.store.Save()` guarded by `if p.store != nil { ... }`. This persists all processed unit data to `storage.json` after each pass, closing the data loss window.

## Code References
```
internal/app/pipeline/pipeline.go:152-224
internal/infra/persistence/jsonstore/jsonstore.go:106-125
```

## Testing Plan
- Unit tests: Add a test in `internal/app/pipeline/pipeline_test.go` that verifies `store.Save()` is invoked after `RunOnce()` processes units, ensuring in-memory store state matches persisted state.
- Integration tests: Use `scripts/live_test.sh` to run the pipeline, kill it after processing a unit, and verify the processed unit data is present in `storage.json`.
- End-to-end test scenarios: Run a full pipeline pass and confirm `storage.json` contains all processed scripts after each pass completes.
- Performance test requirements: Verify `store.Save()` performance after each unit is acceptable for the small JSON file size.
- Regression test considerations: Ensure `captureProcessingHistory` still correctly stores `.done` files from the previous cycle without duplication or data loss.

## FACTS Scale Output
F: 5  A: 5  C: 5  T: 5  S: 5  Mean: 5.00  --> PASS

## Dependencies & Sequencing
- Inter-task dependencies: Phase 1 has a single task with no dependencies.
- Critical path: The store save call is the only required change; no external system dependencies.
- External system dependencies: None. The fix relies only on the existing `jsonstore.Store.Save()` method.

## Risk Mitigation
- Technical risks: Calling `Save()` after every pass adds disk writes. Mitigation: JSON file is small, writes are fast and thread-safe via `sync.RWMutex`.
- Contingency plans: If `Save()` fails, the existing `captureProcessingHistory` logs a warning; the pipeline continues and retries on the next pass.
- Early warning indicators: Monitor for `store save failed` warnings in logs.
- Escalation criteria: If `Save()` returns persistent errors, escalate to investigate file permissions or disk issues.

## Rollback Strategy
- Rollback points: The change is a single guarded `p.store.Save()` call in `RunOnce()`.
- Rollback procedure: Revert the added lines in `internal/app/pipeline/pipeline.go:152-224` to restore the pre-fix behavior where data only persisted on shutdown or next pass.
- Feature flag configurations: Not applicable; the write-through behavior is unconditionally enabled as it is the correct semantics for a crash-resilient journal.
