---
problem_id: METADATA-001
title: hot-reload metadata.json during long-running pipeline
repository: github.com/Galdoba/lazyam3
baseline_commit: ce62a63df30839a61b034c783f4c89be3eb96fa1
problem_type: bug
status: CONFIRMED
---

## current_behavior: |
Pipeline loads metadata.json exactly once at startup via metadata.New() (cmd_process.go:85),
which calls LoadRoot() to read and parse the file (types.go:107-120). The parsed Root struct
is stored in-memory in Repository.root (repository.go:15).

For long-running pipeline (Run() loop, pipeline.go:87-134), each scan pass calls RunOnce() which
resolves products via FindByID() (repository.go:33-49). FindByID reads exclusively from the
in-memory Repository.root snapshot -- it never re-reads the file.

When metadata.json is updated externally (e.g., new episodes added to catalog, metadata refreshed
by upstream system), the running pipeline continues to use the stale snapshot from startup.

## desired_behavior: |
Pipeline detects metadata.json changes between scan passes and reloads it transparently.

Two acceptable approaches (design phase):
  A) On each RunOnce() pass, stat the file and compare mtime/size against last known state;
     reload on change.
  B) Periodic background reload (e.g., every N seconds or every M passes) to avoid stat overhead
     in tight poll loops.

On reload, the new Root replaces the old one atomically (Repository.mu.RWMutex protects root field).
Copy of old metadata.json MUST be preserved.
If reload fails (file missing, parse error), the pipeline logs a warning and falls back to the
previous snapshot for that pass -- never returning "product not found" errors for products that
were previously valid.

## rationale: |
metadata.json is the product catalog that drives the entire pipeline -- it determines which series,
seasons, episodes, and movies get processed. The upstream system that produces metadata.json is
external and may update it at any time (new episodes, corrected titles, added movies).

Without hot-reload, the pipeline silently processes with stale catalog data: new episodes in the
source directory may be misidentified or skipped entirely because their IDs don't match the cached
snapshot. Users with long-running pipelines (no --once flag) are especially affected -- they may
run for hours or days and never see catalog updates.

## evidence:
  - type: code_review
    reference: internal/infra/commands/cmd_process.go:85 -- metadata.New() called once, result reused
  - type: code_review
    reference: internal/infra/metadata/repository.go:15 -- Repository.root stores in-memory snapshot
  - type: code_review
    reference: internal/infra/metadata/repository.go:33-49 -- FindByID reads only from in-memory root
  - type: code_review
    reference: internal/infra/metadata/types.go:107-120 -- LoadRoot reads file once at New() time
  - type: code_review
    reference: internal/app/pipeline/pipeline.go:87-134 -- Run() loop calls RunOnce() repeatedly, reusing same md reference
  - type: code_review
    reference: internal/app/pipeline/pipeline.go:67 -- md field set from Options.Metadata at construction

## acceptance_criteria:
  - criterion: Pipeline reloads metadata.json when file modification time changes between scan passes
    verification_method: unit test creates two versions of metadata.json with different mtimes, verifies reload triggers
  - criterion: On reload failure (missing file, corrupt JSON), pipeline logs warning and continues with previous snapshot
    verification_method: unit test corrupts metadata.json mid-run, verifies pipeline does not panic and uses prior data
    verification_name: unit test corrupts metadata.json mid-run, verifies pipeline does not panic and uses prior data
  - criterion: Reload is thread-safe (atomic swap under write lock, reads under read lock)
    verification_method: race detector pass (go test -race)
  - criterion: --once mode unaffected (single pass, no reload opportunity)
    verification_method: existing --once tests continue to pass
  - criterion: Existing tests continue to pass (go test -race -cover ./...)
    verification_method: go test -race -cover ./...

## non_goals:
  - FSEvents / inotify-based filesystem watches (Linux inotify, macOS FSEvents, Windows ReadDirectoryChangesW)
  - Configurable reload interval (default: on each pass for correctness; interval-based is future work)
  - Partial/incremental updates to metadata (reload always reads full file)
  - Caching layer for FindByID lookups (tree search is O(n) where n is catalog size, fast enough)
  - Migration from current Root structure to a different data model

## constraints:
  - Repository.mu.RWMutex must protect all root field access (already in place, repository.go:14)
  - LoadRoot() signature unchanged (return *Root, error; infrastructure layer)
  - Pipeline.Options.Metadata field type unchanged (metadata.ProductRepository interface)
  - Backward compatible: existing --once workflows unaffected
  - No new external dependencies

## risks:
  - File stat/read race during reload (mitigated: write lock during swap, read lock during FindByID)
  - Metadata file permission change (mitigated: log warning, fall back to snapshot)
  - Large metadata file re-read on each pass (mitigated: stat only, full read only on mtime change; catalog ~690 series + 981 movies)
  - Concurrent pipeline passes (mitigated: pipeline is single-threaded, no goroutines in Run/RunOnce)

## open_questions:
  - Should reload happen before or after the scan pass? (before ensures fresh data for lookup; after avoids stale start-of-pass state)
  - Should we log the reload event (INFO level with old/new product counts)?
  - Should reload be triggered on file size change only, or also on mtime? (mtime is more reliable for external updates)
  - [FUTURE] Configurable reload strategy: stat-on-each-pass vs. periodic background reload
