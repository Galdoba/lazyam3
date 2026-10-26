# Text Logger Format Refactoring — Implementation Spec

## 1. Problem Statement

The current terminal log output uses Go's `slog.TextHandler` format, which is rigid:

**Old format:**
```
time=2026-10-25T22:09:52.794+10:00 level=INFO msg=processing pipeline.dir=/home/galdoba/workspace/lazyam3/incoming/multi-audio
```

**New format (now the default):**
```
26-10-26 12:10:18.812 [INFO] pipeline: processing dir="/home/galdoba/workspace/lazyam3/incoming/multi-audio"
```

Key differences:
- Timestamp: full ISO-style → compact `YY-MM-DD HH:MM:SS.mmm`
- Level: `level=INFO` → `[INFO]` (colorized in place when `color = true`)
- Module: `pipeline.dir=...` → `pipeline: dir=...` (group stack rendered as module, attr keys unprefixed)
- Attrs: string values quoted, non-string values unquoted
- Field order: time, level, module, message, attrs — driven by a template

## 2. Decisions (confirmed before implementation)

| # | Question | Decision |
|---|---|---|
| 1 | Activation | **Ship as new default.** `defaultCFG()` sets `FormatText` and `TimestampFormat`; configs that omit the keys inherit them (configmanager decodes TOML over the defaults). Opt out with `format_text = ""`. |
| 2 | `{level}` rendering | **Bare level name** (`INFO`); brackets are literal template text (`{time} [{level}] ...`). |
| 3 | Empty placeholders | **Separator swallowing:** an empty `{module}`/`{attrs}`/`{message}` also drops the adjacent literal if it consists only of spaces and colons — no dangling `: ` or trailing space. |

Design deviations from the original draft (simpler solutions preferred):

- **Colorization happens inside the handler, not in `colorLevelWriter`.** The draft's §6 regex had no capture groups at all (`loc[4]` would panic) and could not anchor the bracketed level inside a space-containing timestamp. The custom handler knows exactly where the level is, and the level is a fixed known string (no injection risk), so it wraps it in ANSI codes directly. `colorLevelWriter` and its regex stay untouched for the `slog.TextHandler` fallback path only.
- **Template is parsed once at handler creation**, not per record.
- **No shared mutable state:** `Handle` uses a local buffer; `attrs`/`groups` are immutable after construction (`WithAttrs`/`WithGroup` copy). Concurrency safety needs no mutex.
- `textHandlerOptions` is unexported — the package is `internal`, no need for an exported API.

## 3. Architecture

```
slog.Logger
    ↓ multiHandler (internal/infra/logging/logger.go)
    ├── slog.JSONHandler       → file (untouched)
    └── text format:
        ├── FormatText != "" → customTextHandler → os.Stderr  (colorizes level itself)
        └── FormatText == "" → slog.TextHandler  → os.Stderr
                                └── colorLevelWriter (wraps os.Stderr, untouched)
```

**Logger facade:** `internal/infra/logging/logger.go` — `Logger` type wrapping `*slog.Logger` with `With()`, `Group()`, and typed methods (`Info`, `Debug`, `Warn`, `Error`, `Errorf`). API unchanged.

**Usage patterns:**
- `modules.Logger.Group("pipeline")` — pipeline-scoped logging (internal/infra/commands/cmd_process.go)
- `logger.Info("processing", "dir", dir)` — key=value attributes

**Config:** `internal/infra/config/config.go` — `Logging` struct with `Level`, `Path`, `Terminal`, `Format`, `Color`, `FormatText`, `TimestampFormat`.

### 3.1 Files Changed

| File | Change | Description |
|---|---|---|
| `internal/infra/config/config.go` | Modify | `FormatText`, `TimestampFormat` on `Logging`; defaults in `defaultCFG()` |
| `internal/infra/logging/logger.go` | Modify | `customTextHandler` + template parsing; wire in `newTermHandler()` |
| `internal/infra/logging/logger_test.go` | Modify | 14 handler/end-to-end tests |
| `internal/infra/config/config_test.go` | Modify | 2 default/inheritance tests |

### 3.2 Files NOT Affected

| File | Reason |
|---|---|
| `newFileHandler()` | File handler uses `slog.JSONHandler` — untouched (verified by live test: JSON lines unchanged) |
| `multiHandler` | Dispatches to all handlers — agnostic to handler type |
| `Logger` facade | Wraps `*slog.Logger` — agnostic to handler implementation |
| `cmd_process.go`, `pipeline.go` | Use `Group()` / `Info()` — API unchanged |
| `colorLevelWriter` + `levelFieldRe` | Reserved for the `slog.TextHandler` fallback; not used on the custom path |
| Any domain package | No logging dependencies — pure logic |

