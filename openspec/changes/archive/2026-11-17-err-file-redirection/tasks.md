# Tasks

## 1. Branch and baseline

- [x] 1.1 Create the feature branch for this change (project rule: the current feature is implemented on its own branch) and verify `git branch --show-current` reports it
- [x] 1.2 Confirm the baseline is green with `go build ./... && go test -race -cover ./...` before touching anything

## 2. Error log store (infrastructure)

- [x] 2.1 Add the mandatory `error_store_path` config key and verify `internal/infra/config/config_test.go` covers a configured value, default retention, and rejection of an explicitly empty value
- [x] 2.2 Create a second store instance in `internal/infra/persistence/jsonstore/` per config path (done via `AppModules.ErrorStore()`); the store is opened eagerly at command startup so an unopenable path fails fast
- [x] 2.3 Add the error store path and the status path to the built-in configuration sample in `internal/infra/config/config.go` and verify the default-config test still reports the same required fields

## 3. Script generation (domain: script)

- [x] 3.1 Extract the render base-name derivation currently inlined in `buildReadyFile` (`internal/domain/script/script.go:299`) into one helper over the predictions, and verify `go test ./internal/domain/script/...` passes with the ready-marker tests unchanged
- [x] 3.2 Add the error-log copy block: check for `.#err` in in-progress directory, copy to target as `<OUTBASE>.#err` AND copy to metadata as `<SCRIPT_NAME>.err` (full text, nounset-safe; copy, never move), and verify new `internal/domain/script/script_test.go` cases assert the guarded `-f` test, the target path named `<OUTBASE>.#err`, the sidecar path named `<SCRIPT_NAME>.err` containing the log text, and that a clean render's block exits zero so the `&&` chain survives
- [x] 3.3 Add the ready marker copy to the target directory (after the error block) and verify a test asserts the ordering and that no `.#err` or `.ready` is written into the notification directory
- [x] 3.4 Confirm every new placeholder is filled by the builder and verify `internal/app/script/build_test.go` reports no unresolved placeholders and contains the new lines
- [x] 3.5 Remove `Notifications` from required environment fields in `Environment` (it is no longer written by scripts) and verify existing tests still pass

## 4. Error store writes (application: pipeline)

- [x] 4.1 After consuming a `.done` marker, read `<SCRIPT_NAME>.err` sidecar from metadata directory; if present, take its content as `error_text` and write an entry to the error store (script name, source dir, error_text, timestamp); verify `internal/app/pipeline/pipeline_test.go` covers the errored case, the clean case, and a pre-change payload (the clean/no-entry behavior is refined in section 9)
- [x] 4.2 Remove the sidecar marker alongside the done artifact after a successful commit to both stores; keep both when either store write aborts; verify tests assert marker presence after abort and absence after success
- [x] 4.3 Wire `AppModules.ErrorStore()` into `pipeline.Options` in `cmd_process.go` (production wiring bug: the store was never handed to the pipeline); ensure an unopenable error store path aborts startup with an early error return
- [x] 4.4 Ensure the error store write does not happen if the main store write fails (atomicity); verify a test asserts the error store is not written when the main store write is aborted

## 5. Generated script execution (end to end with bash)

- [x] 5.1 Extend `internal/app/script/build_e2e_test.go` to plant an error log beside the in-progress source before running the generated script, and verify: the target directory ends up with `<OUTBASE>.#err` sharing its base name with the ready marker; the metadata directory holds `<SCRIPT_NAME>.err` containing the planted error text; the planted source is untouched; the in-progress directory still holds the original `.#err`; no files are written to the notification directory
- [x] 5.2 Add the clean-render case to the same test file, and verify no `.#err` appears in the target directory, no `.err` sidecar exists in metadata, the ready marker is still copied, the script exits zero
- [x] 5.3 Verify the notification directory is untouched in both cases

## 6. Documentation

