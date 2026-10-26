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

### Pipeline loop: poll interval + ticker (this session)

#### Changes
- `internal/app/pipeline/pipeline.go` — `Run` now waits via `time.NewTicker(p.pollInterval)`; ad-hoc `afterPoll` channel + `debugTumeout` sleep removed; `defaultPollInterval = 1s`.
- `internal/app/pipeline/dto.go` — `Options.PollInterval` added (0 → default).

#### Tests
- Added `TestNewPipeline_PollIntervalDefault`, `TestRun_LoopsUntilContextCancelled`; full suite green.

#### Gates
- `go build ./...`, `go vet ./...`, `gofmt -l`, `golangci-lint run ./...` → clean; `go test -race ./...` → ok.

## Session: D1 — Dormancy period for cyclic mode

### Work Completed

#### D1 — Dormancy period for cyclic mode (done)

- **Problem:** The cyclic mode polls every 1 second even when the pipeline is idle, wasting CPU cycles. No concept of "process once, then rest for N minutes."
- **Solution:** Configurable dormancy period after each scan pass. Zero value preserves current polling behaviour.

**Config:**
- `internal/infra/config/config.go` — new `PipelineConfig` struct with `DormancyPeriod int` (seconds); `Config` gains `Pipeline` field; `defaultCFG()` sets `DormancyPeriod: 0`
- TOML: `[pipeline]` section with `dormancy_period = 10` (integer seconds)

**Pipeline:**
- `internal/app/pipeline/dto.go` — `DormancyPeriod time.Duration` added to `Options`
- `internal/app/pipeline/pipeline.go` — `Pipeline.dormancyPeriod` field; `NewPipeline` copies from options; `Run()` sleeps `dormancyPeriod` after each pass when > 0 (cancellable via context), falls back to polling ticker when == 0

**Wiring:**
- `internal/infra/commands/cmd_process.go` — `--dormancy` flag (Duration); falls back to `cfg.Pipeline.DormancyPeriod * time.Second`

**Files changed:**
- `internal/infra/config/config.go` — `PipelineConfig` struct, `Pipeline` field on `Config`, `defaultCFG` update
- `internal/app/pipeline/dto.go` — `DormancyPeriod` on `Options`
- `internal/app/pipeline/pipeline.go` — `dormancyPeriod` field, `NewPipeline` wiring, `Run()` dormancy logic
- `internal/infra/commands/cmd_process.go` — `--dormancy` flag, config wiring with int→Duration conversion
- `internal/app/pipeline/pipeline_test.go` — 5 new tests
- `doc/feature_list.json` — D1 feature entry
- `AGENTS.md` — current feature updated

**Tests added:**
- `TestNewPipeline_DormancyPeriodDefault` — zero value means disabled
- `TestRun_DormancyMode_SleepsBetweenPasses` — verifies sleep between passes with fast duration
- `TestRun_DormancyMode_ContextCancelledDuringSleep` — context cancellation during dormancy exits loop
- `TestRun_DormancyDisabled_Polling` — zero dormancy uses polling ticker
- `TestRun_DormancyMode_ContinuesAfterScanFailure` — sleep happens even after failed passes

**Live test:**
- Config loads correctly: `[pipeline]` section with `dormancy_period = 10` parses as int
- Cyclic mode: scans 10 seconds apart (20:33:10 → 20:33:20) confirmed via debug timestamps
- `--once` mode unaffected (bypasses `Run()`)

**Metrics:**
- **Lines changed:** ~60 lines of code, ~200 lines of tests (including test file additions)
- **Gates passed:** `go build`, `go test -race -cover`, `go vet`, `golangci-lint run` — all clean
- **Live test:** passed end-to-end with config-driven 10s dormancy

### What to Watch For
- Next feature: `S4` — Configurable ffmpeg/filter command parameters (in_review).
- `--dormancy` flag overrides config value; flag takes precedence.

## Session 2026-10-26 — A5: Custom slog text handler with format template

**Feature:** `A5` (`doc/text_logger_format_refactoring.md`) — status moved `proposed` → `in_review`.

**Evaluation findings (before implementation):**
- Proposal contradicted itself on `{level}` rendering (§5 `[INFO]` vs §11 `[{level}]`), on test #4 vs #5 (group prefix kept vs stripped), and on whether the desired format was a default or opt-in (§1 vs §3.6/§8).
- §6 regex draft had no capture groups while `colorizeLevel()` indexes `loc[4]:loc[5]` → would have panicked; also couldn't anchor `[INFO]` behind a space-containing timestamp.
- Gaps: empty-placeholder separator behavior, `format_text`×`format="json"` gating, handler concurrency (struct-level buffer = data race).

**Decisions confirmed with user:**
1. Ship the desired format as the new default (configmanager merges TOML over `defaultCFG()` — verified in appcontext source; missing keys inherit defaults).
2. `{level}` = bare level name, brackets literal in template.
3. Empty placeholders swallow adjacent space/colon-only literals.

**Implemented:**
- `config.Logging` += `FormatText`, `TimestampFormat` (+ defaults: `"{time} [{level}] {module}: {message} {attrs}"`, `"06-01-02 15:04:05.000"`).
- `customTextHandler` in `internal/infra/logging/logger.go`: template parsed once, immutable handler state, local buffer in `Handle`, group stack → `{module}`, attrs unprefixed + quoted strings, nested `slog.Group` flattening, `Resolve()`d values, in-handler level colorization (fallback path keeps `colorLevelWriter` untouched).
- Spec doc rewritten with decisions/deviations; `feature_list.json` A5 synced (source now points at the real doc, 18 tests listed); `feature_index.md`, `AGENTS.md`, `HANDOFF.md` updated.

**Tests added (18):** 14 in `logging` (`TestCustomHandler_*` ×13, `TestNewTermHandler_*` ×3 — exact-line render, attrs quoting, module/nesting, prefix stripping, color, layouts, swallowing, reordering, unknown placeholder, nested group, concurrency, e2e, JSON gating, fallback), 2 in `config` (`TestDefaultConfig_LoggerFormatText`, `TestLoad_FormatTextInheritedWhenMissing`).

**Metrics:**
- Gates: `go build` ✓ · `go test -race -cover ./...` ✓ (logging 93.2%, config 84.8%) · `go vet` ✓ · `golangci-lint` → 106 issues = pre-existing baseline (`//log=` comment noise in untouched files); **0 new issues** (stash-diff verified).
- Live test: passed — 6/6 units, terminal `26-10-26 12:10:18.812 [INFO] pipeline: processing dir="..."`, file log unchanged JSON.
- Lines changed: ~230 lines of code, ~380 lines of tests.

**What to watch for:**
- Non-string attrs with spaces render unquoted (`error=product 63580 not found`) — per spec, but ambiguous for parsers.
- Lint gate is red repo-wide even on clean checkout (gci/gocritic on `//log=` placeholders) — pre-existing, out of A5 scope.
- Priority numbering in `feature_list.json` is ambiguous (2 "higher" than 4?) — next-feature selection needs clarification.
