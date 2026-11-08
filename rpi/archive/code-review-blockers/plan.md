# Code Review Blockers Plan

## Implementation Overview

This plan resolves three independent blocker categories from the code review so the build, test, and strict-lint gate are green again. Each fix is scoped to the smallest change that satisfies the constraint, following the project rule to make only minimal edits and to keep the domain layer pure.

High-level approach:
1. Harden the Renderer so a nil logger degrades gracefully instead of panicking.
2. Remove the forbidden time.Sleep from the PRT timestamp helpers while keeping PRT uniqueness across sequential buildScriptConfig calls (two versions of the same episode in one cycle must not collide).
3. Reformat the gofumpt violation in the pipeline log line.

Key architectural decisions and patterns (informed by golang-safety, golang-concurrency, and golang-design-patterns):
- Nil logger: apply the nil-deref guard pattern (check the receiver field before calling a method). The Renderer.New constructor leaves logger nil unless WithLogger is passed, so the nil-check is the correct defensive fix (golang-safety).
- PRT uniqueness: protect the shared lastPRTSec counter with a package-level sync.Mutex and return a strictly increasing 12-digit timestamp by incrementing the second on collision. sync.Mutex has a usable zero value, so no lazy init is needed (golang-safety and golang-concurrency). Keep the critical section short with no I/O inside the lock (golang-concurrency).
- Timezone and format preservation: newPRTTimestamp currently returns local time via time.Now().Format("060102150405") with no PRT prefix. The monotonic generator must preserve that exact local-time format and no-prefix behavior so rendered script names change only in the uniqueness guarantee. Do not switch to UTC unless a deliberate behavior change is intended.
- Avoid over-abstraction: the unused internal/platform/clock/clock.go abstraction is not needed for this blocker; the mutex-plus-counter is the minimal solution (golang-design-patterns: smallest pattern that fits).

Integration points:
- Renderer.Render feeds into script.Build, which passes a concrete *logging.Logger. The nil-check keeps Build working whether or not a logger is injected.
- newPRTTimestamp feeds buildScriptConfig, which feeds PredictScriptName (names_output.go). The 12-digit PRT must still satisfy the prtRe regex.

Performance and scalability:
- The mutex is held only for integer comparisons and one time formatting call, so contention is negligible for a pipeline that processes a modest number of episodes. No lock is held across file I/O.

## Task Breakdown

### Phase 1: Nil-safe Renderer logging
- [x] In internal/infra/script/script.go wrap the r.logger.Info call at line 58 in `if r.logger != nil { ... }`. Confirm New (lines 36-43) still leaves logger nil by default so the guard is required. Verify by building and running the Renderer test without WithLogger.

### Phase 2: Remove time.Sleep from NewPRT
- [x] In internal/domain/callname/shared.go delete the time.Sleep(time.Second) call at line 9 and keep `return "PRT" + time.Now().UTC().Format("060102150405")`. Verify the forbidigo linter no longer flags the file and the NewPRT unit test still passes (PRT prefix plus 12 digits).

### Phase 3: Monotonic PRT timestamp generator
- [x] In internal/app/pipeline/config.go add `var prtMu sync.Mutex` and `var lastPRTSec int64` just above newPRTTimestamp (lines 143-146), and add `import "sync"`. Replace the sleep-based body with a mutex-guarded generator that reads time.Now().Unix(), forces the value strictly greater than lastPRTSec on collision (lastPRTSec + 1), stores it back, and returns time.Unix(sec, 0).Format("060102150405") in local time with no PRT prefix. Verify the forbidigo linter no longer flags config.go and add a unit test that calls the generator multiple times in quick succession and asserts strictly increasing 12-digit strings.

### Phase 4: Fix gofumpt formatting
- [x] In internal/app/pipeline/pipeline.go reformat the script-executed log line at line 237 so the file is gofumpt-clean (run the project formatter on the file). Verify the gofumpt check inside golangci-lint passes.

### Phase 5: Full gate verification
- [x] Run go build ./..., go vet ./..., go test -race -cover ./..., golangci-lint run ./..., and scripts/live_test.sh. Confirm every command exits cleanly and the end-to-end pipeline pass succeeds.

## Code References

Order follows implementation sequence.

