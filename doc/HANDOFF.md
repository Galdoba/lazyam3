# HANDOFF

## Session: pipeline loop vs `once` behaviour

### Analysis
- Cycling = `Pipeline.Run` loop (`internal/app/pipeline/pipeline.go`):
  - `--once` → `RunOnce` single scan pass (`internal/infra/commands/cmd_process.go`).
  - default → `Run` loop: first pass immediate, then wait, repeat until ctx cancelled.
- Old wait mechanism: ad-hoc channel + goroutine sleeping on hardcoded `debugTumeout = 1` (TODO: configurable poll interval from config).
- Cancellation: ctx cancelled → stop after current unit, keep totals, return ctx.Err → command logs `pipeline stopped`, exit 0.

### Change made
- `Pipeline.Options.PollInterval` added (0 → `defaultPollInterval = 1s`).
- `Pipeline.Run` now waits via `time.NewTicker(p.pollInterval)` + ctx-aware select; `afterPoll` and `debugTumeout` removed.

### Tests
- Added `TestNewPipeline_PollIntervalDefault`, `TestRun_LoopsUntilContextCancelled` (two passes then cancel, totals checked).
- Gates: `go build ./...`, `go vet ./...`, `gofmt -l`, `golangci-lint run ./...` → clean; `go test -race ./...` → all ok.

### Watch out / next steps
- Poll interval source from config is still TODO (Options field added, wiring pending).
- `--once` fails fast on scan error; polling loop retries scan failures (warn + continue).

## Session: D1 — Dormancy period for cyclic mode

### Analysis
- Cycling = `Pipeline.Run` loop (`internal/app/pipeline/pipeline.go`):
  - `--once` → `RunOnce` single scan pass (`internal/infra/commands/cmd_process.go`).
  - default → `Run` loop: scan pass, then wait.
- Old wait mechanism: ad-hoc channel + goroutine sleeping on hardcoded `defaultPollInterval = 1s`.
- New mechanism: when `dormancyPeriod > 0`, sleeps for that duration after each pass (cancellable via context); when == 0, uses polling ticker.

### Changes made
- `internal/infra/config/config.go` — new `PipelineConfig` struct with `DormancyPeriod int` (seconds); `Config` gains `Pipeline` field; `defaultCFG()` sets 0.
- `internal/app/pipeline/dto.go` — `DormancyPeriod time.Duration` added to `Options`.
- `internal/app/pipeline/pipeline.go` — `Pipeline.dormancyPeriod` field; `NewPipeline` copies from options; `Run()` sleeps after each pass when > 0, falls back to polling when == 0.
- `internal/infra/commands/cmd_process.go` — `--dormancy` flag (Duration); falls back to `cfg.Pipeline.DormancyPeriod * time.Second`.
- 5 tests added covering: default value, sleep between passes, context cancellation during sleep, polling mode, continuation after scan failure.

### Tests
- `TestNewPipeline_DormancyPeriodDefault`, `TestRun_DormancyMode_SleepsBetweenPasses`, `TestRun_DormancyMode_ContextCancelledDuringSleep`, `TestRun_DormancyDisabled_Polling`, `TestRun_DormancyMode_ContinuesAfterScanFailure`.
- Gates: `go build ./...`, `go vet ./...`, `golangci-lint run ./...` → clean; `go test -race ./...` → all ok.
- Live test: config `[pipeline]` section with `dormancy_period = 10` parses correctly; scans 10s apart confirmed.

### Watch out / next steps
- Config stores `DormancyPeriod` as `int` (seconds), converted to `time.Duration` in cmd_process.go wiring.
- `--dormancy` flag overrides config value; flag takes precedence.
- `--once` mode bypasses `Run()` entirely, dormancy is irrelevant.
