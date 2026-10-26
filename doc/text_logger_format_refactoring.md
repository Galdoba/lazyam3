# Text Logger Format Refactoring — Analysis

## 1. Problem Statement

The current terminal log output uses Go's `slog.TextHandler` format, which is rigid:

**Current format:**
```
time=2026-10-25T22:09:52.794+10:00 level=INFO msg=processing pipeline.dir=/home/galdoba/workspace/lazyam3/incoming/multi-audio
```

**Desired format:**
```
26-10-25 22:09:52.794 [INFO] pipeline: processing dir="/home/galdoba/workspace/lazyam3/incoming/multi-audio"
```

Key differences:
- Timestamp: full ISO-style → compact `YY-MM-DD HH:MM:SS.mmm`
- Level: `level=INFO` → `[INFO]`
- Module: `pipeline.dir=...` → `pipeline: dir=...` (group name extracted, key unprefixed)
- Attrs: string values quoted, non-string values unquoted
- Field order: time, level, module, message, attrs

## 2. Current Architecture

```
slog.Logger
    ↓ multiHandler (internal/infra/logging/logger.go:205)
    ├── slog.JSONHandler   → file (internal/infra/logging/logger.go:188)
    └── slog.TextHandler   → os.Stderr (internal/infra/logging/logger.go:136)
                            └── colorLevelWriter (wraps os.Stderr, internal/infra/logging/logger.go:150)
```

**Logger facade:** `internal/infra/logging/logger.go:19` — `Logger` type wrapping `*slog.Logger` with `With()`, `Group()`, and typed methods (`Info`, `Debug`, `Warn`, `Error`, `Errorf`).

**Usage patterns:**
- `modules.Logger.Group("pipeline")` — pipeline-scoped logging (internal/infra/commands/cmd_process.go:108)
- `logger.Info("processing", "dir", dir)` — key=value attributes

**Config:** `internal/infra/config/config.go:122` — `Logging` struct with `Level`, `Path`, `Terminal`, `Format`, `Color`.

## 3. Impact Analysis

### 3.1 Files Requiring Changes

| File | Change Type | Description |
|---|---|---|
| `internal/infra/config/config.go` | Modify | Add `FormatText`, `TimestampFormat` fields to `Logging`; update `defaultCFG()` |
| `internal/infra/logging/logger.go` | Modify | Implement `customTextHandler`; wire in `newTermHandler()`; update `colorLevelWriter` regex |
| `internal/infra/logging/logger_test.go` | Modify | Add tests for custom handler, color integration, config wiring |

### 3.2 Files NOT Affected

| File | Reason |
|---|---|
| `internal/infra/logging/logger.go:newFileHandler()` | File handler uses `slog.JSONHandler` — untouched |
| `internal/infra/logging/logger.go:multiHandler` | Dispatches to all handlers — agnostic to handler type |
| `internal/infra/logging/logger.go:Logger` facade | Wraps `*slog.Logger` — agnostic to handler implementation |
| `internal/infra/commands/cmd_process.go` | Uses `logger.Group()` / `logger.Info()` — API unchanged |
| `internal/app/pipeline/pipeline.go` | Uses `p.logger.Info()` / `p.logger.Warn()` — API unchanged |
| Any domain package | No logging dependencies — pure logic |

### 3.3 `colorLevelWriter` Compatibility

**Critical:** The existing `colorLevelWriter` regex won't match the new format.

Current regex (`logger.go:141`):
```go
var levelFieldRe = regexp.MustCompile(`^(time=\S+ )?level=([A-Z]+)`)
```

This matches `level=INFO` but NOT `[INFO]`. **Recommended fix:** update the regex to support both formats:
```go
var levelFieldRe = regexp.MustCompile(`^(?:\S+\s+)?(?:\[?(?:DEBUG|INFO|WARN|ERROR)\]?|level=(?:DEBUG|INFO|WARN|ERROR))`)
```

This matches:
- Old format: `time=2026-10-25T22:09:52.794+10:00 level=INFO`
- New format: `26-10-25 22:09:52.794 [INFO]`

The color injection logic needs a small adjustment for the new capture group indices.

