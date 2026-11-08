# Code Review Blockers Research

## Problem Context

- Restated problem: Fix three categories of failures from the codebase review: (1) nil-logger panic in `Renderer.Render`, (2) `time.Sleep` forbidigo violations in two files, (3) gofumpt formatting violation in `internal/app/pipeline/pipeline.go:237`.
- Business/functional intent: Restore green build/test/lint gate; the pipeline must render scripts and generate PRT timestamps without panicking or violating lint rules.
- Current vs desired behavior: Tests panic on nil logger; lint gate fails on `time.Sleep` and gofumpt; desired is passing tests and a clean strict-lint gate.
- Constraints: Domain layer must stay pure (stdlib-only); `NewPRT()` format must remain `PRT` + 12-digit `060102150405` timestamp to satisfy `prtRe` regex; no production `time.Sleep`; PRT must be unique per `buildScriptConfig` call to avoid script-name collisions when 2 versions of the same episode appear in one cycle.

## Affected Files

```
internal/infra/script/script.go:58
internal/infra/script/script.go:36-43
internal/app/script/build.go:13-31
internal/domain/callname/shared.go:8-11
internal/app/pipeline/config.go:143-146
internal/app/pipeline/pipeline.go:237
```

- `internal/infra/script/script.go:58`: `r.logger.Info(...)` called unconditionally; `New()` (36-43) only sets `mode`, leaves `logger` nil unless `WithLogger` used.
- `internal/app/script/build.go:13-31`: `Build` takes `*logging.Logger`, passes to renderer; `Build` guards its own logs with `if logger != nil` but does not control renderer's logger.
- `internal/domain/callname/shared.go:8-11`: `NewPRT()` sleeps 1s then formats timestamp; only used by its own test.
- `internal/app/pipeline/config.go:143-146`: `newPRTTimestamp()` sleeps 1s to guarantee uniqueness across sequential `buildScriptConfig` calls (2 versions of same episode in one cycle); used in `buildScriptConfig`.
- `internal/app/pipeline/pipeline.go:237`: gofumpt formatting violation (script-name log line).

## Code Examples

Current `Renderer.Render`:
```go
func (r *Renderer) Render(sc dscript.Script) error {
	// ... write file ...
	r.logger.Info("script rendered", "path", sc.Destination) // panics if r.logger == nil
	return nil
}
```

Proposed (nil-safe):
```go
func (r *Renderer) Render(sc dscript.Script) error {
	// ... write file ...
	if r.logger != nil {
		r.logger.Info("script rendered", "path", sc.Destination)
	}
	return nil
}
```

Current `NewPRT` / `newPRTTimestamp`:
```go
func NewPRT() string {
	time.Sleep(time.Second)
	return "PRT" + time.Now().UTC().Format("060102150405")
}
```

Proposed `newPRTTimestamp` (mutex-protected monotonic seconds, no sleep):
```go
var prtMu sync.Mutex
var lastPRTSec int64

func newPRTTimestamp() string {
	prtMu.Lock()
	defer prtMu.Unlock()
	sec := time.Now().UTC().Unix()
	if lastPRTSec != 0 && sec <= lastPRTSec {
		sec = lastPRTSec + 1
	}
	lastPRTSec = sec
	return time.Unix(sec, 0).UTC().Format("060102150405")
}
```

Proposed `NewPRT` (test-only, remove sleep):
```go
func NewPRT() string {
	return "PRT" + time.Now().UTC().Format("060102150405")
}
```

## External Research Findings

- Go nil-deref panic: calling a method on a nil pointer panics with `panic: nil pointer dereference`. The idiomatic defensive pattern is a nil-check before method calls, or ensuring constructors never return objects with nil dependencies. (golang-safety pattern)
- `time.Sleep` in production: forbidden by project lint; idiomatic alternatives use `time.Timer`, `time.After`, or `context`-based timeouts rather than blocking sleeps. For PRT timestamp uniqueness within the same second, a mutex-protected monotonic counter that increments the seconds field preserves the 12-digit format without blocking. (golang-concurrency / golang-design-patterns)

## FAR Scale Output

### Factual Score: 5
### Actionable Score: 4
### Relevant Score: 5
### Mean: 4.67  --> PASS

## Testing Strategy

- Unit: `internal/infra/script.TestRenderer_Render` and `internal/app/script.TestBuildAndExecute_RealInputs` must pass without panic when logger is not injected.
- Unit: `internal/domain/callname` test `TestNewPRT` must still pass (prefix + 12-digit parse) after removing sleep.
- Lint: `golangci-lint run ./...` must exit 0; `go vet ./...` and `go build ./...` pass.
- Repro: run `scripts/live_test.sh` to verify end-to-end pipeline pass.

## Potential Design Pattern Recommendations

- Logger port vs concrete: `Build` takes `*logging.Logger` (infra concrete) instead of the `pipeline.Logger` port, a design finding to consider, but out of scope for the blocker fix (nil-check is sufficient).
- PRT uniqueness: `newPRTTimestamp()` must return strictly increasing 12-digit timestamps across sequential calls within the same cycle (2 versions of same episode). Solution: module-level `sync.Mutex` + `lastPRTSec` tracker that increments the second on collision, preserving format and `prtRe` regex.

## Assumptions

- `Renderer` is constructed without `WithLogger` in the failing tests (verified by `New()` implementation).
- `NewPRT()` is only used in its test file (verified via grep).
- The 12-digit timestamp format must be preserved for `prtRe` regex (verified by test expectations).
- 2 versions of the same episode can appear in a single pipeline cycle; their PRT timestamps must differ to avoid script-name collisions.

## Out of Scope

- App to infra coupling reduction (type-asserts in pipeline, `Build` taking concrete logger).
- Stale TODOs removal beyond the gofumpt line.
- Fate of unused `clock`/`retry`/`cache` packages.
