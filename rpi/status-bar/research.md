# Status Bar for Pipeline Stages Research

## Problem Context

- **Restated problem**: Users see no output when the pipeline is dormant. They need a status bar showing active stages during processing and a countdown timer during dormancy.
- **Business intent**: Improve operator visibility into what the pipeline is doing, when it's busy, and when it's idle.
- **Current behavior**: During active processing, only slog-level debug/info messages appear (visible only when terminal logging is enabled). During dormancy (sleep between scan passes), no output is produced.
- **Desired behavior**: A persistent status line showing "working..." during active work and "dormant: HH:MM:SS" with a countdown timer during idle periods.
- **Constraints**: Must work with existing Go stdlib and minimal dependencies; compatible with current CLI structure; should not block the pipeline loop.

## Affected Files

```
internal/app/pipeline/pipeline.go:90-147
Pipeline.Run() - the main loop that transitions between scan passes and dormancy. This is the primary location where status updates need to be emitted.

internal/app/pipeline/pipeline.go:152-224
Pipeline.RunOnce() - the single scan pass that executes scan, ingest, metadata, script build, render. Status updates during this phase need finer granularity.

internal/infra/commands/cmd_process.go:129-141
Event consumer goroutine - currently logs unit_done/unit_failed. Could be extended to emit status events or wire a status bar.

internal/infra/logging/logger.go:38-41
Logger surface (Debug, Info, Warn, Error) - current logging mechanism. Status bar would need a separate output path, not mixed with structured logs.

internal/presentation/tui/tui.go:1-107
Bubbletea v2 model - exists but not wired to the pipeline. Could be extended for a status bar, but adds dependency complexity.

internal/app/pipeline/dto.go:51-71
PipelineEvent types (unit_done, unit_failed, scan_done) - current event infrastructure. Could be extended to include a "status_update" event.

internal/infra/config/config.go:125-133
Logging config (Terminal, Format, Color) - determines whether terminal output is visible. Status bar would need its own config or piggyback on terminal logging.

internal/platform/clock/clock.go
Clock abstraction - if time tracking for countdown needs to be abstracted.
```

## Code Examples

### Current pipeline loop (dormancy period)
`internal/app/pipeline/pipeline.go:129-145`
```go
if p.dormancyPeriod > 0 {
    timer := time.NewTimer(p.dormancyPeriod)
    select {
    case <-ctx.Done():
        timer.Stop()
        return total, ctx.Err()
    case <-timer.C:
    }
} else {
    select {
    case <-ctx.Done():
        return total, ctx.Err()
    case <-ticker.C:
    }
}
```
No output during this sleep period. The timer counts down silently.

### Current RunOnce structure
`internal/app/pipeline/pipeline.go:152-224`
```go
p.logger.Debug("scan")
snap, err := fso.Scan(p.sourceDir, opts)
// ... ingest, metadata lookup, script build, render
p.emit(EventScanDone, 0, nil)
```
Stages are logged at Debug level - not visible to operator unless log level is set to Debug.

### Current event consumer
`internal/infra/commands/cmd_process.go:131-141`
```go
go func() {
    defer close(eventsDone)
    for e := range events {
        switch e.Type {
        case pipeline.EventUnitDone:
            logger.Debug("unit done", "id", e.ID)
        case pipeline.EventUnitFailed:
            logger.Warn("unit failed", "id", e.ID, "error", e.Error)
        }
    }
}()
```
Events are consumed silently (Debug/Warn level). No status bar wiring.

## External Research Findings

### Go terminal status bar patterns
1. **ANSI escape sequences** - The simplest approach: use `\r` to overwrite the same line, `\x1b[2J` to clear, `\x1b[?` for cursor positioning. No external dependency needed.
   - Pattern: `fmt.Printf("\r%s\n", status)` - overwrites previous line
   - Countdown: `fmt.Printf("\rscanning...\n")` to `fmt.Printf("\rdormant: %02d:%02d:%02d\n", remaining)`
   - Source: Go stdlib, no external libs required. Common pattern in CLI tools like `rsync`, `git`, `go build`.

2. **bubbletea/v2** - Already a dependency in the project. The TUI model exists but is not wired. Could be extended to show a status bar, but it's a full TUI framework that blocks stdin. For a simple status line, it's overkill and would change the interaction model (user can't type commands while TUI is active).

3. **termenv / termios** - Third-party libs for terminal detection (TTY vs pipe). Not needed if we assume TTY or accept that `\r` works even in pipes (no harm).

### Recommended approach
- **ANSI escape sequences via stdlib** - minimal, no new dependencies, compatible with existing output. Use `\r` to keep status on a single line, separate from structured log output.
- **Status bar as a separate output channel** - not mixed with slog. Write to a dedicated writer (could be os.Stdout or a configurable writer).
- **Countdown timer** - use `time.Since(startTime)` or `time.Until(deadline)` to compute remaining time. Format as HH:MM:SS.

## FAR Scale Output

### Factual Score: 5
- All code references verified in actual files. Pipeline loop, RunOnce, event consumer, logging infrastructure all confirmed. Dormancy mechanism is real (dormancyPeriod config).

### Actionable Score: 4
- Clear path identified: ANSI `\r` + separate status writer + countdown timer. No external libraries needed. Can proceed to implementation planning.

### Relevant Score: 5
- Directly addresses the problem. All findings map to the specific code surfaces that need modification.

### Mean: 4.67  --> PASS

## Testing Strategy

- **Unit test touchpoints**: Status bar writer (formatting, countdown logic). Mock the pipeline loop to verify status output during dormancy.
- **Integration surfaces**: Wire status bar into Pipeline.Run() and Pipeline.RunOnce(). Verify output appears during active processing and dormancy.
- **Repro steps**: Run `lazyam3 process --once` (should show status during active work). Run without `--once` (should show status bar cycling between "scanning..." and "dormant: HH:MM:SS").
- **Edge cases**: Non-TTY output (pipes), rapid stage transitions, context cancellation during status update.

## Potential Design Pattern Recommendations

1. **StatusWriter interface** - A simple interface `StatusWriter` with `Write(status string, countdown time.Duration)`. Injected into Pipeline. Minimal abstraction, easy to swap.
2. **Event-driven status** - Add `EventStatusUpdate` to PipelineEvent. The event consumer in cmd_process.go could drive the status bar. This keeps status logic in the presentation layer.
3. **Inline status in RunOnce** - Emit status updates at key points: scan, ingest, metadata, script build, render. Use a local StatusWriter that tracks elapsed time.
4. **Countdown via time.Timer** - Use `time.Now()` and `dormancyPeriod` to compute remaining time. Format with `time.Since()` or `time.Until()`.

## Assumptions

- Terminal output is desired (not file-only logging). If terminal logging is disabled, status bar may not appear.
- TTY detection is not required - ANSI `\r` works in pipes (no visual harm, just overwrites).
- Status bar should be a separate output from structured logs - not mixed into slog output.
- The status bar is a CLI-level concern, not a TUI concern (bubbletea is not needed).

## Out of Scope

- Full TUI implementation (bubbletea integration)
- Metrics/alerting integration (slog/Prometheus)
- Config changes for status bar visibility (assume it's always on when terminal is enabled)
- Multi-line status bars or progress bars (single-line status only)
- Color coding for different stages (keep it simple)