### 3.4 Attribute Quoting Logic

```go
func formatAttrValue(v any) string {
    switch val := v.(type) {
    case string:
        return strconv.Quote(val)
    default:
        return fmt.Sprintf("%v", v)
    }
}
```

### 3.5 Group/Module Handling

**Key clarification:** The group stack is maintained via `WithGroup()`. When rendering attributes:

1. The group path becomes the **module** (e.g., `["pipeline"]` → `"pipeline"`, `["pipeline","command"]` → `"pipeline.command"`)
2. Attribute keys that start with a group prefix are **stripped** (e.g., `pipeline.dir` → `dir`)
3. Attributes are rendered as `key=value` pairs, space-separated

Current `slog.TextHandler` output for `logger.Group("pipeline").Info("processing", "dir", "/path")`:
```
pipeline.dir=/path  (group.key=value — group prefix retained)
```

Desired output:
```
pipeline: dir="/path"  (group as module prefix, key unprefixed)
```

### 3.6 Timestamp Format

Configurable via new `TimestampFormat` field. **Default behavior:** if `TimestampFormat` is empty, use Go's standard `Time.Format("2006/01/02 15:04:05.000")` to match existing `slog.TextHandler` behavior.

## 4. Proposed Config Changes

```go
type Logging struct {
    Level         string `toml:"level"`
    Path          string `toml:"path"`
    Terminal      bool   `toml:"terminal"`
    Format        string `toml:"format"`         // "json" (default) or "text"
    Color         bool   `toml:"color"`         // colorize level in text output
    FormatText    string `toml:"format_text"` // custom template (NEW, opt-in)
    TimestampFormat string `toml:"timestamp_format"` // Go time format (NEW)
}
```

**Defaults in `defaultCFG()`:**
- `FormatText = ""` (empty → fall back to `slog.TextHandler`, backward compatible)
- `TimestampFormat = ""` (empty → Go standard format)

## 5. `customTextHandler` Design

```go
type customTextHandler struct {
    opts   HandlerOptions
    attrs  []slog.Attr
    groups []string
    out    io.Writer
    buf    bytes.Buffer
}

type HandlerOptions struct {
    Level          slog.Level
    TimestampFormat string
    Template       string // e.g., "{time} [{level}] {module}: {message} {attrs}"
}
```

**Methods:**
- `Enabled(ctx, level) bool` — returns true if `level >= opts.Level`
- `Handle(ctx, record) error` — renders template segments against record + accumulated attrs
- `WithAttrs(attrs) Handler` — returns handler with accumulated attrs prepended
- `WithGroup(name) Handler` — returns handler with group name prepended to stack

**Rendering pipeline (Handle):**
1. Parse template into segments (literal text + placeholders)
2. For each segment:
   - `{time}` → `record.Time.Format(opts.TimestampFormat)`
   - `{level}` → `[INFO]`
   - `{module}` → joined group stack (e.g., `"pipeline"`)
   - `{message}` → `record.Msg` (rendered as text, not as `msg=` key)
   - `{attrs}` → all attrs as `key=value` pairs (group prefixes stripped), space-separated
3. Write final buffer to `out`

**Message rendering clarification:** The `record.Message` is rendered as plain text after the module, not as a key-value pair. Empty messages are rendered as empty text (not omitted), matching the template.

### Template Parsing

Template is parsed at handler creation time:
- Split on `{...}` into alternating literal/placeholder segments
- Unknown placeholders (e.g., `{foo}`) are rendered as literal `{foo}` — no error
- Example: `"{time} [{level}] {module}: {message} {attrs}"` →
  - Literal: `"{time} [{level}] {module}: "`
  - Placeholder: `{time}`, `{level}`, `{module}`, `{message}`, `{attrs}`

### Edge Cases
| Case | Handling |
|---|---|
| Unknown placeholder (`{foo}`) | Rendered as literal `{foo}` (forgiving) |
| Empty `FormatText` | Fall back to `slog.TextHandler` |
| Empty `TimestampFormat` | Use Go default `2006/01/02 15:04:05.000` |
| `record.Msg` empty | Render empty text (no `msg=` key) |
| Attrs with complex values | Use `fmt.Sprintf("%v", v)` |
| Group prefix matching | Strip full group prefix from attr keys |

