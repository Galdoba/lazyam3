# Logger Debug Level Plan

## Implementation Overview

The goal is to enable Debug-level logging visibility into per-unit processing decisions, file operations, metadata lookups, script generation details, and storage operations when the `[logger]` level is configured as `debug`.

High-level approach:
- Downgrade per-unit Info calls to Debug (they fire every polling cycle or per-unit, flooding logs at Info level).
- Add new Debug calls at key decision points that are currently unlogged (ingest result, metadata lookup, script config build, file moves, predictions, storage operations).
- Keep lifecycle Info calls as-is (pipeline started/stopped, modules initialized, metadata repository created, create pipeline, pipeline finished).

Key architectural decisions:
- Follow existing logging conventions: `logger.Debug("msg", "key", val)`.
- Use `logger.With("unit_id", unit.ID).Debug(...)` for grouped per-unit messages.
- Avoid over-logging in tight loops: use aggregate Debug calls (e.g., "moved N files") instead of per-item Debug calls in `prepareFiles`.
- Logging stays in app/infra layers only. Domain files (`internal/domain/*`) are pure logic and must not contain logging.

Integration points: pipeline.go, config.go, ingest.go, script/build.go, cmd_process.go, metadata/repository.go, jsonstore.go, filescanner.go, ffprobe.go.

## Task Breakdown

Organized into logical phases with atomic tasks using checkboxes. Each task is single-responsibility, independently completable, and verifiable.

### Phase 1: Downgrade per-unit Info calls to Debug

- [x] In `internal/app/pipeline/pipeline.go:158`, change `p.logger.Info("scan")` to `p.logger.Debug("scan")`.
- [x] In `internal/app/pipeline/pipeline.go:164`, change `p.logger.Info("dirs found", "count", len(dirs))` to `p.logger.Debug("dirs found", "count", len(dirs))`.
- [x] In `internal/app/pipeline/pipeline.go:187`, change `p.logger.Info("sanitized empty dirs", "count", len(snap.SanitizeablePaths))` to `p.logger.Debug("sanitized empty dirs", "count", len(snap.SanitizeablePaths))`.
- [x] In `internal/app/pipeline/pipeline.go:246`, change `p.logger.Info("processing", "dir", dir)` to `p.logger.Debug("processing", "dir", dir)`.
- [x] In `internal/app/pipeline/pipeline.go:266`, change `p.logger.Info("dry run: script build skipped", "id", unit.ID)` to `p.logger.Debug("dry run: script build skipped", "id", unit.ID)`.
- [x] In `internal/app/ingest/ingest.go:31`, change `ing.logger.Info("ingesting directory", "dir", dir)` to `ing.logger.Debug("ingesting directory", "dir", dir)`.
- [x] In `internal/app/script/build.go:15`, change `logger.Info("building rendering script")` to `logger.Debug("building rendering script")`.
- [x] In `internal/app/script/build.go:28`, change `logger.Info("writing rendered script to destination", "path", config.Destination)` to `logger.Debug("writing rendered script to destination", "path", config.Destination)`.
- [x] In `internal/infra/commands/cmd_process.go:134`, change `logger.Info("unit done", "id", e.ID)` to `logger.Debug("unit done", "id", e.ID)`.
- [x] In `internal/infra/metadata/repository.go:61`, change `r.logger.Info("metadata reloaded", "path", r.metadataPath)` to `r.logger.Debug("metadata reloaded", "path", r.metadataPath)`.
- [x] In `internal/infra/persistence/jsonstore/jsonstore.go:122`, change `s.logger.Info("store saved entries", "count", len(s.data))` to `s.logger.Debug("store saved entries", "count", len(s.data))`.
- [x] In `internal/infra/persistence/jsonstore/jsonstore.go:135`, change `s.logger.Info("store closed and saved")` to `s.logger.Debug("store closed and saved")`.

### Phase 2: Add Debug calls in pipeline.go decision points

