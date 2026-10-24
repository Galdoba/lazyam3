# HANDOFF — Knowledge for Next Session

## Current State

- **Current Feature:** `S2` — Make ffmpeg binary configurable via app configuration (proposed, priority 4, no blockers)
- **Previous:** `S1` — Config struct refactor: ScriptGeneration block (done)
- **Also proposed:** `A5` — Custom slog text handler (priority 5, no blockers)

## What's Done

| Feature | Status | Notes |
|---|---|---|
| C1 — Config package refactoring | done | `Logger.Path` → `""`; executor map seeded with 6 keys; `Validate()` added |
| S1 — ScriptGeneration block | done | `Hardsub` moved from top-level to `ScriptGeneration.Hardsub`; `FfmpegBinary` added; breaking config change |

## What's Proposed

| Feature | Priority | Blocked By | Effort |
|---|---|---|---|
| S2 — ffmpeg binary configurable | 4 | — | 2-3 hours |

## S1 Implementation Summary (for S2 context)

S1 introduced `ScriptGeneration` struct grouping all script-generation settings:

```go
type ScriptGeneration struct {
    Hardsub      Hardsub `toml:"hardsub"`
    FfmpegBinary string  `toml:"ffmpeg_binary,omitempty"`
}
```

- `Hardsub` struct kept as standalone type, now only used within `ScriptGeneration`
- TOML format: `[script_generation]` → `[script_generation.hardsub]` → `tag` + `ffmpeg_binary`
- Breaking change: old `[hardsub]` section no longer works — users must migrate
- All accessors updated: `cfg.Hardsub.Tag` → `cfg.ScriptGeneration.Hardsub.Tag`
- `cmd_process.go` line 100: `HardsubTag: cfg.ScriptGeneration.Hardsub.Tag`
- Default values: `Hardsub.Tag = "Forsirovannye"`, `FfmpegBinary = "ffmpeg"`

**S1 tests:** `TestScriptGeneration_HardsubTag`, `TestScriptGeneration_FfmpegBinary`, `TestScriptGeneration_TOMLNewFormat`, `TestDefaultConfig_ScriptGeneration` (in `config_test.go`)

## S2 Plan (next feature)

**Goal:** Make ffmpeg binary path configurable instead of hardcoded literal in template.

**Files to touch:**
- `internal/infra/config/config.go` — `ScriptGeneration.FfmpegBinary` already exists (added in S1)
- `internal/app/pipeline/dto.go` — add `FfmpegBinary string` to `Options`
- `internal/app/pipeline/config.go` — pass `FfmpegBinary` through `buildScriptConfig`
- `internal/domain/script/config.go` — add `FfmpegBinary string` to `ScriptConfig`
- `internal/domain/script/template.go` — replace hardcoded `ffmpeg` with `|=ffmpeg_binary=|` placeholder
- `internal/infra/commands/cmd_process.go` — wire `cfg.ScriptGeneration.FfmpegBinary` → pipeline options

**Data flow:** config → Pipeline.Options.FfmpegBinary → buildScriptConfig → ScriptConfig.FfmpegBinary → template placeholder → rendered script

**Fallback:** empty string → `"ffmpeg"` (same pattern as ffprobe)

**Tests needed:**
- `TestBuildScriptConfig_FfmpegBinary` — custom path appears in generated script
- `TestBuildScriptConfig_FfmpegBinaryFallback` — default `"ffmpeg"` when not configured
- `TestBuildScriptConfig_FfmpegBinaryEmpty` — empty string falls back to `"ffmpeg"`
- `TestTemplate_FfmpegPlaceholder` — `|=ffmpeg_binary=|` replaced in rendered template

## Key Technical Decisions

### TOML Map Merge Semantics
`go-toml/v2` **merges** decoded maps into existing seeded maps — missing keys keep seed values. This means:
- Seeded executor paths become effective runtime values for configs lacking `[paths.executor]`
- Partial sections (e.g. only 5 of 6 keys) fill the missing key from seed
- `[hardsub]` is a breaking change — existing configs must migrate to `[script_generation.hardsub]` manualy

### Config Loading
`configmanager` uses `go-toml/v2` for TOML unmarshaling. **Does NOT support** `UnmarshalTOML` methods — uses standard `encoding/json`-style field mapping via TOML tags. Custom unmarshal hooks must be done in the `Load()` function after `configmanager` returns.

