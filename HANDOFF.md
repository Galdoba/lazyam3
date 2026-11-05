# HANDOFF: logger-debug-level Implementation

## Summary

Implemented Debug-level logging visibility for the media processing pipeline. When `[logger] level = 'debug'` is configured, per-unit processing decisions, file operations, metadata lookups, script generation details, and storage operations now emit Debug messages. Info-level calls that fired every polling cycle or per-unit were downgraded to Debug to avoid log flooding.

## Changes Made

### Phase 1: Downgraded Info to Debug
- `internal/app/pipeline/pipeline.go`: scan, dirs found, sanitized empty dirs, processing, dry run messages
- `internal/app/ingest/ingest.go`: ingesting directory
- `internal/app/script/build.go`: building rendering script, writing rendered script
- `internal/infra/commands/cmd_process.go`: unit done
- `internal/infra/metadata/repository.go`: metadata reloaded
- `internal/infra/persistence/jsonstore/jsonstore.go`: store saved entries, store closed and saved

### Phase 2: Pipeline decision-point Debug calls
- `internal/app/pipeline/pipeline.go`: unit loop entry, ingest complete, metadata lookup failed/not found, script config built, files prepared, findDoneFile entries scanned/matches found, captureProcessingHistory stored/saved

### Phase 3: Infra file logging (required logger threading)
- `internal/infra/fso/filescanner.go`: added `Logger *logging.Logger` to `Options`, logs "filescanner scan complete" with root, files, dirs, writable counts
- `internal/infra/fso/ffprobe.go`: added `logger *logging.Logger` to `Prober`, logs probe start, stat failed, probe failed, invalid json, probe success
- Updated callers: pipeline.go threads logger to `fso.Scan`, ingest.go passes logger to `fso.NewProber`

### Phase 4: Tests and verification
- Added `TestLogger_DebugLevel_EmitsDebug` and `TestLogger_InfoLevel_FilterDebug` to `internal/infra/logging/logger_test.go`
- Verified via `scripts/live_test.sh` with `level = 'debug'`: Debug messages appear in terminal and file output
- Verified with `level = 'info'`: Debug messages are filtered, only INFO/WARN appear

## Verification Results

- `go build ./...`: passes
- `go test -race -cover ./...`: passes
- `go vet ./...`: passes
- `golangci-lint run ./...`: 1 pre-existing warning in `internal/domain/callname/shared.go` (time.Sleep forbidden) — unrelated to this change
- Live test with debug level: all expected Debug messages emitted
- Live test with info level: Debug messages correctly filtered

## Implementation.md Location

`rpi/logger-debug-level/implementation.md` — validated against deterministic checks.

## Draft PR

Branch: `implement/logger-debug-level` (pushed). Draft PR to be created by human at `https://github.com/Galdoba/lazyam3/pull/new/implement/logger-debug-level`.

## Known Limitations

- fso.Debug logging only fires when a `*logging.Logger` is passed; custom `pipeline.Logger` implementations not typed as `*logging.Logger` won't trigger fso Debug logs.
- ffprobe probe Debug logs result status but not stream counts (ProbeFile returns raw JSON).
