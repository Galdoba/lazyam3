# Custom slog Text Handler Proposal

## Problem

`slog.TextHandler` has no format customization options — it always emits `key=value` pairs separated by spaces, newline-terminated. The only knobs are `Level` (threshold) and `HandlerOptions`.

## Current Architecture

```
slog → multiHandler → slog.TextHandler → os.Stderr → (optionally wrapped by) colorLevelWriter
```

`colorLevelWriter` is a post-processing `io.Writer` wrapper that scans each line for `level=XXX` and injects ANSI escapes. This works because `TextHandler` writes one `Write()` call per log record.

## Options

### Option 1: Custom `slog.Handler` with template (Recommended)

Write a replacement handler implementing `slog.Handler`, rendering a format string:

```toml
[logger]
format = 'text'
format_text = '{time} [{level}] {msg} {attrs}'
```

**How it works:**
- Parse template into segments (literal text + `{time}`, `{level}`, `{msg}`, `{attrs}` placeholders)
- `Handle()` renders each segment against the `slog.Record`
- `WithAttrs()` accumulates attrs in the handler; `WithGroup()` adds a key prefix
- `Enabled()` compares level; `Attrs()` iterates over accumulated attrs + record attrs

**Effort: ~100 lines of new code**

| Component | Lines |
|---|---|
| Template parsing | ~20 |
| `Handle()` rendering | ~40 |
| `WithAttrs()` / `WithGroup()` | ~25 |
| Tests | ~15 |

**Pros:**
- Full control over output format (field order, separators, quoting, etc.)
- `colorLevelWriter` already works — it's a post-processing wrapper on `io.Writer`
- No new dependencies (just `fmt`, `strings`, `time`)
- File handler stays JSON (unaffected)

**Cons:**
- Replaces `slog.TextHandler` entirely — lose any future upstream improvements
- Template parsing is basic (no conditionals, per-field formatting)

### Option 2: `ReplaceAttr` + post-processing (Low effort, limited)

Use `slog.HandlerOptions.ReplaceAttr` to modify attribute values before `TextHandler` writes them. Combined with the existing `colorLevelWriter` pattern.

**Effort: ~20 lines**

**Pros:**
- Minimal code
- Stays on top of `slog.TextHandler`

**Cons:**
- **Cannot change field order** (TextHandler's internal structure)
- **Cannot change separators** (always space)
- **Cannot add/remove fields** (only modify existing ones)
- Limited to attribute-level changes

### Option 3: Full template engine (High effort)

Use Go's `text/template` with a full record context:

```toml
format_text = '{{ .Time.Format "15:04:05" }} [{{ .Level }}] {{ .Message }}{{ range .Attrs }} {{ .Key }}={{ .Value }}{{ end }}'
```

**Effort: ~150-200 lines**

**Pros:**
- Maximum flexibility
- Leverages Go's built-in template system

**Cons:**
- Template parsing/rendering overhead per log record
- Overkill for the likely use cases

### Option 4: Format presets (Medium effort, user-friendly)

Predefined format names instead of raw templates:

```toml
format_text = 'compact'   # {time} [{level}] {msg}
format_text = 'detailed'  # {time} level={level} msg={msg} attrs={attrs}
format_text = 'raw'       # raw slog.TextHandler output
```

Plus a custom field for arbitrary templates:

```toml
format_text_template = '{time} | {level} | {msg}'
```

**Effort: ~80-120 lines**

**Pros:**
- User-friendly (most users pick a preset)
- Templates available for power users

**Cons:**
- More config fields to manage
- Preset names need to be chosen and documented

## Effort / Gain Balance

| Option | Effort | Gain | Complexity |
|---|---|---|---|
| 1. Custom handler + template | ~100 lines | High (full control) | Medium |
| 2. ReplaceAttr + post-process | ~20 lines | Low (attribute tweaks only) | Low |
| 3. Full template engine | ~150-200 lines | Very high | High |
| 4. Format presets | ~80-120 lines | Medium (preset covers most) | Medium |

## Recommendation

**Option 1** (custom handler with basic template) is the best balance:
- ~100 lines, no dependencies
- Sufficient for all reasonable use cases
- `colorLevelWriter` integration already proven
- Follows "simples solutions SHOULD examined first"

A minimal template set:
- `{time}` — timestamp
- `{level}` — level name
- `{msg}` — message
- `{attrs}` — all key=value pairs

That covers 95% of customization needs (field order, separators, quoting, timestamp format).
