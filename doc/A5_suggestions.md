# A5 — Custom slog Text Handler with Format Template

## 1. Problem Statement

`slog.TextHandler` (Go standard library) has no format customization:
- Field order is fixed (time, level, msg, then attrs)
- Separator is always a single space
- Timestamp format is fixed (`2006/01/02 15:04:05.000`)
- No way to add/remove fields or change quoting
- `HandlerOptions.ReplaceAttr` can only modify attribute *values*, not structure

This limits users who want:
- Custom timestamp formats (e.g., `15:04:05` instead of full date)
- Different field layouts (e.g., `[level] message` without key=value attrs)
- Custom separators, quoting, or grouping
- Clean, concise log lines for terminal consumption

## 2. Current Architecture

```
slog.Logger
    ↓ multiHandler (internal/infra/logging/logger.go:202)
    ├── slog.JSONHandler   → file (internal/infra/logging/logger.go:184)
    └── slog.TextHandler   → os.Stderr (internal/infra/logging/logger.go:132)
                                └── colorLevelWriter (wraps os.Stderr, internal/infra/logging/logger.go:146)
```

**Key integration point**: `colorLevelWriter` (lines 146–153) is a post-processing `io.Writer` that scans each line for `level=XXX` and injects ANSI escapes. It works because `slog.TextHandler` writes one `Write()` call per log record.

**Config structure** (`config.Logging` at `internal/infra/config/config.go:122`):
```go
type Logging struct {
    Level    string `toml:"level"`
    Path     string `toml:"path"`
    Terminal bool   `toml:"terminal"`
    Format   string `toml:"format"` // "json" (default) or "text"
    Color    bool   `toml:"color"`    // colorize level in text output
}
```

## 3. Analysis of Options

### Option 1: Custom `slog.Handler` with basic template (RECOMMENDED)

Implement a replacement handler that renders a format string:

```toml
[logger]
format = "text"
format_text = "{time} [{level}] {module}: {message} {attrs}"
```

**How it works:**
1. Parse template string into segments: literals + `{time}`, `{level}`, `{message}`, `{attrs}` placeholders
2. `Handle()` renders each segment against `slog.Record`
3. `WithAttrs()` accumulates attrs; `WithGroup()` adds key prefix
4. `Enabled()` compares level
5. `colorLevelWriter` wraps the writer — post-processing on raw bytes still works

**Estimated effort: ~100 lines new code, ~100 lines tests**

| Component | Lines |
|---|---|
| Template parsing | ~15 |
| `Handle()` rendering | ~40 |
| `WithAttrs()` / `WithGroup()` | ~25 |
| `Enabled()` | ~10 |
| Tests | ~100 |

**Pros:**
- Full control over output format (field order, separators, quoting, timestamp format)
- `colorLevelWriter` still works (post-processing on `io.Writer`)
- File handler stays JSON (unaffected)
- No new dependencies (only `fmt`, `strings`, `time`)
- Minimal template set covers 95% of use cases

**Cons:**
- Replaces `slog.TextHandler` entirely — lose future upstream improvements
- Basic template (no conditionals, no per-field formatting)
- Additional code to maintain (~75 lines non-test)

### Option 2: `ReplaceAttr` + post-processing (rejected)

Use `slog.HandlerOptions.ReplaceAttr` to modify attribute values before `TextHandler` writes them.

