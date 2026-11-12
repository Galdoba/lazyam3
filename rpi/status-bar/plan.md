# Status Bar Plan

## Implementation Overview

- Use ANSI `\r` escape sequences via Go stdlib to write a single-line status bar to stderr
- Status bar is separate from structured logs; writes directly to os.Stderr
- During active processing (RunOnce): show "scanning..." or stage names
- During dormancy (Run loop sleep): show countdown "dormant: HH:MM:SS"
- Inject a StatusWriter into Pipeline via Options; minimal abstraction
- No external dependencies required; compatible with existing CLI structure

## Task Breakdown

### Phase 1: StatusWriter interface and implementation
- [x] Add `StatusWriter` interface (with `Write(status string)` method) to `internal/app/pipeline/dto.go` alongside `Logger` interface
- [x] Create `internal/platform/status.go` - implement `StdStatusWriter` that writes ANSI `\r`-prefixed output to stderr

### Phase 2: Wire StatusWriter into Pipeline construction
- [x] Add `statusWriter StatusWriter` field to Pipeline struct in `internal/app/pipeline/pipeline.go:30-52`
- [x] Update `internal/app/pipeline/pipeline.go:55-88` NewPipeline to accept StatusWriter from Options, store it, and guard against nil (skip writes if nil)

### Phase 3: Emit status during RunOnce (active processing)
- [x] Update `internal/app/pipeline/pipeline.go:152-224` RunOnce to emit status at key stages: scan, unit processing, scan_done
- [x] Add status calls: `p.statusWriter.Write("scanning...")` before scan, `p.statusWriter.Write("processing: <dir>")` per unit, `p.statusWriter.Write("scan complete")` after emit(EventScanDone)

### Phase 4: Emit status during dormancy countdown in Run loop
- [x] Update `internal/app/pipeline/pipeline.go:129-145` dormancy section: replace silent timer sleep with a countdown loop that emits "dormant: HH:MM:SS" using `time.Until(deadline)`, polling `ctx.Err()` each iteration so cancellation isn't blocked

### Phase 5: Wire StatusWriter in cmd_process.go
- [x] Update `internal/infra/commands/cmd_process.go:108-127` to create a StatusWriter instance and pass it in Pipeline.Options

### Phase 6: Testing
- [ ] Run `scripts/live_test.sh` to verify status bar output during active processing and dormancy
- [ ] Verify status appears on terminal when `logger.terminal=true`, does not interfere with structured logs

## Code References

```
internal/app/pipeline/dto.go
internal/app/pipeline/pipeline.go:90-147
internal/app/pipeline/pipeline.go:152-224
internal/infra/commands/cmd_process.go:108-127
internal/platform/status.go [NEW]
```

## Testing Plan

- Run `scripts/live_test.sh` - verifies end-to-end pipeline with status bar output
- Verify "scanning..." appears during RunOnce active work
- Verify "dormant: HH:MM:SS" countdown appears during dormancy period
- Verify status does not mix with structured log output (logs remain intact)
- Edge case: non-TTY pipe - `\r` is harmless, no visual corruption

## FACTS Scale Output

F: 4  A: 3  C: 4  T: 4  S: 3  Mean: 3.60  --> PASS

## Dependencies & Sequencing

- Phase 1 (StatusWriter interface) must complete before Phase 2 (Pipeline wiring)
- Phase 2 (Pipeline construction) must complete before Phase 3 and 4 (status emission)
- Phase 5 (cmd_process.go wiring) must complete before Phase 6 (testing)
- Critical path: Phase 1 -> Phase 2 -> Phase 3 -> Phase 4 -> Phase 5 -> Phase 6

## Risk Mitigation

- **Risk**: Status bar output interferes with structured logs on stderr
  - **Mitigation**: StatusWriter writes with `\r` prefix, separate from slog output path. Logs use their own handler; status bar uses direct os.Stderr writes.
- **Risk**: Countdown formatting error during short dormancy periods
  - **Mitigation**: Use `time.Until(deadline)` which handles edge cases; format with `%02d:%02d:%02d` for safe zero-padded output.
- **Risk**: Status bar in non-TTY environments
  - **Mitigation**: `\r` is harmless in pipes - no visual harm, just overwrites. No TTY detection needed.

## Rollback Strategy

- Phase 1 rollback: Remove `internal/platform/status.go`
- Phase 2 rollback: Remove StatusWriter field from Pipeline struct and Options
- Phase 3 rollback: Remove status emit calls from RunOnce
- Phase 4 rollback: Restore original silent timer sleep in Run loop
- Phase 5 rollback: Remove StatusWriter creation in cmd_process.go
- Each phase is independently reversible; no shared state changes.
