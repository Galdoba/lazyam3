---
problem_id: METADATA-001
title: hot-reload metadata.json during long-running pipeline
generated_at: 2025-11-03T00:00:00Z
verdict: APPROVED
---

## approaches:

### Approach 1: Self-contained reload in Repository

**Description:** Add `metadataPath` and `lastMod` fields to `Repository`. Method `Reload()` stats the file, compares mtime with `lastMod`, on change calls `LoadRoot()` and replaces `root` atomically under write-lock. Call `Reload()` from pipeline before each `RunOnce()` in the `Run()` loop.

**Files:**
- `internal/infra/metadata/repository.go` - add fields, accept path in `New()`, add `Reload()` method
- `internal/infra/metadata/repository_test.go` - reload tests, failure fallback, race detector
- `internal/infra/commands/cmd_process.go:85` - pass path to `metadata.New()`
- `internal/app/pipeline/pipeline.go:100` - call `Reload()` before `RunOnce()` in `Run()` loop

**Pros:**
- Minimal file count (4 files changed)
- Everything encapsulated in Repository - caller is unaware of hot-reload
- `Reload()` returns `(bool, error)` - pipeline logs only on reload event (INFO with old/new product counts)
- Full control over locking: write-lock on swap, read-lock on FindByID (already in place)
- No new dependencies, no interface changes

**Cons:**
- Repository knows the file path (infra-detail in infra layer - acceptable, this is infra code)
- Pipeline must have concrete `*metadata.Repository` type instead of interface (not breaking, pipeline already works with concrete type)

**Effort:** 2-3 hours

### Approach 2: Decorator / wrapper with hot-reload

**Description:** Create a new type `HotReloadRepository` that wraps `metadata.ProductRepository`. It stores path, mtime, and delegates FindByID to the wrapped repository. On `Reload()` swaps the inner repository.

**Files:**
- `internal/infra/metadata/hot_reload.go` - new file, new type
- `cmd_process.go:85` - wrapper around `metadata.New()`
- `pipeline.go:100` - call `Reload()` on the wrapper

**Pros:**
- Domain interface `ProductRepository` does not change
- Clean composition - reload logic is separated from storage logic
- Can swap wrapper for plain Repository in `--once` mode without changes

**Cons:**
- +1 type, +1 file, more code for the same result
- Pipeline must know about HotReloadRepository (wiring change)
- Double indirection: `FindByID` -> wrapper -> inner repo (negligible perf, extra layer)
- Logger must be passed to wrapper for reload event logging

**Effort:** 3-4 hours

### Approach 3: Pipeline-level stat-only (no Repository changes)

**Description:** Store path and mtime in Pipeline. Before `RunOnce()`, do `os.Stat` - if mtime changed, call `metadata.LoadRoot()` directly, create new Repository, and replace `p.md` via setter or unsafe.

**Pros:**
- Zero changes to Repository (minimal impact)

**Cons:**
- `p.md` has interface type `ProductRepository` - cannot replace instance without setter or unsafe
- `LoadRoot()` returns `*Root`, not `*Repository` - need to call `metadata.New()` entirely
- If using a setter on Pipeline, adds mutation point
- Violates encapsulation: Pipeline knows about `LoadRoot()` and file path
- Race condition risk if pipeline becomes multi-threaded (interface value replacement is not atomic)

**Effort:** 2 hours (high risk due to architectural issues)

## verdict: APPROVED

**Rationale:**

Approach 1 (self-contained reload) provides the best balance of simplicity and reliability:
- Changes exactly 4 files, all in the infrastructure layer
- Repository already has `mu` (RWMutex) and `logger` - ideal primitives for safe reload
- `Reload()` returns bool (changed?) - pipeline logs INFO only on reload event, not on every stat
- Domain interface `ProductRepository` does not change - FindByID remains the only public method
- Pipeline receives concrete `*metadata.Repository` (already done in cmd_process.go:85 -> pipeline.Options.Metadata)

Approach 2 (decorator) is a valid alternative if hot-reload is needed for other entities in the future (e.g., config files). But for a single use-case, it is over-engineering.

Approach 3 (pipeline-level) is rejected due to architectural issues: impossible to safely replace interface value, requires knowledge of `LoadRoot()` in pipeline layer, violates separation of concerns.

## open_questions:
- Reload event logging: INFO on reload with product counts (old/new) - confirmed, useful for monitoring
- Trigger timing: reload BEFORE scan pass (ensures fresh data for FindByID in this pass) - follows PS desideratum
- Size vs mtime: use mtime (more reliable for external updates, as noted in PS open_questions)
- Should `Reload()` return `(bool, error)` or just `error`? Bool indicates whether reload actually happened (useful for conditional logging)
