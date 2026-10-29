---
problem_id: STORAGE-002
title: persist .done processing history to jsonstore before deletion
repository: github.com/Galdoba/lazyam3
baseline_commit: 0033e0b9558578df7285348cce0d9256eba24498
problem_type: feature
status: CONFIRMED
---

## current_behavior: |
Pipeline scans for empty directories with matching .done artifacts (pipeline.go:155-163).
When found, it deletes the source directory and then removes the .done file (pipeline.go:165-175).
The .done file contains processing metadata: source_dir, unit_id, script_name, product_name,
created_at, initial_files, created_files (enriched by STORAGE-001).

The jsonstore package exists at internal/infra/persistence/jsonstore/jsonstore.go with
Get/Set/Delete/Save/Close methods, and is wired into AppModules (app.go:76-84), but is
never used for any business logic. Config.Storage.Path defaults to "/path/to/lazyam3/storage.json"
(config.go:167-168).

After .done deletion, the only records are the generated .sh script (in archive) and
noisy logs -- no way to query "how was this episode processed?" or "were subtitles burned?"

## desired_behavior: |
Store write + .done deletion executed as one transactional operation: both succeed or both fail.
Before deleting a .done file, the pipeline reads its JSON payload and stores it in jsonstore
with key "script_{script_name}" (script_name contains timestamp + project name, unique per run).
The Store is wired through Pipeline.Options. After all .done files are processed, Store.Save()
persists the data to disk (already handled by AppModules.Shutdown() -> Store.Close() -> Save()).

If store write succeeds: .done file is deleted.
If store write fails: transaction aborts, .done file remains for retry next cycle.

Pipeline handles empty/missing Store path gracefully: if cfg.Storage.Path is empty or store
initialization fails, store operations are no-ops and .done files are deleted normally.

## rationale: |
Without persisting processing history, users cannot answer basic questions about past work:
"What subtitles were used for S01E05?", "Which source files went into this render?",
"When was this episode processed?". The jsonstore infrastructure already exists but is unused.
This fills the gap between ephemeral .done files and permanent archive.

## evidence:
  - type: code_review
    reference: internal/app/pipeline/pipeline.go:155-175 -- .done file deletion loop
  - type: code_review
    reference: internal/infra/persistence/jsonstore/jsonstore.go -- Store API (Get/Set/Delete/Save)
  - type: code_review
    reference: internal/app/app.go:76-84 -- Store wired in AppModules but never used
  - type: code_review
    reference: internal/infra/config/config.go:167-168 -- default storage path
  - type: code_review
    reference: internal/app/pipeline/pipeline.go:323-333 -- .done file JSON parsing in findDoneFile

## acceptance_criteria:
  - criterion: Pipeline stores .done JSON payload in jsonstore before deleting the .done file
    verification_method: unit test verifies Store.Set called with correct key and payload
  - criterion: Store is accessible via Pipeline.Options.Store field
    verification_method: compilation check; cmd_process.go wires modules.Store() into pipeline.Options
  - criterion: If store write fails, .done file is NOT deleted (safe fallback)
    verification_method: unit test simulates store error, verifies .done file persists
  - criterion: Store.Save() is called on pipeline cycle end in RunOnece (before 	`p.emit(EventScanDone, 0, nil)`)
    verification_method: integration test verifies storage.json contains entries after RunOnce
  - criterion: Existing tests continue to pass (sanitization, .done file handling)
    verification_method: go test -race -cover ./...

## non_goals:
  - Query API for history (read access from jsonstore is future work)
  - Migrate existing .done files to store on upgrade
  - Replace jsonstore with a real database (storage layer abstraction)
  - History retention policy (TTL, cleanup of old entries)
  - UI/CLI command to browse history

## constraints:
  - Pipeline must work with domain purity rules: Store is infra, accessed via Options (not domain)
  - jsonstore is thread-safe (RWMutex); pipeline runs single-threaded in Run/RunOnce
  - Store path comes from config (cfg.Storage.Path); empty path means store is disabled (no-op)
  - Backward compatible: existing .done files are unaffected; store is additive
  - Key format in store: "script_{script_name}" -- script_name includes timestamp and project name

## risks:
  - Store write failure blocks .done deletion, pending files accumulate (mitigated: retried each cycle)
  - Store file corruption on crash (mitigated: jsonstore uses atomic-ish write via os.WriteFile; future: write to temp then rename)
  - Store grows unbounded over time (non-goal: retention policy is future work)

## open_questions:
  - Should store write happen in the same goroutine (blocking) or async (fire-and-forget)?
  - Should the store entry include the full .done payload or a subset?
  - [FUTURE] Fallback mechanism: when store write fails, log ERROR-level entry with payload for manual recovery?
  - [FUTURE] Transaction scope: should store write + .done deletion be wrapped in a single transaction block?
