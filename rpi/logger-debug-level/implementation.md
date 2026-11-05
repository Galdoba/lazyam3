# Logger Debug Level Implementation

## Summary

Enabled Debug-level logging visibility into per-unit processing decisions, file operations, metadata lookups, script generation details, and storage operations when the `[logger]` level is configured as `debug`. Phase 1 downgraded per-unit Info calls to Debug across pipeline.go, ingest.go, build.go, cmd_process.go, repository.go, and jsonstore.go. Phase 2 added Debug calls at decision points in pipeline.go (loop entry, ingest complete, metadata lookup, script config built, files prepared, findDoneFile entries, captureProcessingHistory). Phase 3 added Debug logging to filescanner.go (scan complete with files/dirs/writable counts) and ffprobe.go (probe start, result status), requiring logger parameters to be threaded through the fso package. Phase 4 added unit tests for debug/info level filtering and verified via live_test.sh that Debug messages appear at debug level and are filtered at info level. Build, test, and vet pass. The golangci-lint warning in internal/domain/callname/shared.go is pre-existing and unrelated to this change.

## Pull Requests

| PR | Description | Status |
|----|-------------|--------|
| [#1](https://github.com/Galdoba/lazyam3/pull/new/implement/logger-debug-level) | Logger debug level implementation | Draft |

## Unexpected Discoveries

- The fso package (filescanner.go and ffprobe.go) had no logging infrastructure. To add Debug calls, I added a `Logger *logging.Logger` field to `fso.Options` and a `logger *logging.Logger` field to the `Prober` struct, updating `DefaultOptions()`, `NewProber()`, and their callers (pipeline.go and ingest.go).
- The pipeline.Logger is an interface (`pipeline.Logger`), while fso.Options.Logger expects `*logging.Logger`. In pipeline.go I convert via `p.logger.(*logging.Logger)` before passing to `fso.Scan`.
- ffprobe.go ProbeFile returns raw JSON without stream counts, so "streams found count" from the plan could not be logged there; I logged probe start and result status instead.

## Plan Deviations

- Phase 3 filescanner.go: the plan asked for "root path scanned, files found count, dirs found count, and writable check summary". I implemented one aggregate Debug call "filescanner scan complete" at the end of Scan with root, files, dirs, and writable counts, plus the root path is implicitly captured in the message.
- Phase 3 ffprobe.go: the plan asked for "file path being probed, probe result status, and streams found count". Streams count is not available in ProbeFile (it returns raw JSON), so I logged probe start (file path) and probe result status (success/fail/stat/invalid json) without stream counts.

## Architecture Decision Records

### Logger threading through fso

context:
filescanner.go and ffprobe.go needed Debug logging but had no logger parameter or global logger.

options:
- Add a logger parameter to Scan() and NewProber() signatures.
- Use a global logger in the fso package.
- Skip logging in fso entirely.

decision:
Add a `Logger *logging.Logger` field to `fso.Options` and a `logger *logging.Logger` field to `Prober`, updating callers to pass the logger when available.

rationale:
This keeps logging explicit and testable without introducing global state, and matches the existing logging pattern in app/infra layers.

consequences:
- Callers of fso.Scan and NewProber must pass the logger.
- pipeline.go and ingest.go were updated to thread the logger through.
- ffprobe_test.go calls to NewProber were updated to pass nil.

### Pipeline logger conversion

context:
pipeline.go uses a pipeline.Logger interface, but fso.Options.Logger requires a concrete *logging.Logger.

options:
- Change pipeline.Logger interface to accept concrete logging.Logger.
- Convert via type assertion in pipeline.go before calling fso.Scan.

decision:
Convert via `log, ok := p.logger.(*logging.Logger)` and set `opts.Logger = log` when the assertion succeeds.

rationale:
This avoids changing the pipeline.Logger interface and preserves the existing abstraction while allowing fso Debug logging when a concrete logger is present.

consequences:
- If p.logger is not a *logging.Logger (e.g. a custom implementation), fso logging is silently skipped.

## Handoff Notes

### Known Limitations
- fso.Debug logging only fires when a *logging.Logger is passed through Options/Prober; custom pipeline.Logger implementations that are not *logging.Logger will not trigger fso Debug logs.
- ffprobe probe Debug logging logs probe start and result status but not stream counts, since ProbeFile returns raw JSON.

### Data Models
- fso.Options: extended with `Logger *logging.Logger` field for debug logging.
- fso.Prober: extended with `logger *logging.Logger` field.

### API Surface
- `fso.Scan(root string, opts *fso.Options)`: now respects `opts.Logger` for Debug output.
- `fso.NewProber(binPath string, logger *logging.Logger)`: now accepts a logger for Debug output.

## Reference
- internal/app/pipeline/pipeline.go
- internal/app/ingest/ingest.go
- internal/app/script/build.go
- internal/infra/commands/cmd_process.go
- internal/infra/metadata/repository.go
- internal/infra/persistence/jsonstore/jsonstore.go
- internal/infra/fso/filescanner.go
- internal/infra/fso/ffprobe.go
- internal/infra/logging/logger_test.go