## 4. Config

```go
type Logging struct {
	Level           string `toml:"level"`
	Path            string `toml:"path"`
	Terminal        bool   `toml:"terminal"`
	Format          string `toml:"format"`           // "json" or "text" — terminal only
	Color           bool   `toml:"color"`            // colorize level in terminal text output
	FormatText      string `toml:"format_text"`      // text template; empty falls back to slog.TextHandler
	TimestampFormat string `toml:"timestamp_format"` // Go time layout for {time}; empty uses the default layout
}
```

**Defaults in `defaultCFG()`:**
- `FormatText = "{time} [{level}] {module}: {message} {attrs}"` — produces the desired format out of the box
- `TimestampFormat = "06-01-02 15:04:05.000"` — compact `YY-MM-DD HH:MM:SS.mmm`

**Gating:** `format_text` is honored only when `format = "text"`. With `format = "json"` the terminal stays JSON regardless of `format_text`.

## 5. `customTextHandler` Design

```go
type textHandlerOptions struct {
	level           slog.Level
	timestampFormat string
	template        string
	color           bool
}

type textTemplateSegment struct {
	text        string   // literal text (empty for placeholders)
	placeholder string   // "time" | "level" | "module" | "message" | "attrs"
}

type customTextHandler struct {
	out    io.Writer
	opts   textHandlerOptions
	tmpl   []textTemplateSegment // parsed once at creation
	attrs  []slog.Attr           // accumulated via WithAttrs (immutable copy)
	groups []string              // accumulated via WithGroup (immutable copy)
}
```

**Methods:**
- `Enabled(ctx, level)` — `level >= opts.level`
- `Handle(ctx, record)` — renders template segments into a local buffer, writes one line (+"\n")
- `WithAttrs(attrs)` — returns a copy with accumulated attrs appended
- `WithGroup(name)` — returns a copy with the group name appended to the stack

**Placeholders:**

| Placeholder | Rendering |
|---|---|
| `{time}` | `record.Time.Format(layout)`; empty `timestampFormat` → `2006/01/02 15:04:05.000` |
| `{level}` | Bare level name (`INFO`); wrapped in ANSI color codes when `color = true` |
| `{module}` | Group stack joined with `.` (`pipeline`, `pipeline.command`); empty when ungrouped |
| `{message}` | `record.Message` as plain text (not a key=value pair) |
| `{attrs}` | Space-separated `key=value` pairs — accumulated + record attrs, group prefix NOT repeated (the module shows it). Strings quoted via `strconv.Quote`, everything else `fmt.Sprintf("%v")`. Nested `slog.Group` attrs flatten to `outer.inner.key=value`. Values are `Resolve()`d (LogValuer support). |

**Template parsing (once, at creation):**
- Split on `{...}` into alternating literal/placeholder segments
- Unknown placeholders (`{foo}`) stay literal text — forgiving, no error

**Separator swallowing:**
When a placeholder renders empty, the adjacent literal is dropped if it consists only of spaces and colons:
- empty `{module}` in `"... {module}: {message}"` → drops `": "`
- empty `{attrs}` in `"... {message} {attrs}"` → drops the trailing `" "`
- empty `{message}` → drops `": "` and the following `" "`

**Edge cases:**

| Case | Handling |
|---|---|
| Unknown placeholder (`{foo}`) | Rendered literally as `{foo}` |
| Empty `format_text` | Fall back to `slog.TextHandler` (old format, `colorLevelWriter` path) |
| Empty `timestamp_format` | Default layout `2006/01/02 15:04:05.000` |
| Empty `record.Message` | Rendered empty; adjacent separators swallowed |
| Attr value with spaces (e.g. error) | Non-strings are unquoted per spec: `error=product 63580 not found` |
| `format = "json"` + `format_text` set | `format_text` ignored — terminal stays JSON |
| Concurrent `Handle` calls | Safe — no shared mutable state (verified by `-race` test) |

## 6. `colorLevelWriter` — Unchanged

The draft's proposed regex replacement had no capture groups (the existing
`colorizeLevel` slices `loc[4]:loc[5]` — an index-out-of-range panic waiting to
happen) and could not match a bracketed level behind a space-containing
timestamp. Since the custom handler renders the level itself, no regex surgery
is needed at all:

- Custom path: handler colorizes `{level}` directly (level is a fixed string —
  no escaping/injection concern; attr values are never colorized).
- Fallback path (`format_text = ""`): existing `levelFieldRe` + `colorLevelWriter`
  keep working on `slog.TextHandler` output exactly as before.

