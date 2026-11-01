---
problem_id: STORAGE-002
title: persist .done processing history to jsonstore before deletion
generated_at: 2026-10-30T12:00:00Z
verdict: APPROVED
---

## approaches:

### Approach 1: Direct Store Wiring in Pipeline (Recommended)

**Description:**
Add `Store` field to `pipeline.Options` struct. In `processUnit()` method:
1. Before writing `.processing` artifact, call `opts.Store.Set(key, payload)` where key = "script_" + scriptName
2. In `RunOnce()`, before emitting `EventScanDone`, call `opts.Store.Save()` to persist all entries
3. Wrap Store operations in error handling: if Set fails, skip .done deletion
4. Wire `modules.Store()` into pipeline.Options in `cmd_process.go`

The Store is already wired via `AppModules` with lazy initialization (sync.Once) and error handling. The pipeline accesses it through Options, maintaining domain purity.

**Pros:**
- Minimal code changes (add 1 field to Options, 2 method calls)
- Reuses existing jsonstore infrastructure already wired in AppModules
- Maintains architecture: Store (infra) accessed via Options (not domain)
- Transactional: Store.Save() called after all .done processing, before EventScanDone
- Graceful degradation: empty store path - no-op, .done files deleted normally
- Thread-safe: Store uses RWMutex; pipeline runs single-threaded in Run/RunOnce

**Cons:**
- Store write blocking: if Store.Set blocks (e.g., disk full), .done deletion waits
- Pending .done files if store fails: mitigated by retrying each cycle

**Effort:** 2-3 hours

### Approach 2: Async Store Writes (Fire-and-Forget)

**Description:**
Similar to Approach 1, but Store.Set is called in a separate goroutine without blocking:
1. In `processUnit()`, spawn goroutine that calls `opts.Store.Set(key, payload)`
2. Fire-and-forget; don't wait for completion
3. Store.Save() still called at end of RunOnce

**Pros:**
- Non-blocking pipeline execution
- Better throughput for large batches

**Cons:**
- Race condition: Store.Save() might not see entries written by async goroutines
- Complexity: need to track which entries are "in-flight"
- Loss of atomicity: Store write can succeed while .done deletion fails (opposite of desired behavior)
- Higher memory usage: entries accumulate until Save()

**Effort:** 4-6 hours

### Approach 3: Hybrid Blocking on Batch

**Description:**
Collect all .done files in a pass, then batch Store operations:
1. Scan all .done files first (without deletion)
2. Build batch of Store.Set calls
3. Call Store.Save() with all entries
4. If Save() succeeds, delete all .done files; else retry next cycle

**Pros:**
- Single transaction: all-or-nothing for the entire pass
- Reduced I/O: one Save() call instead of one per unit
- Better atomicity guarantee

**Cons:**
- More complex state management: track which .done files are pending
- Memory usage: hold all .done paths until Save() succeeds
- Slower: must wait for all units before deleting any .done file
- Blocks pipeline longer: no incremental progress if store fails mid-pass

**Effort:** 6-8 hours

## verdict: APPROVED

**Rationale:**
Approach 1 is the clear winner for this problem:
- **Simplicity**: Minimal changes, easy to understand and maintain
- **Correctness**: Matches the problem statement exactly - Store is wired via Options, Save() called at end of RunOnce
- **Architecture**: Respects domain purity rules (Store is infra, accessed via Options)
- **Existing infrastructure**: Leverages AppModules.Store() which already handles lazy initialization and error propagation
- **Trade-offs acceptable**: Blocking Store.Set is fine since:
  - Pipeline runs single-threaded (no goroutine contention)
  - Store operations are fast (JSON file, mutex-protected)
  - Failure is handled gracefully (retry next cycle)

Approach 2 would introduce race conditions that don't align with the transactional requirement.
Approach 3 is over-engineered for the scope - batching isn't needed for correctness.

## open_questions:
- None - all acceptance criteria can be met with Approach 1
  - Criterion 1 (Store.Set before delete): Handled in processUnit()
  - Criterion 2 (Options.Store field): Add to Options struct
  - Criterion 3 (failure keeps .done): Wrap Store.Set in error check
  - Criterion 4 (Store.Save() before EventScanDone): Call in RunOnce()
  - Criterion 5 (existing tests pass): No breaking changes
