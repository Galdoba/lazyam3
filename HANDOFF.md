# HANDOFF

## Summary
Completed STORAGE-002: persist `.done` processing history to jsonstore before deletion, with transactional semantics. A successful store `Set` gates the `.done` deletion; a store failure aborts the transaction so `.done` files survive for the next scan cycle.

## What was done
- **dto.go**: Added `Store *jsonstore.Store` field to `Options` under the "Infra" section, plus the `jsonstore` import.
- **pipeline.go**: Added `store *jsonstore.Store` to `Pipeline` struct; copied it from `opts.Store` in `NewPipeline`; changed `findDoneFile` signature from `(string, bool)` to `(string, map[string]any, bool)`; updated `doneMatch` struct to include `payload map[string]any`; implemented transaction logic via `captureProcessingHistory` helper method.
- **cmd_process.go**: Fetched `store, err := modules.Store()` after metadata loading, and passed `Store: store` into `pipeline.NewPipeline` options.
- **pipeline_test.go**: Added `jsonstore` import and three tests: `TestPipeline_StoreHistoryOnSuccess`, `TestPipeline_StoreFailurePersistsDone`, `TestPipeline_StorageJsonAfterRunOnce`.

## Verification
- `go build ./...` — passed.
- `go test -race -cover ./...` — passed (pipeline coverage 81.6%).
- `go vet ./...` — passed.
- `golangci-lint run ./...` — passed (only pre-existing unrelated `forbidigo` issue in `internal/domain/callname/shared.go:9:2`, which uses `time.Sleep`).

## Notes
- Transactional semantics: store `Set` failure aborts the transaction, so `.done` files persist for the next scan cycle.
- `findDoneFile` change is a signature change; only one call site exists in `RunOnce`.
- `payload["script_name"]` is guaranteed present by the artifact writer.
