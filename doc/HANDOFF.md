# HANDOFF — Knowledge for Next Session

## Current State

- **Just completed:** `D1` — Default configuration refactoring (status: `done`)
- **Current Feature:** `A5` — Custom slog text handler with format template (proposed, priority 5, no blockers)
- **Previous:** `S2` — Make ffmpeg binary configurable via app configuration (done)

## What's Done

| Feature | Status | Notes |
|---|---|---|
| D1 — Default configuration refactoring | done | Self-documenting defaults, `IsDefault`, `ErrDefaultConfig`, fail-fast startup |
| C1 — Config package refactoring | done | `Logger.Path` → `""`; executor map seeded with 6 keys; `Validate()` added |
| S1 — ScriptGeneration block | done | `Hardsub` moved into `ScriptGeneration.Hardsub`; `FfmpegBinary` added; breaking config change |
| S2 — ffmpeg binary configurable | done | `|=ffmpeg_binary=|` placeholder in template; fallback to `"ffmpeg"`; config-driven path |

## D1 Implementation Summary

**Goal:** built-in default config must be self-documenting placeholders; startup must fail fast when effective config equals the default.

**Changes:**
| File | Change |
|---|---|
| `internal/infra/config/config.go` | `defaultCFG()` now returns self-documenting placeholders (`/path/to/lazyam3/input`, `/path/to/lazyam3/output`, `/path/to/lazyam3/metadata.json`, `/path/to/lazyam3/storage.json`, `/path/to/lazyam3/lazyam3.log`, executor mirrors `/path/to/lazyam3/output/<key>`); added `ErrDefaultConfig` sentinel, `*DefaultConfigError{ConfigPath}`, `Config.IsDefault()` (reflect.DeepEqual vs `defaultCFG()`); `Load()` returns the typed error when the effective config equals the default |
| `internal/app/app.go` | `AppModules.Init` passes `config.ErrDefaultConfig` through unwrapped (fail fast before logger/pipeline) |
| `internal/infra/config/config_test.go` | Updated defaults tests; added `TestDefaultConfig_ContainsPlaceholders`, `TestIsDefault`, `TestLoad_DefaultConfigDetected`, `TestLoad_UserConfigNotDefault`, `TestLoad_ExplicitConfigEqualToDefault` |

**Error message (as implemented):**
```text
default configuration in use: edit the config file "<config file path>" and fill in the required fields (paths.generator.input_directory, paths.generator.output_directory, paths.generator.metadata_path) to enable processing
```

**Data flow:** `cmd/lazyam3` → Root `Before` → `AppModules.Init(configPath)` → `config.Load(configPath)` → `cfg.IsDefault()` → `*DefaultConfigError{ConfigPath}` → returned before logger creation / pipeline.

## Key Technical Findings (D1)

1. **configmanager `Manager.Path()` after `EnsureLoaded`:** on a fresh install the manager writes the default to the XDG path and `m.path` is then set to that file → the `DefaultConfigError.ConfigPath` points at the just-written default config file. `Load("")` uses `cm.EnsureLoaded()`; `Load(suggested)` uses `cm.DiscoverAndLoad(suggested)`.

2. **TOML seed-merge semantics:** absent keys keep seeded values (verified: user TOML without `[logger]` inherits seeded `Logger.Level="info"`, `Terminal=true`, `Format="text"`, `Color=true`, and now `Path="/path/to/lazyam3/lazyam3.log"`). This became critical with D1: a user config that fills only the required generator paths inherits the new placeholder log path → file handler fails at logger init. Known trade-off, see below.

3. **`reflect.DeepEqual` on `Config`:** works (all fields are strings/bools/maps of strings).

4. **DiscoverAndLoad swallows validation errors:** falls through to standard paths; invalid explicit configs silently load the user's real config. Pre-existing BUG-2, not fixed here.

## Known Issues / Footguns

### Default config detection misses partial defaults
Equality-based detection (`IsDefault`) only fires when the effective config *fully* equals `defaultCFG()`. A user config that sets real input/output/metadata paths but leaves `[logger]` absent inherits the placeholder `logger.path` → startup fails at logger creation with "failed to create file handler: failed to open /path/to/lazyam3/lazyam3.log". Not a default-config error, so fail-fast does not catch it. Accepted trade-off of D1 placeholders; watch when A5 lands.

### DiscoverAndLoad Validation Swallowing
`configmanager.DiscoverAndLoad` **swallows validation errors** and falls through to standard paths (e.g. `~/.config/lazyam3/config.toml`). An invalid explicit config silently loads the user's real config. Only surfaces when *no* candidate loads. This is a `configmanager` library behavior — not in scope to fix.

### Storage.Path Placeholder
Resolved by D1 — `defaultCFG()` now uses the self-documenting `/path/to/lazyam3/storage.json` placeholder.

### `--out` Flag Overrides Config
In `cmd_process.go`, the `--out` flag overrides `cfg.Paths.GeneratorOutput()`. If you pass `--out /some/path`, both source files AND scripts land there — not the buffer directory. Use without flags to use config value.

## Testing

### Live Test Script
`scripts/live_test.sh` — end-to-end pipeline test: Run before/after any pipeline change to verify end-to-end behavior.

### Test Suite
```bash
go build ./...
go test -race -cover ./...
go vet ./...
golangci-lint run ./...
```

## Files Changed This Session (D1)

| File | Change |
|---|---|
| `internal/infra/config/config.go` | Self-documenting `defaultCFG()` placeholders; `ErrDefaultConfig` + `*DefaultConfigError`; `Config.IsDefault()`; `Load()` default detection |
| `internal/app/app.go` | `AppModules.Init` fail-fast passthrough for `config.ErrDefaultConfig` |
| `internal/infra/config/config_test.go` | Updated defaults tests; added D1 tests (placeholders, IsDefault, fresh-install detection, user-config-not-default, explicit-default) |
| `doc/feature_list.json` | D1 status → done, outcome + tests updated |
| `doc/feature_index.md` | D1 status → done |
| `doc/BACKLOG.md` | BUG-3 resolved note |
| `doc/architecture.md` | Default fail-fast documented |

## What to Watch For

1. **A5** next: custom slog text handler. `colorLevelWriter` still works as post-processing wrapper; file handler stays JSON.
2. **Breaking configs:** S1 changed `[hardsub]` → `[script_generation.hardsub]`. Any user with old config gets default values until they update.
3. **D1 trade-off:** equality-based detection also flags user configs that intentionally mirror the defaults.
4. **`logger.path` placeholder:** with D1, a user config that omits `[logger]` inherits the placeholder and fails at logger init — known footgun above.

## Commands

```bash
go build ./...
go test -race -cover ./...
go vet ./...
golangci-lint run ./...
```

## Live Test
`scripts/live_test.sh` — passed end-to-end after D1 (6 units processed, 0 failed) with the real user config at `~/.config/lazyam3/config.toml`.
