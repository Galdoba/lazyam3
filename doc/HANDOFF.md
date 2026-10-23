# HANDOFF — Knowledge for Next Session

## Current State

- **Current Feature:** `S1` — Config struct refactor: ScriptGeneration block (proposed, priority 4, no blockers)
- **Next after S1:** `S2` — Make ffmpeg binary configurable (blocked by S1)
- **Also proposed:** `A5` — Custom slog text handler (priority 5, no blockers)

## What's Done

| Feature | Status | Notes |
|---|---|---|
| A2 — Logger.Group() | done | `func (l *Logger) Group(name string) *Logger` — preserves typed *Logger surface |
| A3 — JSON/text format toggle | done | `Format string` on `config.Logging` — text for terminal, JSON for file |
| A4 — Color support | done | `Color bool` on `config.Logging` — `colorLevelWriter` post-processes os.Stderr, only for text mode |
| C1 — Config package refactoring | done | `Logger.Path` fixed to `""`; executor map seeded with 6 keys; `Validate()` added |
| BUG-1 — Env.TargetRoot regression | done | Fixed: `TargetRoot: scriptRoot → renderRoot` |

## What's Proposed

| Feature | Priority | Blocked By | Effort |
|---|---|---|---|
| S1 — ScriptGeneration block | 4 | — | 1 hour |
| S2 — ffmpeg binary configurable | 4 | S1 | 2-3 hours |
| A5 — Custom slog text handler | 5 | — | 2-3 hours |

## Key Technical Decisions

### TOML Map Merge Semantics
`go-toml/v2` **merges** decoded maps into existing seeded maps — missing keys keep seed values. This means:
- Seeded executor paths become effective runtime values for configs lacking `[paths.executor]`
- Partial sections (e.g. only 5 of 6 keys) fill the missing key from seed
- **Implication for S1:** When moving Hardsub into ScriptGeneration, TOML merge will handle both `[hardsub]` (legacy) and `[script_generation.hardsub]` (new) via configmanager's merge semantics — no custom migration needed

### Color Implementation
`colorLevelWriter` wraps `os.Stderr` as an `io.Writer` that intercepts each `Write()` call and injects ANSI escapes into the level field. This works because `slog.TextHandler` writes one `Write()` per record. **Cannot** use `ReplaceAttr` — `TextHandler` quotes control chars via `strconv.Quote`, so ANSI codes would be escaped as literal `\x1b`.

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

### ffmpeg Hardcoded in Template
`internal/domain/script/template.go:43` has `ffmpeg` as a literal. S1+S2 will make it configurable.

## Files Changed This Session

| File | Change |
|---|---|
| `internal/infra/config/config.go` | `Logger.Path` → `""`; executor seeded; `Validate()` added; `FfmpegBinary` added to `GeneratorPaths` |
| `internal/infra/config/config_test.go` | 7 new/updated tests |
| `doc/feature_list.json` | 8 features (A2-A5, BUG-1, C1, S1, S2) |
| `doc/feature_index.md` | Auto-generated sync table |
| `doc/custom_slog_handler_proposal.md` | A5 analysis with 4 options and effort/gain balance |
| `AGENTS.md` | Updated with current feature, new docs |

## What to Watch For

1. **S1** requires updating all `cfg.Hardsub.Tag` references to `cfg.ScriptGeneration.Hardsub.Tag`. grep for `Hardsub.Tag` to find them.
2. **S1** backwards compat: `[hardsub]` TOML key should still populate `ScriptGeneration.Hardsub` — configmanager merge semantics handle this naturally.
3. **S2** touches domain layer (`ScriptConfig`) — add one field, propagate through pipeline. Template placeholder `|=ffmpeg_binary=|` is straightforward.
4. **A5** if chosen: custom handler replaces `slog.TextHandler`. `colorLevelWriter` still works as post-processing wrapper.

## Commands

```bash
go build ./...
go test -race -cover ./...
go vet ./...
golangci-lint run ./...
```
