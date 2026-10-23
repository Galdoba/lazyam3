# Backlog

## Known Issues / Footguns

### BUG-2 — DiscoverAndLoad Swallows Validation Errors
**Severity:** medium · **Source:** `configmanager` library behavior
`configmanager.DiscoverAndLoad` **swallows validation errors** and falls through to standard paths (e.g. `~/.config/lazyam3/config.toml`). An invalid explicit config silently loads the user's real config. Only surfaces when *no* candidate loads. This is a `configmanager` library behavior — not in scope to fix from lazyam3.

**Workaround:** Users must ensure their explicit config paths are valid. For debugging, isolate HOME to prevent fallback.

### BUG-3 — Storage.Path Placeholder
**Severity:** medium · **Source:** `defaultCFG()` in `internal/infra/config/config.go`
`Storage.Path` still has `"/path/to/file.json"` in `defaultCFG()`. Same class as the `Logger.Path` bug — would fail on first `Store()` use with permission denied. **Out of scope** for C1; flag for future.

### BUG-4 — ffmpeg Hardcoded in Template
**Severity:** low · **Source:** `internal/domain/script/template.go:43`
`ffmpeg` binary path is hardcoded as a literal in the template. Users cannot specify a custom ffmpeg binary (e.g., /opt/bin/ffmpeg, nvidia-enabled build). ffprobe IS configurable (`paths.generator.ffprobe_binary`), but ffmpeg (used in generated scripts) is not.

**Fix planned:** S1 + S2 — ScriptGeneration config block + configurable `FfmpegBinary` field.

## Deferred Items

- **Language translation** — deferred (see BACKLOG.md reference in AGENTS.md)
- **Source file naming hook** — deferred (see BACKLOG.md reference in AGENTS.md)
