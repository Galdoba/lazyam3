# Storage JSONL Appending Research

## Problem Context
- Restated problem: Explore changing the storage handling in lazyam3 from a key-value JSON store (in-memory map flushed to a JSON file on Save/Close) to a JSONL (JSON Lines) file that appends records.
- Business/functional intent: The pipeline uses the store as a processing history log: it records `script_<name>` payloads after each script execution. The current store loads the whole JSON file into memory, does Set/Save (full overwrite), and Close. The desired shape appends each new record as a JSONL line, and reading is delegated to other means (out of scope here).
- Current behavior: `jsonstore.Store` keeps `map[string]any` in memory, `Set` updates the map, `Save`/`Close` marshal the whole map to a JSON file (`os.WriteFile`), `New`/`load` read the file back on startup.
- Desired behavior: `Set` appends a JSON-encoded line + newline to the file in append mode; no in-memory map, no full-file reload, no `Save`/`Close` overwrite semantics. `Get`/`Delete`/`Keys`/`Len` are no longer needed.
- Constraints: Storage must be safe for the pipeline's usage (called per-unit in a loop, potentially across multiple units). Reading is out of scope. Infrastructure failures must fail fast per project rules.

## Affected Files
```
internal/infra/persistence/jsonstore/jsonstore.go
internal/app/app.go:76-85
internal/app/pipeline/dto.go:28
internal/app/pipeline/pipeline.go:226-256
internal/app/pipeline/pipeline_test.go:1337-1437
```
- `internal/infra/persistence/jsonstore/jsonstore.go`: The core store implementation. Current design is a key-value map with load/save semantics. New design: append-only JSONL writer.
- `internal/app/app.go:76-85`: `Store()` initializes `jsonstore.New(path, logger)`. Must remain compatible (returns `*jsonstore.Store`).
- `internal/app/pipeline/dto.go:28`: `Store *jsonstore.Store` field in `Options`. Must remain compatible.
- `internal/app/pipeline/pipeline.go:226-256`: `captureProcessingHistory` calls `store.Set` then `store.Save`. New JSONL store would drop `Save` (or make it a no-op/flush).
- `internal/app/pipeline/pipeline_test.go:1337-1437`: Tests that create a store, `Set`, `Get`, and assert values. These assert the key-value semantics; JSONL append semantics differ and reading is out of scope.

## Code Examples

Current key-value store (overwrite semantics):
```go
// jsonstore.go (current)
type Store struct {
    path   string
    mu     sync.RWMutex
    data   map[string]any
    closed bool
    logger *logging.Logger
}

func (s *Store) Set(key string, value any) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.data[key] = value
    return nil
}

func (s *Store) Save() error {
    raw, err := json.MarshalIndent(s.data, "", "  ")
    os.WriteFile(s.path, raw, 0o644)
}
```

Proposed JSONL append store:
```go
// jsonstore.go (proposed)
type Store struct {
    path   string
    mu     sync.Mutex
    file   *os.File
    logger *logging.Logger
}

func New(path string, logger *logging.Logger) (*Store, error) {
    f, err := os.Open(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
    if err != nil { return nil, err }
    return &Store{path: path, file: f, logger: logger}, nil
}

func (s *Store) Set(key string, value any) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    enc := json.NewEncoder(s.file)
    if err := enc.Encode(value); err != nil { // writes JSON + '\n'
        return err
    }
    return nil
}
```

Note: `json.Encoder.Encode` writes the JSON value followed by a newline, so each `Set` produces one JSONL line. `Save`/`Close` semantics change: `Close` would `file.Close()` and `Save` becomes a no-op or a `file.Sync()`.