**Why rejected:**
- Cannot change field order (TextHandler's internal structure is opaque)
- Cannot change separators (always space)
- Cannot add/remove fields
- `strconv.Quote` in TextHandler escapes control characters, so ANSI codes cannot be injected via `ReplaceAttr` (this is why `colorLevelWriter` exists as a writer-level wrapper)

### Option 3: Full `text/template` engine (overkill)

Use Go's `text/template` with a full record context:

```toml
format_text = "{{ .Time.Format \"15:04:05\" }} [{{ .Level }}] {{ .Message }}{{ range .Attrs }} {{ .Key }}={{ .Value }}{{ end }}"
```

**Why rejected:**
- Template parsing/rendering overhead per log record
- Overkill for the likely use cases (users want simpler output, not full templating)
- Harder to test (template errors are runtime, not compile-time)
- Adds complexity for marginal benefit

### Option 4: Format presets (secondary consideration)

Predefined format names instead of raw templates:

```toml
format_text = "compact"   # {time} [{level}] {message}
format_text = "detailed"  # {time} level={level} message={message} attrs={attrs}
```

**Assessment:** Could be layered on top of Option 1 as a convenience. Not a standalone approach — presets are just pre-written templates. Recommend implementing Option 1 first, then adding presets if users request them.

## 4. Recommended Implementation: Option 1

### 4.1 Template Design

**Minimal template set (5 placeholders):**

| Placeholder | Renders | Example |
|---|---|---|
| `{time}` | `record.Time.Format(timestampFormat)` | `2025/01/15 14:30:00` or custom |
| `{level}` | `record.Level.String()` | `INFO` |
| `{module}` | Module/context name (from logger group) | `pipeline` |
| `{message}` | `record.Msg` | `pipeline started` |
| `{attrs}` | All attrs as `key=value` pairs, space-separated | `key=value unit_id=42` |

**Timestamp format** (new config field):
```toml
[logger]
format = "text"
format_text = "{time} [{level}] {module}: {message} {attrs}"
timestamp_format = "15:04:05"  # Go time format, overrides default
```

**Why this set covers 95% of use cases:**
- Field order: user controls by reordering placeholders in template
- Separators: user controls by adding literal text between placeholders
- Timestamp format: configurable via `timestamp_format`
- Module context: `{module}` pulls from logger group or `With("module", ...)` attribute
- Quoting: attrs are unquoted (simple `key=value`), matching TextHandler's behavior

### 4.2 Data Flow

```
config.Logging
    ├── format = "text"              (existing)
    ├── format_text = "{time} [{level}] {module}: {message} {attrs}"  (NEW)
    ├── timestamp_format = "15:04:05"  (NEW)
    └── color = true                 (existing)
        ↓
newTermHandler(cfg)                  (internal/infra/logging/logger.go:117)
    ├── strings.EqualFold(cfg.Format, "text")
    │   └── customTextHandler(w, opts)  ← NEW
    │       └── colorLevelWriter(w)     ← existing, wraps writer
    └── slog.NewJSONHandler(os.Stderr, opts)
```

### 4.3 File Locations

| File | Change |
|---|---|
| `internal/infra/config/config.go` | Add `FormatText string` and `TimestampFormat string` fields to `Logging` struct; update `defaultCFG()` |
| `internal/infra/logging/logger.go` | Add `customTextHandler()` function; replace `slog.NewTextHandler` call in `newTermHandler` |
| `internal/infra/logging/logger.go` | Add `customHandler` type implementing `slog.Handler` |
| `internal/infra/logging/logger_test.go` | Add 6+ tests for custom handler |

### 4.4 `customHandler` Type Design

```go
// customHandler renders log records using a format template.
// It implements slog.Handler.
type customHandler struct {
    opts           HandlerOptions   // Level, timestamp format, template
    mu             sync.Mutex
    attrs          []slog.Attr      // accumulated attrs from WithAttrs
    groups         []string         // nested group names (prefix for keys)
    attrBuf        strings.Builder  // buffer for rendering attrs
    out            io.Writer
}

type HandlerOptions struct {
    Level           slog.Level
    TimestampFormat string
    Template        string        // e.g., "{time} [{level}] {message} {attrs}"
}
```

**Key methods:**
- `Enabled(ctx, level) bool` — returns true if `level >= opts.Level`
- `Handle(ctx, record) error` — renders template against record + accumulated attrs
- `WithAttrs(attrs) Handler` — returns new handler with accumulated attrs
- `WithGroup(name) Handler` — returns new handler with group name appended to stack

**Rendering logic (Handle):**
1. Build a slice of segments from template (parse once at init)
2. For each segment:
   - If literal: write as-is
   - If `{time}`: write `record.Time.Format(opts.TimestampFormat)`
   - If `{level}`: write `record.Level.String()`
   - If `{module}`: write logger module/group name (from `slog.WithGroup` or `With("module", ...)`)
   - If `{message}`: write `record.Msg`
   - If `{attrs}`: render all attrs (record attrs + accumulated + group-prefixed) as `key=value` pairs
3. Write final buffer to `out`

**Template parsing:**
- Parse template string once at handler creation
- Split on `{...}` into alternating literal/placeholder segments
- Cache parsed segments in `customHandler`

**Attrs rendering:**
- Collect: record's attrs (`record.Attributes()`) + accumulated (`h.attrs`) + group-prefixed versions
- Skip attrs whose keys are in `groups` (already prefixed)
- Render as `key=value` pairs, space-separated, no quoting

### 4.5 Config Changes

```go
// internal/infra/config/config.go

type Logging struct {
    Level             string `toml:"level"`
    Path              string `toml:"path"`
    Terminal          bool   `toml:"terminal"`
    Format            string `toml:"format"`          // "json" (default) or "text"
    Color             bool   `toml:"color"`           // colorize level in text output
    FormatText        string `toml:"format_text"`     // custom text template (NEW)
    TimestampFormat   string `toml:"timestamp_format"` // Go time format, e.g. "15:04:05" (NEW)
}
```

**Defaults** (in `defaultCFG()`):
- `FormatText = "{time} [{level}] {message} {attrs}"` (same layout as default TextHandler)
- `TimestampFormat = ""` (empty → use default Go format `2006/01/02 15:04:05.000`)

### 4.6 Wiring in `newTermHandler`

```go
func newTermHandler(cfg config.Logging) slog.Handler {
    if !cfg.Terminal {
        return nil
    }
    opts := &slog.HandlerOptions{
        Level: levelFromString(cfg.Level),
    }
    if strings.EqualFold(cfg.Format, "text") {
        var w io.Writer = os.Stderr
        if cfg.Color {
            w = &colorLevelWriter{w: w}
        }
        
        // Use custom handler if format_text is configured,
        // otherwise fall back to standard TextHandler.
        if cfg.FormatText != "" {
            return newCustomHandler(w, cfg.FormatText, cfg.TimestampFormat, opts.Level)
        }
        return slog.NewTextHandler(w, opts)
    }
    return slog.NewJSONHandler(os.Stderr, opts)
}
```

**Fallback behavior:** If `format_text` is not set but `format = "text"`, fall back to standard `slog.TextHandler` (preserves existing behavior). Custom handler is opt-in via `format_text`.

### 4.7 Test Plan

| # | Test | What it verifies |
|---|---|---|
| 1 | `TestCustomHandler_RenderBasic` | `{time} [{level}] {message} {attrs}` produces expected output |
| 2 | `TestCustomHandler_RenderAttrs` | `{attrs}` renders `key=value` pairs from record attrs |
| 3 | `TestCustomHandler_AccumulatedAttrs` | Attrs from `WithAttrs` appear in output |
| 4 | `TestCustomHandler_GroupPrefix` | `WithGroup("g")` prefixes attrs as `g.key=value` |
| 5 | `TestCustomHandler_ColorIntegration` | `colorLevelWriter` still colorizes when wrapping custom handler |
| 6 | `TestCustomHandler_TimestampFormat` | Custom `timestamp_format` changes time output |
| 7 | `TestCustomHandler_TemplateReordering` | `{message} {level} {time}` reverses field order |
| 8 | `TestNewTermHandler_CustomFormatText` | End-to-end: `newTermHandler` with `FormatText` returns `*customHandler` |
| 9 | `TestNewTermHandler_CustomFormatTextFallback` | Without `FormatText`, falls back to `slog.TextHandler` |
| 10 | `TestCustomHandler_NilRecord` | Handles edge case of empty message |

**Capture helper:** Use existing `captureTermOutput()` pattern (swap `os.Stderr` with pipe, log, read back).

### 4.8 Edge Cases

| Case | Handling |
|---|---|
| Unknown placeholder in template (e.g., `{foo}`) | Render as literal text `{foo}` (don't error — be forgiving) |
| `timestamp_format` is empty | Use Go default: `"2006/01/02 15:04:05.000"` |
| `FormatText` is empty | Fall back to `slog.TextHandler` (backward compat) |
| Attrs with complex values (struct, slice) | Use `slog.DefaultValue()` → `fmt.Sprintf("%v", v)` |
| Attrs with `slog.GroupValue` | Flatten groups: `group.key=value` |
| `colorLevelWriter` on non-text writer | Handled: only wraps when `format = "text"` |
| Template with no placeholders | Renders as literal text (valid, if unusual) |

### 4.9 Example Outputs

**Default (no `format_text` set):**
```
2025/01/15 14:30:00.000 level=INFO msg=pipeline started
```

**Custom `format_text = "{time} [{level}] {module}: {message} {attrs}"`:**
```
2025/01/15 14:30:00.000 [INFO] pipeline: pipeline started
```

**Custom `format_text = "{time} [{level}] {module}: {message}"` (no attrs):**
```
2025/01/15 14:30:00.000 [INFO] pipeline: pipeline started
```

**Custom `format_text = "{time} [{level}] {module}: {message} {attrs}"` with `timestamp_format = "15:04:05"`:**
```
14:30:00 [INFO] pipeline: pipeline started
```

**Custom `format_text = "[{level}] {module}: {message} | {attrs}"`:**
```
[INFO] pipeline: pipeline started | unit_id=42 format=srt
```

### 4.10 Implementation Steps

1. **Config** — Add `FormatText` and `TimestampFormat` fields to `config.Logging`; update `defaultCFG()` with defaults
2. **Handler type** — Implement `customHandler` struct + `HandlerOptions` in `logger.go`
3. **Template parsing** — Implement `parseTemplate()` → `[]segment` (string literals + placeholder types)
4. **Handle rendering** — Implement `Handle()` using parsed segments
5. **WithAttrs / WithGroup** — Implement accumulation and group prefixing
6. **Enabled** — Implement level comparison
7. **Wire in newTermHandler** — Check `cfg.FormatText`, create `customHandler` or fall back
8. **Tests** — Write all 10 tests
9. **Gates** — `go build`, `go test -race -cover`, `go vet`, `golangci-lint run`

### 4.11 Backward Compatibility

- **No breaking changes.** Existing configs work unchanged.
- `format_text` is opt-in: if empty, behavior is identical to current (falls back to `slog.TextHandler`).
- `timestamp_format` is opt-in: if empty, uses Go default format.
- `color` flag continues to work with custom handler (wraps the writer).
- File handler (JSON) is unaffected.

### 4.12 Trade-offs Summary

| Aspect | Assessment |
|---|---|
| **Code surface** | ~75 lines new code (non-test), ~100 lines tests |
| **Dependencies** | None (only `fmt`, `strings`, `time`, `sync`, `log/slog`) |
| **Maintenance** | Low — handler is self-contained, no external deps |
| **Upstream drift** | Lose future `slog.TextHandler` improvements (rarely updated) |
| **Performance** | Template parsing per-record (minor); comparable to TextHandler |
| **Breaking changes** | None — opt-in via `format_text` |
| **Complexity** | Medium — slog.Handler interface, but straightforward |

## 5. Future Enhancements (Out of Scope for A5)

- **Format presets**: Add `format_text_preset` field with named presets (`compact`, `detailed`, `json-like`) that expand to template strings
- **Per-level templates**: Allow different templates per log level (e.g., `ERROR` gets more detail)
- **Attr filtering**: Exclude certain attrs from output (e.g., hide internal keys)
- **Full `text/template`**: Upgrade from custom placeholder parsing to Go's `text/template` for power users

## 6. Risks and Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Template parsing bugs (edge cases) | Low | Comprehensive tests for all placeholder combos; unknown placeholders rendered literally |
| Attr rendering inconsistency with slog.TextHandler | Low | Follow same `key=value` pattern; test against TextHandler output |
| Performance regression | Very low | Benchmarks vs TextHandler; string builder reuse for attrs |
| Color integration breaks | Low | Test `colorLevelWriter` wrapping custom handler; same post-processing approach |
| Group nesting bugs | Low | Mirror `slog.Handler` contract exactly; test with chained `WithGroup` calls |

## 7. Configuration Example

```toml
[logger]
level = "info"
terminal = true
format = "text"
format_text = "{time} [{level}] {module}: {message} {attrs}"
timestamp_format = "15:04:05"
color = true
```

This produces output like:
```
15:04:05 [INFO] pipeline: pipeline started
15:04:05 [DEBUG] pipeline: scanning input_dir | unit_id=42 format=srt
```
