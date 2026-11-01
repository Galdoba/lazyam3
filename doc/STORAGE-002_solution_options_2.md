---
problem_id: STORAGE-002
title: persist .done processing history to jsonstore before deletion
generated_at: 2026-10-30T12:00:00Z
verdict: APPROVED
---

## approaches:

### Approach 1: Direct Store Wiring in Pipeline

Add `Store` field to `pipeline.Options` (dto.go:23-47). In `RunOnce` deletion loop (pipeline.go:170-174), before `os.Remove`, call `Store.Set(key, payload)` where key = "script_" + scriptName. After all deletions, call `Store.Save()` before `EventScanDone` (pipeline.go:196). Wire `modules.Store()` into Options in `cmd_process.go`.

**Pros:**
- Minimal changes: 1 field in Options, 2 call sites in Pipeline
- Reuses existing jsonstore infrastructure (lazy init via `AppModules.Store()`)
- Respects domain purity: Store is infra, accessed via Options (not domain)
- Graceful degradation: empty store path → nil store → no-op, `.done` files deleted normally
- Existing tests pass unchanged

**Cons:**
- Adds jsonstore dependency to pipeline layer (acceptable: `app.go` already imports infra packages)
- In-memory payload before `Save()` — crash between Set and Save loses entries (mitigated by atomic-ish write in `Save()`)

**Effort:** 1-2 hours

### Approach 2: Encapsulated Store Helper

Same as Approach 1, but extract store-write logic into a `storeDone(match *doneMatch) bool` method on Pipeline that returns success/failure, keeping the `RunOnce` loop clean.

**Pros:**
- Cleaner Pipeline loop, isolates store logic for unit testing
- Easier to add fallback logging later

**Cons:**
- Slightly more indirection
- Marginal complexity gain for a single call site

**Effort:** 1-2 hours

## verdict: APPROVED

**Rationale:** The problem is fully solvable within existing architecture. Approach 1 is the simplest path — it follows the existing `statusPath` wiring pattern (status.json written after each pass). The Store is infra, accessed via Options (not domain), respecting purity rules. The transactional semantics (store write succeeds → delete; fails → keep `.done`) are straightforward. The empty/missing store path is handled gracefully since `p.store` would be nil.

Approach 2 offers cleaner encapsulation but adds marginal complexity for a single call site. Consider if the deletion loop grows to multiple call sites — then refactor to Approach 2.

## open_questions:

- Should store write happen in the same goroutine (blocking) or async (fire-and-forget)? → Blocking (in deletion loop) is simplest and satisfies transactional requirement.
- Should the store entry include the full `.done` payload or a subset? → Full payload (`map[string]any`), no subset needed.
- [FUTURE] Fallback mechanism: when store write fails, log ERROR-level entry with payload for manual recovery? → Currently logs Warn on Set error; ERROR-level fallback is future work.
- [FUTURE] Transaction scope: should store write + `.done` deletion be wrapped in a single transaction block? → Single-threaded pipeline; in-memory Set then `os.Remove` is sufficient, no DB transaction needed.
