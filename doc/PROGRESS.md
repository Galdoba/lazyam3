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

#### D1 — Default configuration refactoring (done)
- **Problem:** built-in default config used usable-looking paths (`/tmp/lazyam3/input`) or dummy placeholders (`/path/to/file.log`, `/path/to/file.json`); fresh installs silently ran on unusable defaults and users had no idea which fields existed.
- **Solution:**
  - `defaultCFG()` refactored to self-documenting placeholders: `/path/to/lazyam3/input`, `/path/to/lazyam3/output`, `/path/to/lazyam3/metadata.json`, `/path/to/lazyam3/storage.json`, `/path/to/lazyam3/lazyam3.log`, executor mirrors `/path/to/lazyam3/output/<key>`.
  - `Config.IsDefault()` added (reflect.DeepEqual against `defaultCFG()`).
  - Typed error `*DefaultConfigError{ConfigPath}` wrapping sentinel `ErrDefaultConfig` added.
  - `Load()` returns the default-config error when the effective config equals the default; `AppModules.Init` fails fast (unwrapped) before logger/pipeline creation.

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
- **Features completed:** 5 (A2, A3, A4, C1)
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

## Session: S2 — Make ffmpeg binary configurable via app configuration

### Work Completed

#### S2 — Ffmpeg binary configurable (done)

- **Problem:** ffmpeg binary path was hardcoded in template (`ffmpeg -n -r 25 -i ...`)
- **Solution:** Config-driven path via `ScriptGeneration.FfmpegBinary` (already existed from S1)
- **Data flow:** config → Pipeline.Options.FfmpegBinary → Pipeline.ffmpegBinary → buildScriptConfig → ScriptConfig.FfmpegBinary → template placeholder → rendered script
- **Fallback:** empty string → `"ffmpeg"` (same pattern as ffprobe)

**Files changed:**
- `internal/domain/script/config.go` — added `FfmpegBinary string` to `ScriptConfig`
- `internal/domain/script/template.go` — replaced hardcoded `ffmpeg` with `|=ffmpeg_binary=|` placeholder
- `internal/domain/script/script.go` — added placeholder replacement with fallback
- `internal/app/pipeline/dto.go` — added `FfmpegBinary string` to `Options`
- `internal/app/pipeline/pipeline.go` — added `ffmpegBinary` field to `Pipeline`, wired from options
- `internal/app/pipeline/config.go` — added `ffmpegBinary` parameter to `buildScriptConfig`
- `internal/infra/commands/cmd_process.go` — wired config → pipeline with `nonEmpty` fallback
- `internal/app/pipeline/pipeline_test.go` — updated existing tests, added 3 new tests
- `internal/domain/script/script_test.go` — added 2 new tests

**Tests added:**
- `TestBuildScript_FfmpegBinary` — custom path `/opt/bin/ffmpeg-nvenc` in generated script
- `TestBuildScript_FfmpegBinaryFallback` — empty FfmpegBinary falls back to `"ffmpeg"`
- `TestBuildScriptConfig_FfmpegBinary` — custom path stored in ScriptConfig
- `TestBuildScriptConfig_FfmpegBinaryFallback` — empty FfmpegBinary in config
- `TestBuildScriptConfig_FfmpegBinaryEmpty` — empty config produces valid script with fallback

**Metrics:**
- **Lines changed:** ~50 lines of code, ~100 lines of tests
- **Gates passed:** `go build`, `go test -race -cover`, `go vet`, `golangci-lint run` — all clean
- **Live test:** passed end-to-end pipeline

## Session: D1 — Default configuration refactoring

### Work Completed

#### D1 — Default configuration refactoring (done)
- **Problem:** built-in default config used usable-looking paths (`/tmp/lazyam3/input`) or dummy placeholders (`/path/to/file.log`, `/path/to/file.json`); fresh installs silently ran on unusable defaults and users had no idea which fields existed.
- **Solution:**
  - `defaultCFG()` refactored to self-documenting placeholders: `/path/to/lazyam3/input`, `/path/to/lazyam3/output`, `/path/to/lazyam3/metadata.json`, `/path/to/lazyam3/storage.json`, `/path/to/lazyam3/lazyam3.log`, executor mirrors `/path/to/lazyam3/output/<key>`.
  - `Config.IsDefault()` added (reflect.DeepEqual against `defaultCFG()`).
  - Typed error `*DefaultConfigError{ConfigPath}` wrapping sentinel `ErrDefaultConfig` added.
  - `Load()` returns the default-config error when the effective config equals the default; `AppModules.Init` fails fast (unwrapped) before logger/pipeline creation.

#### D1 — Error message
```
default configuration in use: edit the config file "<config file path>" and fill in the required fields (paths.generator.input_directory, paths.generator.output_directory, paths.generator.metadata_path) to enable processing
```

### Tests Added/Updated
- `TestDefaultConfig_ContainsPlaceholders` — defaults are self-documenting placeholders.
- `TestIsDefault` — detects effective config equal to the built-in default; edited config → false.
- `TestLoad_DefaultConfigDetected` — fresh install (no user config) → `ErrDefaultConfig` with `ConfigPath` = `<home>/.config/lazyam3/config.toml`.
- `TestLoad_UserConfigNotDefault` — user-edited config loads normally.
- `TestLoad_ExplicitConfigEqualToDefault` — explicit default-equal config → `ErrDefaultConfig`.
- Updated: `TestDefaultConfig` (new placeholder expectations), `TestDefaultConfig_LoggerPath`, `TestDefaultConfig_ExecutorMapSeeded`.

### Key Technical Findings
1. configmanager `Manager.Path()` after `EnsureLoaded` reflects the just-written default path → `DefaultConfigError.ConfigPath` points to the fresh-install default file.
2. TOML absent keys keep seeded values (struct/map merge) — user configs omitting `[logger]` inherit seeded `Logger.Path` placeholder → file handler fails later at logger init.

### Metrics
- Lines changed: ~90 in `internal/infra/config`, ~5 in `internal/app`, ~200 in tests.
- Features completed this session: `D1` — Default configuration refactoring (plus previously done: A2, A3, A4, C1, BUG-1, S1, S2).
- Gates passed: `go build`, `go test -race -cover`, `go vet`, `golangci-lint run` — all clean.
- Live test: passed end-to-end after D1 (6 units processed, 0 failed).

### What to Watch For
- Next feature: `A5` — Custom slog text handler with format template (proposed, priority 5, no blockers).
- D1 trade-off: equality-based detection also flags user configs that intentionally mirror the defaults.

### S3: Configurable hardsub parameters (in_review)

#### Changes
- `doc/feature_list.json` — corrected S3 `tests` array to list actual test names.
- `internal/domain/script/script_test.go` — added `TestHSUBConfigValueChangesOutput`
  asserting that changing `HarsubConfig` fields produces different `force_style` output
  (tests `MarginV` and `FontName`).

#### Gates
- `go build ./...` — pass
- `go test -race -cover ./...` — pass (domain/script 98.0%)
- `go vet ./...` — pass
- `golangci-lint run` — 0 issues

#### Notes
- S3 is ready to commit. The feature_list.json `status` is already `done`.
- Next feature: `S4` — Configurable ffmpeg/filter command parameters (in_review).