### Pipeline Architecture
```
cmd_process.go → Pipeline.Options → buildScriptConfig() → ScriptConfig → Build() → Renderer.Render()
```
- `p.destination` (from `cfg.Paths.GeneratorOutput()`) is where **both** source files and scripts land
- `prepareFiles()` moves source files to `p.destination`
- `Build()` writes scripts to `p.destination`
- Executor paths (`target`, `buffer`, etc.) are separate from `p.destination` — they're used by generated scripts at runtime

### S1 Seeding Strategy
Executor seeds are derived paths (chosen by user):
```
target:        /tmp/lazyam3/output/target
buffer:        /tmp/lazyam3/output
in_progress:   /tmp/lazyam3/output
done:          /tmp/lazyam3/output
notifications: /tmp/lazyam3/output
archive:       /tmp/lazyam3/output/archive
```
These mirror `buildScriptConfig` fallbacks for `root = Generator.OutputDirectory`. Trade-off: stale if user changes `output_directory` without updating executor keys.

## Known Issues / Footguns

### DiscoverAndLoad Validation Swallowing
`configmanager.DiscoverAndLoad` **swallows validation errors** and falls through to standard paths (e.g. `~/.config/lazyam3/config.toml`). An invalid explicit config silently loads the user's real config. Only surfaces when *no* candidate loads. This is a `configmanager` library behavior — not in scope to fix.

### Storage.Path Placeholder
`Storage.Path` still has `"/path/to/file.json"` in `defaultCFG()`. Same class as the `Logger.Path` bug (would fail on first `Store()` use). **Out of scope** for C1; flag for future.

### `--out` Flag Overrides Config
In `cmd_process.go`, the `--out` flag overrides `cfg.Paths.GeneratorOutput()`. If you pass `--out /some/path`, both source files AND scripts land there — not the buffer directory. Use without flags to use config value.

### configmanager Doesn't Support UnmarshalTOML
`go-toml/v2` (used by `configmanager`) does NOT call custom `UnmarshalTOML` methods. If you need custom TOML handling, do it in `Load()` after `configmanager` returns, or add struct fields with TOML tags that capture the legacy format.

## Testing

### Live Test Script
`scripts/live_test.sh` — end-to-end pipeline test:
1. Wipes all files in `~/workspace/lazyam3/` (keeps directories)
2. Copies `testdata/synthetic/` → `incoming/`
3. Builds lazyam3 and runs `process --once` (no flags — uses config)
4. Prints all files in the test ground

Run before/after any pipeline change to verify end-to-end behavior.

### Test Suite
```bash
go test -race -cover ./...   # all tests
go test -race -cover ./internal/infra/config/...  # config tests (13 tests)
```

## Files Changed This Session (S1)

| File | Change |
|---|---|
| `internal/infra/config/config.go` | `ScriptGeneration` struct added, `Hardsub` moved from top-level, `FfmpegBinary` added |
| `internal/infra/config/config_test.go` | 4 new tests for ScriptGeneration (HardsubTag, FfmpegBinary, TOMLNewFormat, DefaultConfig) |
| `internal/infra/commands/cmd_process.go` | `cfg.Hardsub.Tag` → `cfg.ScriptGeneration.Hardsub.Tag` |
| `doc/feature_list.json` | S1 status → done, S2 blocked_by → empty |
| `doc/feature_index.md` | S1 → done, S2 unblocked |
| `AGENTS.md` | Current feature → S2, live_test.sh documented |
| `~/.config/lazyam3/config.toml` | `[hardsub]` → `[script_generation.hardsub]` |

## What to Watch For

1. **S2** touches domain layer (`ScriptConfig`) — add one field, propagate through pipeline. Template placeholder `|=ffmpeg_binary=|` is straightforward. Ffmpeg binary path flows: config → Pipeline.Options → buildScriptConfig → ScriptConfig → template.
2. **A5** if chosen: custom handler replaces `slog.TextHandler`. `colorLevelWriter` still works as post-processing wrapper.
3. **configmanager limitation:** No `UnmarshalTOML` support. Any TOML custom parsing must be done in `Load()` after `configmanager` returns.
4. **Breaking configs:** S1 changed `[hardsub]` to `[script_generation.hardsub]`. Any user with old config will get default values (empty hardsub tag) until they update.

## Commands

```bash
go build ./...
go test -race -cover ./...
go vet ./...
golangci-lint run ./...
```