- [x] 6.1 Record the two naming families, their exception for the operator's own announcement name, and the new artifacts in `doc/domain_script.md`, and verify each documented name matches what the tests show the generated script produces
- [x] 6.2 Describe the error log store, its entry format (including `error_text`), the sidecar marker, the mandatory config paths with fail-fast startup, and the atomicity guarantee in `doc/architecture.md` next to the existing storage description, and verify the described flow matches `internal/app/pipeline/pipeline.go` and `internal/infra/config/config.go`

## 7. Integration verification

- [x] 7.1 Run `scripts/live_test.sh` and confirm the regenerated script in the test ground carries the copy block (target + sidecar) in the expected order, with the rest of the artifact tree unchanged
- [x] 7.2 Confirm static checks are clean with `go vet ./...`, `golangci-lint run ./...`, and `gofmt -l`

## 8. Mandatory writable paths and startup fail-fast

- [x] 8.1 Make `logger.path`, `storage.path`, `storage.error_store_path`, and `storage.status_path` mandatory config fields (non-empty, validated at load) and cover each with a `config_test.go` case
- [x] 8.2 Add the `storage.status_path` config key; the pipeline uses it instead of deriving the status file next to the storage path, and `RunOnce` writes the status report so `--once` runs also update it
- [x] 8.3 Open every writable file eagerly at startup (log via `Init`, both stores via `AppModules`, status via `AppModules.EnsureStatusFile`) and return an early error when a path cannot be opened for writing
- [x] 8.4 Update the delta spec and design to replace the optional-error-store wording with mandatory-paths fail-fast wording

## 9. Error store records only failed renders

- [x] 9.1 Write the error store entry only when the `<SCRIPT_NAME>.err` sidecar actually exists; a clean render adds no entry, so every entry in the store is a real failure (`internal/app/pipeline/pipeline.go`)
- [x] 9.2 Update the pipeline tests so the clean-render and pre-change-payload cases assert the error store stays empty while the processing history entry is still written
- [x] 9.3 Update `scripts/live_test_2.sh` to assert the error store holds exactly the planted failures while the history store holds the clean renders too
- [x] 9.4 Update proposal, design, and the execution-signals spec delta to state that only renders with an actual error log are recorded

## 10. Review follow-ups

- [x] 10.1 Cover the "Mixed pass" scenario end to end: `scripts/live_test_2.sh` plants an error log for three of the nine generated scripts, and a single `lazyam3 process --once` commits every marker of that executor cycle, so the same pass records three error store entries and eight history entries
- [x] 10.2 Cover the "Writable path cannot be opened" requirement with `internal/app/app_test.go`: `Store`, `ErrorStore`, and `EnsureStatusFile` each fail with a wrapped open error, the failure is cached across calls, the methods refuse to run before `Init`, and `Init` itself fails when the log path cannot be opened
- [x] 10.3 Assert `source_dir` and `timestamp` on the error store entry, and that a single errored render produces exactly one entry
- [x] 10.4 Correct design decisions 5, 6, and 9 so they match the implemented script behaviour (the only `touch` is in the target directory), the single `<OUTBASE>` derivation (the error store entry is keyed by script name), and the joint cleanup step
- [x] 10.5 Run `golangci-lint run ./...` and resolve every finding: `RunOnce` complexity was lowered by extracting the marker commit step, the unused script-name parameter was dropped from the copy block builder, and the two UNC-joining helpers were reformatted
- [x] 10.6 Delete the standalone ready-file builder left unreachable once the ready marker moved into the copy block, and move its base-name coverage to `TestRenderOutBase`

## Workflow follow-up

- Archive the change once implementation and review requirements are satisfied (`openspec-verify-change`, then `openspec-archive-change`).
- ADRs extracted ahead of archive: `doc/adr/0015` (error log redirection through the metadata sidecar), `0016` (error store records only renders with an error log), `0017` (two naming families, one OUTBASE derivation), `0018` (mandatory writable paths fail fast at startup). `doc/adr/INDEX.md` lists them.