## 6. `colorLevelWriter` Update

Current regex (line 141):
```go
var levelFieldRe = regexp.MustCompile(`^(time=\S+ )?level=([A-Z]+)`)
```

Updated regex:
```go
var levelFieldRe = regexp.MustCompile(`^(?:\S+\s+)?(?:\[?(?:DEBUG|INFO|WARN|ERROR)\]?|level=(?:DEBUG|INFO|WARN|ERROR))`)
```

The color injection logic needs a small adjustment to find the correct capture group indices for the new regex.

## 7. Test Plan

| # | Test | What it verifies |
|---|---|---|
| 1 | `TestCustomHandler_RenderBasic` | Template with `{time} [{level}] {module}: {message} {attrs}` |
| 2 | `TestCustomHandler_RenderAttrs` | `{attrs}` renders `key=value` pairs |
| 3 | `TestCustomHandler_AccumulatedAttrs` | Attrs from `WithAttrs` appear in output |
| 4 | `TestCustomHandler_GroupPrefix` | `WithGroup("g")` prefixes attrs as `g.key=value` |
| 5 | `TestCustomHandler_GroupPrefixStripping` | Group prefix stripped from attr keys |
| 6 | `TestCustomHandler_ColorIntegration` | `colorLevelWriter` still colorizes |
| 7 | `TestCustomHandler_TimestampFormat` | Custom format changes time output |
| 8 | `TestCustomHandler_Message` | `record.Msg` rendered as text, empty msg rendered as empty |
| 9 | `TestCustomHandler_TemplateReordering` | `{message} {level} {time}` reverses field order |
| 10 | `TestNewTermHandler_CustomFormatText` | End-to-end: `newTermHandler` with `FormatText` |
| 11 | `TestNewTermHandler_Fallback` | Without `FormatText`, falls back to `slog.TextHandler` |

**Capture helper:** Use existing `captureTermOutput()` pattern (swap `os.Stderr` with pipe, log, read back).

## 8. Backward Compatibility

- **No breaking changes.** Existing configs work unchanged.
- `format_text` is opt-in: if empty, behavior is identical to current (falls back to `slog.TextHandler`).
- `timestamp_format` is opt-in: if empty, uses Go default.
- `color` flag continues to work with custom handler (wraps the writer).
- File handler (JSON) is unaffected.

## 9. Trade-offs Summary

| Aspect | Assessment |
|---|---|
| Code surface | ~75 lines new code (non-test), ~100 lines tests |
| Dependencies | None (only `fmt`, `strings`, `time`, `sync`, `log/slog`) |
| Upstream drift | Lose future `slog.TextHandler` improvements (rarely updated) |
| Performance | Template parsing per-record (minor); comparable to TextHandler |
| Breaking changes | None — opt-in via `FormatText` |
| Complexity | Medium — handler is self-contained |

## 10. Implementation Steps

1. **Config** — Add `FormatText` and `TimestampFormat` fields to `config.Logging`; update `defaultCFG()` with defaults
2. **Handler type** — Implement `customTextHandler` struct + `HandlerOptions` in `logger.go`
3. **Template parsing** — Implement `parseTemplate()` → `[]segment` (literal/placeholder types)
4. **Handle rendering** — Implement `Handle()` using parsed segments
5. **WithAttrs / WithGroup** — Implement accumulation and group prefixing
6. **Enabled** — Implement level comparison
7. **Wire in newTermHandler** — Check `cfg.FormatText`, create `customHandler` or fall back
8. **Tests** — Write all 11 tests
9. **Gates** — `go build`, `go test -race -cover`, `go vet`, `golangci-lint run`

## 11. Configuration Example

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

## 12. Example Outputs

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
15:04:05 [INFO] pipeline: pipeline started
```

**Custom `format_text = "[{level}] {module}: {message} | {attrs}"`:**
```
[INFO] pipeline: pipeline started | unit_id=42 format=srt
```