- [x] In `internal/app/pipeline/pipeline.go:196-212` loop entry, add `p.logger.Debug("unit loop entry", "dir", d.Path)` before processing each directory, and add Debug for `snap.FilesIn(d.Path)` count and `processUnit` result (success/failure).
- [x] In `internal/app/pipeline/pipeline.go:247-251` after ingest, add Debug for ingest complete: `p.logger.Debug("ingest complete", "unit_id", unit.ID, "streams", len(unit.Streams))`.
- [x] In `internal/app/pipeline/pipeline.go:255-264` after metadata lookup, add Debug for metadata found/not-found with fallback indication.
- [x] In `internal/app/pipeline/pipeline.go:270-274` after script config build, add Debug: `p.logger.Debug("script config built", "unit_id", unit.ID, "predictions", len(cfg.Predictions))`.
- [x] In `internal/app/pipeline/pipeline.go:335-349` prepareFiles, add aggregate Debug for files moved: `p.logger.Debug("files prepared", "count", len(callNames))` after the loop.
- [x] In `internal/app/pipeline/pipeline.go:362-388` findDoneFile, add Debug for entries scanned, payloads parsed, and matches found.
- [x] In `internal/app/pipeline/pipeline.go:220-242` captureProcessingHistory, add Debug for keys stored and save result.

### Phase 3: Add Debug calls in infra files

- [x] In `internal/infra/fso/filescanner.go:73-133`, add Debug for root path scanned, files found count, dirs found count, and writable check summary.
- [x] In `internal/infra/fso/ffprobe.go:41-57`, add Debug for file path being probed, probe result status, and streams found count.

### Phase 4: Tests and verification

- [x] Add unit tests verifying Debug messages are emitted when level is debug and filtered when level is info.
- [x] Run `scripts/live_test.sh` with `level = 'debug'` and verify Debug messages appear in terminal and file output.
- [x] Run with `level = 'info'` and verify Debug messages are NOT present.
- [x] Run `go test -race -cover ./...`, `go vet ./...`, and `golangci-lint run ./...`.

## Code References

```
internal/app/pipeline/pipeline.go:158
internal/app/pipeline/pipeline.go:164
internal/app/pipeline/pipeline.go:187
internal/app/pipeline/pipeline.go:246
internal/app/pipeline/pipeline.go:266
internal/app/pipeline/pipeline.go:196-212
internal/app/pipeline/pipeline.go:247-251
internal/app/pipeline/pipeline.go:255-264
internal/app/pipeline/pipeline.go:270-274
internal/app/pipeline/pipeline.go:335-349
internal/app/pipeline/pipeline.go:362-388
internal/app/pipeline/pipeline.go:220-242
internal/app/pipeline/config.go:28-141
internal/app/ingest/ingest.go:31
internal/app/script/build.go:15
internal/app/script/build.go:28
internal/infra/commands/cmd_process.go:134
internal/infra/metadata/repository.go:61
internal/infra/persistence/jsonstore/jsonstore.go:122
internal/infra/persistence/jsonstore/jsonstore.go:135
internal/infra/fso/filescanner.go:73-133
internal/infra/fso/ffprobe.go:41-57
```

## Testing Plan

- Unit tests: Verify Debug messages are written when level is "debug" and filtered when level is "info".
- Integration: Run `scripts/live_test.sh` with `level = 'debug'` and verify Debug messages appear in both terminal and file output.
- Filtering: Run with `level = 'info'` and verify Debug messages are NOT present.
- Regression: Run `go test -race -cover ./...` to ensure no existing tests break.

## FACTS Scale Output

F: 5  A: 5  C: 5  T: 5  S: 5  Mean: 5.00  --> PASS

## Dependencies & Sequencing

- Phase 1 must complete before Phase 2 and 3 (downgrades are prerequisites for consistent logging state).
- Phase 2 and 3 are independent of each other but both depend on Phase 1.
- Phase 4 (tests/verification) depends on all implementation phases.
- No external system dependencies; configmanager TOML `level = 'debug'` is the switch.

## Risk Mitigation

- Risk: Over-logging at debug level could flood logs in tight loops. Mitigation: use aggregate Debug calls (e.g., "moved N files") in prepareFiles and avoid per-item Debug in high-volume loops.
- Risk: Changing Info calls to Debug could break tests that assert on Info output. Mitigation: verify existing tests still pass; add tests for debug filtering.
- Risk: Logging in domain files. Mitigation: domain files (`internal/domain/*`) are pure logic and must not contain logging; all Debug calls stay in app/infra layers.
- Early warning indicator: `golangci-lint run` fails or `go vet` reports issues.
- Escalation criteria: If tests fail after changes, revert to the last passing state and re-evaluate the specific change.

## Rollback Strategy

- Each phase is independently reversible: changing `Debug` back to `Info` (or removing added Debug calls) restores prior behavior.
- No data migration or feature flags needed; the `level = 'debug'` TOML setting is the switch.
- If a phase causes regressions, revert only that phase's changes and re-run verification.
