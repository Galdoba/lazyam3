# Storage JSONL Appending Plan

## Implementation Overview
- High-level approach: Rewrite `jsonstore.Store` from an in-memory key-value map with full-file overwrite semantics to an append-only JSONL writer that encodes each `Set` value as a single JSON line via `json.Encoder.Encode`.
- Key architectural decisions: Store holds an `*os.File` opened in `O_CREATE|O_WRONLY|O_APPEND` mode plus a `sync.Mutex` to serialize writes. Drop `Get`/`Delete`/`Keys`/`Len`/`load` since reading is handled externally. `Save` becomes a no-op; `Close` closes and syncs the file.
- Integration points: `AppModules.Store()` returns `*jsonstore.Store` unchanged; pipeline `captureProcessingHistory` calls `Set` per unit and must drop the `Save` call; `AppModules.Shutdown()` calls `Close()` which now closes the file.
- Performance and scalability: Per-unit history writes are low-volume; a mutex around `json.Encoder.Encode` is sufficient. `O_APPEND` guarantees atomic per-line writes at the OS level for small payloads.

## Task Breakdown

### Phase 1: Rewrite jsonstore.go to JSONL append semantics
- [x] Replace the `Store` struct in `internal/infra/persistence/jsonstore/jsonstore.go` to hold `path string`, `mu sync.Mutex`, `file *os.File`, and `logger *logging.Logger`; remove `data map[string]any`, `closed bool`. Update package doc comment to reflect JSONL append semantics.
- [x] Rewrite `New(path, logger)` to open the file with `os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)`, return `nil, err` on failure, and return `&Store{path, file, logger}` on success; remove the `load()` method entirely.
- [x] Rewrite `Set(key string, value any) error` to acquire the mutex, create `json.NewEncoder(s.file)`, call `Encode(value)`, and return errors; drop the `closed` check. The `key` parameter is retained for signature compatibility but not used for indexing.
- [x] Remove `Get`, `Delete`, `Keys`, `Len`, and `Save` methods; update `Close()` to close the file, set a closed flag, call `Sync()`, and log; make `Close()` idempotent.

### Phase 2: Update pipeline captureProcessingHistory
- [x] In `internal/app/pipeline/pipeline.go:226-256`, remove the `p.store.Save()` call and its associated logging block; keep the `store.Set` loop, error handling, and abort logic intact.

### Phase 3: Update pipeline tests for JSONL semantics
- [x] Update `internal/app/pipeline/pipeline_test.go:1337-1377` (history test): replace `store.Get("script_test")` and payload assertions with reading `storePath` line-by-line, parsing each line as JSON, and verifying the payload contains `script_name: test`; replace `store.Close()` with the new semantics.
- [x] Update `internal/app/pipeline/pipeline_test.go:1393-1419` (store failure test): keep the pre-`Close()` that marks the store closed; simulate store failure by closing the store before RunOnce so `Set` returns "store is closed" and aborts the transaction; verify the done file persists.
- [x] Update `internal/app/pipeline/pipeline_test.go:1421-1468` (storage after RunOnce test): replace `os.ReadFile` + `json.Unmarshal` into a map with line-by-line JSONL parsing and verify a line contains `script_name: test`; replace `store.Close()`.

## Code References
```
internal/infra/persistence/jsonstore/jsonstore.go
internal/app/pipeline/pipeline.go:226-256
internal/app/pipeline/pipeline_test.go:1337-1377;1379-1419;1421-1468
internal/app/app.go:76-85;87-104
```

## Testing Plan
- Unit tests: verify `Set` appends valid JSONL lines (read file back and parse each line as JSON); verify `New` opens in append mode and does not overwrite existing lines; verify concurrency produces well-formed lines under the mutex.
- Integration surface: run `scripts/live_test.sh` to verify end-to-end pipeline still executes and the store append works within the pipeline flow.
- Observability: ensure `logger.Info("script executed")` and `logger.Warn("store set failed")` messages remain intact in `captureProcessingHistory`.
- Repro steps: create a store, `Set` several payloads, read file line-by-line, assert each line parses as valid JSON and matches payload.

## FACTS Scale Output
F: 5  A: 4  C: 5  T: 4  S: 4  Mean: 4.40  --> PASS

## Dependencies & Sequencing
- Critical path: Phase 1 (jsonstore.go rewrite) must complete before Phase 2 (pipeline.go) and Phase 3 (tests) can proceed, since the store API changes.
- Phase 2 and Phase 3 are sequential within themselves; Phase 3 depends on Phase 1 and Phase 2 being complete.
- External system dependencies: Go standard library `os`, `encoding/json`, `sync`; no external packages required.

## Risk Mitigation
- Risk: `json.Encoder.Encode` is not thread-safe; mitigation is the `sync.Mutex` around the `Encode` call.
- Risk: Buffered data loss on process crash; mitigation is `Close()` calling `file.Sync()` before `file.Close()`.
- Risk: Existing tests assert key-value semantics; mitigation is rewriting the three affected test functions to verify JSONL line parsing.
- Early warning indicator: build failures in jsonstore.go or pipeline.go indicate signature or import mismatches.
- Escalation criteria: if JSONL parsing in tests fails to match payloads, re-verify the payload structure passed to `Set` in pipeline.go.

## Rollback Strategy
- Phase 1 rollback: restore the original jsonstore.go from git; the Store API (Get/Set/Delete/Keys/Len/Save/Close) is preserved in signature shape for Set/Close, but reading methods are removed.
- Phase 2 rollback: restore the pipeline captureProcessingHistory to re-add the store.Save() call.
- Phase 3 rollback: restore pipeline_test.go to the pre-JSONL assertions.
- Feature flag: no feature flag needed; the change is a storage backend swap with no runtime toggle.
- Deployment rollback: since this is a CLI tool with no running service, rollback is a git revert to the pre-implementation commit.
