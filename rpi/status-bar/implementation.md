# Status Bar Implementation

## Summary
Implemented a single-line status bar using ANSI `\r` escape sequences. StatusWriter interface added to pipeline dto.go, StdStatusWriter impl in platform/status.go. Status emits "scanning...", "processing: <dir>", "scan complete" during RunOnce, and "dormant: HH:MM:SS" countdown during dormancy loop. All tasks completed, build/lint/tests pass.

## Pull Requests
| PR | Description | Status |
|----|-------------|--------|
| [#1](https://github.com/Galdoba/lazyam3/pull/1) | Status bar for pipeline stages | Draft |

## Unexpected Discoveries
The status bar writes to stderr with `\r`, interleaving visually with slog logs on the same terminal line. This is expected ANSI overwrite behavior; structured logs remain intact in the log file and when piped.

## Plan Deviations
None. Implementation followed the plan exactly, with the interface placed in dto.go and impl in platform as specified.

## Architecture Decision Records
### StatusWriter interface location
context:
Pipeline uses ports for logging (Logger) and status (StatusWriter). Deciding where to define the StatusWriter interface.

options:
- Define interface in internal/platform/status.go
- Define interface in internal/app/pipeline/dto.go alongside Logger

decision:
Defined StatusWriter interface in internal/app/pipeline/dto.go alongside Logger.

rationale:
Consistent with existing Logger port placement; pipeline layer owns its ports, platform layer provides stateless implementations.

consequences:
- pipeline imports platform for StdStatusWriter concrete type
- platform implementation satisfies pipeline.StatusWriter interface via structural typing

## Handoff Notes

### Known Limitations
- Status bar uses ANSI `\r` which visually overwrites the last terminal line; interleaves with slog output on TTY.
- No TTY detection; `\r` is harmless in non-TTY pipes.

### Data Models
- StdStatusWriter: concrete status writer, no fields, writes to stderr.

### API Surface
- Write(status string): writes single-line status to stderr with `\r` prefix.

## Reference
- internal/app/pipeline/dto.go
- internal/app/pipeline/pipeline.go
- internal/platform/status.go
- internal/infra/commands/cmd_process.go
