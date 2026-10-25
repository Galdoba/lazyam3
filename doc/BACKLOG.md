# Backlog

## Known Issues / Footguns

### BUG-2 — DiscoverAndLoad Swallows Validation Errors
**Severity:** medium · **Source:** `configmanager` library behavior
`configmanager.DiscoverAndLoad` **swallows validation errors** and falls through to standard paths (e.g. `~/.config/lazyam3/config.toml`). An invalid explicit config silently loads the user's real config. Only surfaces when *no* candidate loads. This is a `configmanager` library behavior — not in scope to fix from lazyam3.

**Workaround:** Users must ensure their explicit config paths are valid. For debugging, isolate HOME to prevent fallback.

