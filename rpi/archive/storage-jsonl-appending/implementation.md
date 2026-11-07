# Storage JSONL Appending Implementation

## Summary
Rewrote jsonstore.Store from an in-memory key-value store with full-file overwrite semantics to an append-only JSONL writer. Each Set call appends a single JSON line via json.Encoder.Encode. The store holds an *os.File opened in O_CREATE|O_WRONLY|O_APPEND mode with a sync.Mutex to serialize writes. Get/Delete/Keys/Len/Save methods were removed; Close now syncs and closes the file. Pipeline captureProcessingHistory drops the Save call. Tests updated to verify JSONL line parsing. Build, tests, and jsonstore lint pass. Live pipeline test executes end-to-end successfully.

## Pull Requests
| PR | Description | Status |
|----|-------------|--------|
| [#1](https://github.com/Galdoba/lazyam3/pulls/new?title=Storage+JSONL+Appending) | Storage JSONL appending implementation | To be opened by human after branch push |

## Unexpected Discoveries
Go 1.26+ uses os.OpenFile (not os.Open) for opening files with flags and mode, returning *os.File. The store.Close() approach properly simulates store failure in tests: after Close, Set returns "store is closed" error, which aborts the transaction.

## Plan Deviations
Phase 3 store failure test: plan suggested passing an invalid path to jsonstore.New to simulate failure, but that results in a nil store which bypasses captureProcessingHistory without aborting. Kept the pre-Store.Close() approach so Set fails with "store is closed" and the transaction aborts, preserving the done file as intended.

## Architecture Decision Records
### JSONL Append Semantics

context:
The jsonstore.Store previously held an in-memory map and flushed to a JSON file on Save/Close. The pipeline only needs to record processing history per unit, not read or update existing entries.

options:
- Keep in-memory map with full-file overwrite (original approach)
- Switch to append-only JSONL where each Set appends one line
- Use a separate log file per unit

decision:
Switch to append-only JSONL where each Set appends a single JSON line via json.Encoder.Encode.

rationale:
Per-unit history writes are low-volume and the pipeline never reads back from the store during execution. JSONL append semantics avoid full-file rewrites and guarantee atomic per-line writes at the OS level for small payloads via O_APPEND.

consequences:
- Get/Delete/Keys/Len/Save methods removed from the Store API
- Reading is handled externally by consumers
- Close must sync the file before closing to avoid data loss on process crash

### Thread-Safe Encoding

context:
json.Encoder.Encode writes to the underlying file; concurrent Set calls could interleave lines.

options:
- Rely on O_APPEND atomicity without a mutex
- Use sync.Mutex around the Encode call
- Use sync.Mutex per-key

decision:
Use a sync.Mutex around the json.NewEncoder(s.file).Encode(value) call.

rationale:
json.Encoder.Encode is not thread-safe for concurrent calls; the mutex serializes writes and guarantees well-formed JSONL lines.

consequences:
- All Set calls are serialized, which is acceptable for low-volume history writes

## Handoff Notes

### Known Limitations
- The store no longer supports reading back entries; consumers must parse the JSONL file externally.
- The key parameter in Set is retained for signature compatibility but not used.

### Data Models
- Store: path string, mu sync.Mutex, file *os.File, logger *logging.Logger

### API Surface
- SET key string, value any: appends value as a single JSON line; key ignored
- CLOSE: syncs and closes the file; idempotent

## Reference
- internal/infra/persistence/jsonstore/jsonstore.go
- internal/app/pipeline/pipeline.go
- internal/app/pipeline/pipeline_test.go
- internal/app/app.go