```
internal/infra/script/script.go:58
internal/infra/script/script.go:36-43
internal/app/script/build.go:13-31
internal/domain/callname/shared.go:9
internal/domain/callname/shared.go:8-11
internal/app/pipeline/config.go:143-146
internal/app/pipeline/config.go:144
internal/domain/callname/names_output.go:207-220
internal/app/pipeline/pipeline.go:237
internal/platform/clock/clock.go
```

## Testing Plan

### Unit tests
- Renderer: the existing Renderer test in the infra/script package must pass without panic when the logger is not injected (no WithLogger).
- NewPRT: the callname test must still accept the output (PRT prefix plus 12 digits) after removing the sleep.
- newPRTTimestamp: add a test in the pipeline package that invokes the generator in a tight loop and asserts each returned value is strictly greater than the previous and matches the 12-digit format. Because the generator deliberately uses package-level state, the test asserts the monotonic property rather than absolute values.

### Integration tests
- BuildAndExecute: the script package test that builds and executes real inputs must pass without a nil-logger panic.

### End-to-end test
- Run scripts/live_test.sh to confirm a full pipeline pass renders scripts and generates unique PRT script names when two versions of the same episode appear.

### Lint and static analysis
- golangci-lint run ./... must exit 0 (no forbidigo or gofumpt issues).
- go vet ./... and go build ./... must pass.

### Regression considerations
- Confirm removing the 1-second sleep does not break existing expectations, and that PRT uniqueness holds under the same-second collision scenario.

## FACTS Scale Output

F: 5  A: 5  C: 5  T: 5  S: 4  Mean: 4.80  --> PASS

## Dependencies & Sequencing

### Dependency chain
- Phase 1 (Renderer nil-check), Phase 2 (NewPRT), and Phase 4 (formatting) are independent of each other.
- Phase 3 (monotonic generator) is independent of Phases 1, 2, and 4 but shares the pipeline package with Phase 4, so Phase 4 should run after Phase 3 so the formatter reflows any lines Phase 3 introduces.
- Phase 5 depends on Phases 1 through 4.

### Critical path
- Phase 3 is the highest-risk task (new shared state, monotonic correctness, test determinism) and should be implemented and unit-tested before the final gate.

### External system dependencies
- The strict-lint gate (golangci-lint with forbidigo and gofumpt) and the live_test.sh end-to-end script.

## Risk Mitigation

### Technical risks
- Global state in Phase 3: the package-level lastPRTSec persists across calls and tests. Mitigation: the test asserts the monotonic property, which is exactly the invariant under test; keep the lock critical section minimal. If test isolation becomes an issue, add a small unexported reset helper used only by tests.
- Behavior change risk: switching to UTC or adding a PRT prefix would alter rendered script names. Mitigation: preserve local time and no-prefix semantics exactly as the current code.
- Collision correctness: two same-second episodes must get distinct timestamps. Mitigation: the strictly-increasing guard guarantees distinctness; the unit test exercises the same-second path.
- Pacing change: the 1-second delay also paced the pipeline. Mitigation: verify live_test.sh still passes; the pacing change is a deliberate improvement, not a regression.

### Early warning indicators
- go test -race reports a data race on lastPRTSec (would indicate a missing lock).
- golangci-lint still reports forbidigo on config.go (would indicate a leftover time.Sleep).

### Escalation criteria
- If the monotonic generator cannot preserve the 12-digit format under extreme collision rates, escalate to reconsider a sub-second component in the timestamp rather than changing the format.

## Rollback Strategy

### Phase 1
- Rollback point: the single nil-check edit in script.go. Revert by removing the guard and restoring the unconditional r.logger.Info call. Data impact: none.

### Phase 2
- Rollback point: the single line deletion in shared.go. Revert by re-adding time.Sleep(time.Second). Data impact: none.

### Phase 3
- Rollback point: the mutex, the two globals, and the newPRTTimestamp body in config.go. Revert by restoring the original sleep-based function and removing the sync import. Data impact: none; only timestamp values change.

### Phase 4
- Rollback point: the formatter reformatting of pipeline.go. Revert by running the formatter against the previous commit version of the file. Data impact: none.

### General
- All changes are confined to four source files and are isolated; each can be reverted independently without affecting the others. No database or config migration is involved, so no data-reversal procedure is required.