## 7. Tests

| # | Test | Verifies |
|---|---|---|
| 1 | `TestCustomHandler_RenderBasic` | Exact line: `26-10-25 22:09:52.794 [INFO] pipeline: processing dir="/path"` |
| 2 | `TestCustomHandler_RenderAttrs` | Quoted strings, unquoted int/bool/float |
| 3 | `TestCustomHandler_AccumulatedAttrs` | `WithAttrs` attrs appear in `{attrs}` |
| 4 | `TestCustomHandler_Module` | Empty module, one group, nested `a.b` |
| 5 | `TestCustomHandler_GroupPrefixStripping` | Group shown via `{module}` only — never `pipeline.dir` |
| 6 | `TestCustomHandler_Color` | ANSI around level when `color=true`, plain when `false` |
| 7 | `TestCustomHandler_TimestampFormat` | Custom layout + default layout fallback |
| 8 | `TestCustomHandler_Message` | Message rendered as plain text |
| 9 | `TestCustomHandler_EmptyMessageAndAttrsSwallowSeparators` | No dangling `: `, no trailing space |
| 10 | `TestCustomHandler_TemplateReordering` | `{message} {level} {time}` field order |
| 11 | `TestCustomHandler_UnknownPlaceholder` | `{foo}` passes through literally |
| 12 | `TestCustomHandler_NestedGroupAttr` | `slog.Group` attr flattens to `inner.k="v"` |
| 13 | `TestCustomHandler_ConcurrentHandle` | 50 goroutines, `-race` clean |
| 14 | `TestNewTermHandler_CustomFormatText` | End-to-end `newTermHandler` output via `captureTermOutput()` |
| 15 | `TestNewTermHandler_FallbackToTextHandler` | Empty `format_text` → `level=INFO` (old format) |
| 16 | `TestNewTermHandler_FormatTextIgnoredForJSON` | `format = "json"` ignores `format_text` |
| 17 | `TestDefaultConfig_LoggerFormatText` (config) | Default template + layout values |
| 18 | `TestLoad_FormatTextInheritedWhenMissing` (config) | TOML without the keys inherits defaults |

## 8. Backward Compatibility

- **File handler (JSON) unchanged.**
- **Terminal format changes by default** (decision #1): configs that do not set
  `format_text` inherit the default template and switch to the new format.
  This is the intended behavior change of this feature.
- Opt out: set `format_text = ""` in `[logger]` to get the exact old
  `slog.TextHandler` output (with `colorLevelWriter` colorization).
- The `Logger` API (`Info`/`Group`/`With`/`Errorf`) is unchanged — no call-site edits.
- `IsDefault()` stays consistent: both new fields are part of `defaultCFG()`.

## 9. Trade-offs Summary

| Aspect | Assessment |
|---|---|
| Code surface | ~180 lines new code (non-test), ~350 lines tests |
| Dependencies | None (only `fmt`, `io`, `strconv`, `strings`, `time`, `log/slog`) |
| Upstream drift | Lose future `slog.TextHandler` improvements on the custom path (fallback still available) |
| Performance | Template parsed once; per-record cost comparable to TextHandler |
| Breaking changes | Terminal output format changes by default (intended, opt-out available) |
| Complexity | Medium — handler is self-contained, no shared mutable state |

## 10. Configuration Example

```toml
[logger]
level = "info"
terminal = true
format = "text"
format_text = "{time} [{level}] {module}: {message} {attrs}"   # default
timestamp_format = "06-01-02 15:04:05.000"                    # default
color = true
```

Produces (live test output):
```
26-10-26 12:10:18.812 [INFO] command: create pipeline
26-10-26 12:10:18.907 [INFO] pipeline: dirs found count=6
26-10-26 12:10:19.000 [WARN] pipeline: metadata lookup failed, using fallback naming id=63580 dir="/.../embedded-sub" error=product 63580 not found
```

## 11. Example Outputs

**Default template (grouped record, attrs):**
```
26-10-25 22:09:52.794 [INFO] pipeline: processing dir="/home/galdoba/workspace/lazyam3/incoming/multi-audio"
```

**Default template (ungrouped, no attrs):** separators swallowed
```
26-10-25 22:09:52.794 [INFO] logger created
```

**`format_text = "[{level}] {module}: {message} | {attrs}"`, `timestamp_format = "15:04:05"`:**
```
15:04:05 [INFO] pipeline: unit done | id=42
```

**`format_text = ""` (opt-out):** old format
```
time=2026-10-25T22:09:52.794+10:00 level=INFO msg=processing pipeline.dir=/path
```
