# Progress Log

## Session: Config, Logging, and Feature Planning

### Work Completed

#### A2 — Logger.Group() helper (done)
- Added `func (l *Logger) Group(name string) *Logger` to `internal/infra/logging/logger.go`
- Preserves typed `*Logger` surface (unlike `slog.Logger.WithGroup()`)
- Migrated `cmd_process.go` off `Slog()` escape hatch
- Pipeline now uses `pipeline.Logger` interface (both `*slog.Logger` and `*logging.Logger` satisfy it)

#### A3 — JSON vs text format toggle (done)
- Added `Format string` to `config.Logging`
- `newTermHandler` switches on `strings.EqualFold(cfg.Format, "text")`
- Text mode → `slog.NewTextHandler`; default → `slog.NewJSONHandler`
- File handler stays JSON (machine-consumed)
- Terminal-only tests capture `os.Stderr` via pipe

#### A4 — Color support for terminal logs (done)
- Added `Color bool` to `config.Logging`
- Implemented `colorLevelWriter` — `io.Writer` wrapper around `os.Stderr` that colorizes the level field
- Colors: DEBUG=cyan, INFO=green, WARN=yellow, ERROR=red, reset after level token
- Only effective when `format = "text"`; JSON output ignored
- No new dependencies — raw ANSI escapes
- ReplaceAttr approach rejected (TextHandler quotes control chars via strconv.Quote)

#### C1 — Config package refactoring (done)
- `Logger.Path` default: `"/path/to/file"` → `""` (empty disables file logging, already supported)
- Executor map seeded with 6 keys mirroring `buildScriptConfig` fallbacks:
  - `target → root/target`, `buffer/in_progress/done/notifications → root`, `archive → root/archive`
- `Validate()` added (configmanager.Validator hook):
  - `logger.level` whitelist: `debug`, `info`, `warn`, `error`
  - Generator paths non-empty: `input_directory`, `output_directory`, `metadata_path`
- 7 tests added/updated

#### Feature Planning
- **A5** — Custom slog text handler with format template (proposed, priority 5)
  - Analysis saved to `doc/custom_slog_handler_proposal.md`
  - 4 options analyzed; Option 1 (custom handler + basic template) recommended
- **S1** — Config struct refactor: ScriptGeneration block (in_rewiew, priority 4)
  - Groups Hardsub + FfmpegBinary under one config block
  - Iterative: start with Hardsub + FfmpegBinary, add more later
- **S2** — Make ffmpeg binary configurable (proposed, priority 4, blocked by S1)
  - Currently hardcoded in `template.go:43`
  - Dataflow: config → Pipeline.Options → ScriptConfig → template placeholder

#### Documentation
- Created `doc/custom_slog_handler_proposal.md` — A5 analysis with 4 options and effort/gain balance
- Created `doc/feature_index.md` — sync table from feature_list.json (Code, Status, Priority, Title, Blocked by)
- Created `doc/HANDOFF.md` — knowledge to pass to next session
- Updated `AGENTS.md` — current feature, new docs in documentation index

### Key Technical Findings

1. **TOML map merge semantics:** `go-toml/v2` merges decoded maps into existing seeded maps — missing keys keep seed values. Partial executor sections fill missing keys from seeds. Verified empirically.

2. **DiscoverAndLoad swallows validation errors:** Falls through to standard paths (`~/.config/...`). Invalid explicit configs silently load user's real config. Only surfaces when no candidate loads. This is a `configmanager` library behavior.

3. **Color via post-processing:** `colorLevelWriter` works because TextHandler writes one `Write()` per record. ReplaceAttr cannot pass ANSI codes (strconv.Quote escapes control chars).

4. **First-run crash fixed:** `Logger.Path = "/path/to/file"` caused `os.OpenFile("/path/to/file")` → permission denied on `/` for non-root users. Fresh installs now work with `Path = ""`.

### Metrics

- **Lines changed:** ~120 lines of code, ~80 lines of tests
- **Features completed:** 5 (A2, A3, A4, C1, BUG-1)
- **Features added:** 3 (A5, S1, S2)
- **Tests added:** 9 (across config and logging packages)
- **Gates passed:** `go build`, `go test -race -cover`, `go vet`, `golangci-lint run` — all clean

## Session: S1 — Config struct refactor: ScriptGeneration block

### Work Completed

#### S1 — ScriptGeneration config block (done)
- New `ScriptGeneration` struct in `internal/infra/config/config.go`:
  ```go
  type ScriptGeneration struct {
      Hardsub      Hardsub `toml:"hardsub"`
      FfmpegBinary string  `toml:"ffmpeg_binary,omitempty"`
  }
  ```
- `Hardsub` moved from top-level `Config` into `ScriptGeneration.Hardsub`
- `FfmpegBinary` added (default: `"ffmpeg"`)
- TOML format changed: `[hardsub]` → `[script_generation.hardsub]` with `tag` + `ffmpeg_binary`
- Breaking change: no backward compat — old `[hardsub]` section ignored by `go-toml/v2`
- All accessors updated: `cfg.Hardsub.Tag` → `cfg.ScriptGeneration.Hardsub.Tag`
- `cmd_process.go` line 100: `HardsubTag: cfg.ScriptGeneration.Hardsub.Tag`
- Default values: `Hardsub.Tag = "Forsirovannye"`, `FfmpegBinary = "ffmpeg"`
- 4 tests added: `TestScriptGeneration_HardsubTag`, `TestScriptGeneration_FfmpegBinary`, `TestScriptGeneration_TOMLNewFormat`, `TestDefaultConfig_ScriptGeneration`
- `~/.config/lazyam3/config.toml` migrated manually
- `scripts/live_test.sh` created — end-to-end pipeline test script

### Key Technical Findings

1. **configmanager doesn't support UnmarshalTOML:** `go-toml/v2` (used by configmanager) ignores custom `UnmarshalTOML` methods. Custom TOML handling must be done in `Load()` after `configmanager` returns, or via struct fields with TOML tags.

2. **S1+S2 prepare template for configurable ffmpeg:** `ScriptGeneration.FfmpegBinary` field added in S1; S2 will wire it through pipeline → ScriptConfig → template placeholder.

3. **Breaking config change:** `[hardsub]` → `[script_generation.hardsub]`. User config at `~/.config/lazyam3/config.toml` updated manually.

### Metrics

- **Lines changed:** ~30 lines of code, ~40 lines of tests
- **Gates passed:** `go build`, `go test -race -cover`, `go vet`, `golangci-lint run` — all clean
