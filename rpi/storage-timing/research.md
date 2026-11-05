# Storage Timing: Data Loss Risk in Pipeline Loop Research

## Problem Context

- **Restated problem**: While running continuous `lazyam3 process`, `.done` files are deleted and data is not in storage.json until the program is stopped - this is a risk of losing data.
- **Business intent**: `storage.json` must serve as a reliable journal of processed scripts, resilient to program crash.
- **Current behavior**:
  - `.done` files from the *previous* cycle are written to store at the start of the *current* cycle (`captureProcessingHistory`)
  - `.processing` files from the *current* cycle are **never** written to store during program runtime
  - `writeStatus` overwrites `storage.json` with status totals, corrupting store data
- **Desired behavior**: Data must reach `storage.json` immediately after each unit is processed, before the program can be stopped or crash, and status totals must not overwrite processing history.

## Affected Files

```
internal/app/pipeline/pipeline.go:226-255
internal/app/pipeline/pipeline.go:92-147
internal/app/pipeline/pipeline.go:152-224
internal/app/pipeline/pipeline.go:258-347
internal/infra/persistence/jsonstore/jsonstore.go:127-138
internal/app/app.go:90-104
internal/infra/commands/cmd_process.go:141-160
```

- `internal/app/pipeline/pipeline.go:226-255` - `captureProcessingHistory`: reads `.done` files and calls `store.Save()`, but only for data from the previous cycle
- `internal/app/pipeline/pipeline.go:92-147` - `Run()`: polling loop, `store.Save()` is not called between iterations
- `internal/app/pipeline/pipeline.go:152-224` - `RunOnce()`: single pass, returns without `store.Save()`
- `internal/app/pipeline/pipeline.go:258-347` - `processUnit()`: creates `.processing` artifact but does not write to store
- `internal/infra/persistence/jsonstore/jsonstore.go:127-138` - `Close()`: calls `Save()`, but only on shutdown
- `internal/app/app.go:90-104` - `Shutdown()`: calls `store.Close()`, but this is too late on crash
- `internal/infra/commands/cmd_process.go:141-160`: after `Run()`/`RunOnce()`, immediately calls `Shutdown()` - no intermediate save

## Code Examples

**Current flow (data loss window):**

```
RunOnce #1:
  captureProcessingHistory() -> store.Set(script_A) -> store.Save()  (data from cycle 0)
  processUnit() -> .processing "script_B" created               (not in store)
  return -> RunOnce done, data in memory, only cycle 0 on disk

[ Crash / kill here ]

RunOnce #2:
  captureProcessingHistory() -> .done "script_B" -> store.Set(script_B) -> Save()
  ... but data from cycle 1 is already lost
```

**Desired flow:**

```
RunOnce #1:
  captureProcessingHistory() -> store.Set(script_A) -> Save()
  processUnit() -> .processing "script_B"
  store.Save()  <- new: save everything after each pass
  return

[ Crash / kill here ] -> storage.json contains script_A and script_B
```

## External Research Findings

No external research needed - this is an internal codebase issue related to state persistence architecture.

## FAR Scale Output

### Factual Score: 5
### Actionable Score: 4
### Relevant Score: 5
### Mean: 4.67  --> PASS

- **F (5)**: All findings confirmed by code analysis - `store.Save()` is only called in `captureProcessingHistory` and `Close()`, no save between `Run()` iterations.
- **A (4)**: Solution is clear - add `store.Save()` at the end of `RunOnce()`. No ambiguity.
- **R (5)**: Direct impact on system reliability - data loss on crash.

## Testing Strategy

- **Unit test**: `pipeline_test.go` - add a test that simulates a crash after `processUnit` and verifies data is in `storage.json`
- **Integration test**: `scripts/live_test.sh` - run pipeline in a loop, kill after processing a unit, check `storage.json`
- **Observability**: Add log message on every `store.Save()` for debugging

## Potential Design Pattern Recommendations

- **Write-through persistence**: Save data to disk immediately after change, not in batches. Simple and reliable for this use case.
- **Atomic writes**: Use write-to-temp + rename to prevent corruption of `storage.json` on crash during write (partially implemented in `jsonstore.Save()` via marshal -> WriteFile, but not atomic).

## Assumptions

- `storage.json` is used as a journal of processed scripts, not as a cache
- Program crash is a valid scenario that must be handled
- `store.Save()` performance after each unit is acceptable (JSON file is small)

## Out of Scope

- Migration to a real database instead of JSON file
- Transactional storage (undo/rollback)
- Performance optimization of `store.Save()`