## External Research Findings
- `json.Encoder.Encode` writes a single JSON value followed by a newline to the underlying writer: https://golang.org/std/current/doc/json/#Encoder.Encode
- Appending to a file in Go: use `os.Open(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)`: https://golang.org/std/current/doc/os/#Open
- Concurrency: `json.Encoder.Encode` is not safe for concurrent calls; serialize writes with a mutex or a single writer goroutine. `sync.Mutex` around the `Encode` call is sufficient for append-only JSONL.
- Performance: `json.Encoder` writes directly to the file; for high throughput, buffer lines (e.g. `bufio.Writer` wrapping the file, or batch `Encode` calls under one lock) and `Sync` periodically. For processing-history logging, per-unit writes are low-volume, so a mutex + direct `Encode` is adequate.
- Gotchas: `json.Encoder.Encode` issues a single write of the JSON bytes + newline; with O_APPEND this is atomic at the OS level per-line for small payloads, so partial lines do not occur from `os.File.Write` (it loops until all bytes written). On process crash, buffered data (if using `bufio.Writer`) is lost: flush/sync after writes. Do not use `json.Encoder.SetIndent`: it produces multi-line pretty JSON, breaking the one-object-per-line JSONL contract. `Encode` always appends `\n`; the file must end with a newline before the first append (or rely on O_APPEND to an existing valid JSONL file). Embedded newlines inside string values are escaped by `json.Encoder`, keeping lines intact.

## FAR Scale Output
### Factual Score: 5
Evidence is concrete: current `jsonstore.go` implements key-value semantics with load/save; pipeline calls `Set`+`Save`; `json.Encoder.Encode` appends a newline per Go docs; `os.Open` with O_APPEND is the standard append idiom.

### Actionable Score: 4
Clear stepwise plan: rewrite `Store` to open file in append mode, `Set` uses `json.NewEncoder(file).Encode(value)` under a mutex, drop `Save`/`Close` overwrite semantics, update tests. Validation path: build + `live_test.sh`.

### Relevant Score: 4
Core to the storage subsystem; the processing history store is used by the pipeline on every unit. Aligns with the stated feature goal.

### Mean: 4.33  --> PASS

## Testing Strategy
- Unit tests: verify `Set` appends valid JSONL lines (read file back and parse each line as JSON); verify `New` opens in append mode and does not overwrite existing lines; verify concurrency (multiple `Set` calls under a mutex produce well-formed lines).
- Integration surface: run `scripts/live_test.sh` to verify end-to-end pipeline still executes and the store append works within the pipeline flow.
- Observability: ensure `logger.Debug("processing history stored")` and `logger.Warn("store set failed")` messages remain or are adapted to JSONL semantics.
- Repro steps: create a store, `Set` several payloads, read file line-by-line, assert each line parses as valid JSON and matches payload.

## Potential Design Pattern Recommendations
- Append-only writer pattern: open file once in append mode, hold a `sync.Mutex`, use `json.Encoder` per record. Internal exemplar: `internal/infra/logging/logger.go` uses structured logging with file backing.
- Single-writer goroutine pattern: if volume grows, route `Set` calls to a channel handled by one writer goroutine that batches and `Sync`s periodically.
- Graceful shutdown: `Close()` closes the file and calls `Sync()` once; `Save()` becomes a no-op or `Sync()`.

## Assumptions
- Reading the JSONL file is handled by external means (scripts, other tools); the store no longer needs `Get`/`Delete`/`Keys`/`Len`/`load`.
- Payloads are `any` serializable to JSON (currently `map[string]any` payloads).
- Concurrency is limited to the pipeline's per-unit loop; a `sync.Mutex` is sufficient (no cross-process locking needed since it is a single lazyam3 process writing history).
- Existing JSONL consumers (if any downstream tool exists) expect one JSON object per line with a trailing newline: `json.Encoder.Encode` provides this.

## Out of Scope
- Implementing readers for the JSONL file (reading is out of scope per user request).
- Backward compatibility with the in-memory key-value semantics (Get/Delete/Keys/Len).
- Cross-process file locking (single-process writer).
- Migration of existing JSON store files to JSONL format.
