# Logging Improvements for `lazyam3`

## Executive Summary

Current logging uses Go's `log/slog` with a custom typed facade. It supports dual output (file + terminal), contextual loggers, and config-driven levels. However, several gaps exist that reduce usability for debugging, monitoring, and long-running operations.

This document catalogs improvements grouped by effort and impact, with implementation details and trade-offs.

---

## Current State

### Architecture

```
Config (TOML)
  └─ Logger
     ├─ Level (string)
     ├─ Path (string)
     ├─ Terminal (bool)
     ├─ Format (string: "json"|"text", terminal only)
     └─ Color (bool, terminal text only)

Logger (*logging.Logger)
  └─ multiHandler (wraps slog.Handler)
     ├─ newFileHandler (JSON, append-only)
     └─ newTermHandler (JSON or Text per cfg.Format, stderr)
```

### What Works Well

- **Typed surface**: `Debug/Info/Warn/Error` methods return `*Logger`, not `*slog.Logger`
- **Child loggers**: `With(args...)` preserves typed surface
- **Dual output**: File + terminal simultaneously
- **Resource safety**: `Close()` drains file handles
- **Escape hatch**: `Slog()` for third-party integrations

---

## Category A: Quick Wins (10–15 min each)

### A1. `Attrs()` Method for Contextual Keys

**Problem**: Users must call `With()` every time they add attributes. No shorthand for single key/value.

**Current**:
```go
l.With("unit_id", 42).Info("processed")
```

**Proposed**:
```go
l.Attrs("unit_id", 42).Info("processed")  // shorter
l.Attrs("unit_id", 42, "duration", 1.5).Info("done")  // fluent
```

**Implementation**:
```go
func (l *Logger) Attrs(kvs ...any) *Logger {
    attrs := make([]slog.Attr, 0, len(kvs)/2+1)
    for i := 0; i < len(kvs); i += 2 {
        if i+1 < len(kvs) {
            attrs = append(attrs, slog.Any(kvs[i], kvs[i+1]))
        } else {
            attrs = append(attrs, slog.Any(kvs[i], nil))
        }
    }
    return &Logger{inner: l.inner.WithAttrs(attrs)}
}
```

**Benefits**:
- Syntactic sugar for quick key-value pairs
- Chains nicely with `With()` for compound context
- Zero runtime cost (just builds `[]slog.Attr`)

**Trade-offs**:
- Adds another method to remember
- Slight code duplication between `With()` and `Attrs()`

---

### A2. `Group()` Helper for Named Contexts

**Problem**: Commands must call `Slog().WithGroup("name")` then wrap back to `*Logger`. Breaks type consistency.

**Current**:
```go
logger := modules.Logger.Slog().WithGroup("command")  // loses typed surface
logger.WithGroup("pipeline").Info("step")             // verbose
```

**Proposed**:
```go
logger := modules.Logger.Group("command")             // returns *Logger
logger.With("id", 42).Info("processed")               // typed surface preserved
```

**Implementation**:
```go
func (l *Logger) Group(name string) *Logger {
    return &Logger{inner: l.inner.WithGroup(name)}
}
```

**Benefits**:
- Preserves typed surface (`*Logger` not `*slog.Logger`)
- Consistent with `With()` API
- Reduces boilerplate in commands

**Trade-offs**:
- Minimal; just another method

---

### A3. JSON vs Text Format Toggle

**Problem**: Terminal output is always JSON (machine-readable) but unreadable for humans. File is always JSON.

**Current**:
```toml
[logger]
level = "info"
path = "/tmp/lazyam3.log"
terminal = true
```

**Proposed**:
```toml
[logger]
level = "info"
path = "/tmp/lazyam3.log"
terminal = true
format = "json"  # or "text"
```

**Implementation**:
```go
type Logging struct {
    Level    string `toml:"level"`
    Path     string `toml:"path"`
    Terminal bool   `toml:"terminal"`
    Format   string `toml:"format"`  // "json"|"text"
}
```

```go
func newTermHandler(cfg config.Logging) (slog.Handler, error) {
    if !cfg.Terminal {
        return nil, nil
    }
    opts := &slog.HandlerOptions{Level: levelFromString(cfg.Level)}
    
    switch strings.ToLower(cfg.Format) {
    case "text":
        return slog.NewTextHandler(os.Stderr, opts), nil
    default:
        return slog.NewJSONHandler(os.Stderr, opts), nil
    }
}
```

**Benefits**:
- Humans get readable logs by default (`text`)
- Machine tools get JSON (`json` or omit)
- No breaking changes (default: `json`)

**Trade-offs**:
- One extra config field
- Terminal text is less structured (no field names)

---

### A4. Color Support for Terminal Logs

**Problem**: Human-readable text logs are colorless; levels don't stand out.

**Proposed**:
```toml
[logger]
terminal = true
color = true  # or false
```

**Implementation**:
```go
func newTermHandler(cfg config.Logging) (slog.Handler, error) {
    if !cfg.Terminal {
        return nil, nil
    }
    opts := &slog.HandlerOptions{
        Level:  levelFromString(cfg.Level),
        ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
            if a.Key == "level" {
                color := getColorForLevel(a.Value.String())
                return slog.String("level", color)
            }
            return a
        },
    }
    
    if cfg.Color {
        // Wrap text handler with colorizer
        return newColorizedTextHandler(os.Stderr, opts), nil
    }
    return slog.NewTextHandler(os.Stderr, opts), nil
}
```

**Benefits**:
- Levels/colors make logs scannable
- No extra deps (use `colorable` or `github.com/mattn/go-colorable`)

**Trade-offs**:
- Adds dependency
- Colors don't copy to file
- Needs terminal support detection

---

## Category B: Medium Effort (1–4 hours)

### B1. TraceID / SpanID Context Injection

**Problem**: No correlation IDs for debugging multi-step pipelines or distributed scenarios.

**Proposed**:
```toml
[logger]
level = "info"
path = "/tmp/lazyam3.log"
terminal = true
trace_id = "abc-123"  # optional
span_id = "def-456"   # optional
```

**Implementation**:
```go
type Logging struct {
    Level     string `toml:"level"`
    Path      string `toml:"path"`
    Terminal  bool   `toml:"terminal"`
    TraceID   string `toml:"trace_id"`
    SpanID    string `toml:"span_id"`
}
```

```go
func NewLogger(cfg config.Logging) (*Logger, error) {
    handlers, closers, err := createHandlers(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to collect handlers: %w", err)
    }
    
    // Inject context attrs into all handlers
    attrs := []any{}
    if cfg.TraceID != "" {
        attrs = append(attrs, slog.String("trace_id", cfg.TraceID))
    }
    if cfg.SpanID != "" {
        attrs = append(attrs, slog.String("span_id", cfg.SpanID))
    }
    
    logger := slog.New(&multiHandler{handlers: handlers})
    if len(attrs) > 0 {
        logger = logger.With(attrs...)
    }
    
    return &Logger{inner: logger, closers: closers}, nil
}
```

**Benefits**:
- Correlate all logs from a single request/pipeline run
- Easy to add to existing config
- Works with OpenTelemetry, Jaeger, etc.

**Trade-offs**:
- Must remember to set config fields
- Adds two more config fields

---

### B2. Log Rotation (Size or Time Based)

**Problem**: Log files grow indefinitely; no cleanup strategy.

**Current**: `os.OpenFile(path, O_APPEND)` — append only.

**Proposed Options**:

#### Option 1: External `logrotate` (Recommended)
```toml
[logger]
path = "/tmp/lazyam3.log"
rotate = true  # let logrotate handle it
```

**Why**: Proven, battle-tested, no Go deps.

#### Option 2: Go-based Rotation (Complex)
```go
func newFileHandler(cfg config.Logging) (slog.Handler, io.Closer, error) {
    if !cfg.Rotate {
        return nil, nil, nil
    }
    
    // Use zapcore or write your own rotation
    // Or: https://github.com/jzelinskie/whitedevil
    return nil, nil, fmt.Errorf("log rotation requires zap or external library")
}
```

**Benefits**:
- File sizes stay bounded
- Old logs can be archived/compressed
- `logrotate` handles permissions, compression, retention

**Trade-offs**:
- External rotation adds complexity
- Go rotation is heavier (deps, more code)

**Decidion**:
- defer, but keep it as a future expansion option. Add it to backlog.

---

### B3. Buffered / Async Writer (for High-Frequency Logging)

**Problem**: Logging blocks during render loops, progress updates, etc.

**Current**: Synchronous writes to file/terminal.

**Proposed**:
```toml
[logger]
buffer_size = 1024  # entries
flush_interval_ms = 500
```

**Implementation**:
```go
type Logger struct {
    inner   *slog.Logger
    closers []io.Closer
    buffer  chan slog.Record
}

func NewLogger(cfg config.Logging) (*Logger, error) {
    buffer := make(chan slog.Record, cfg.BufferSize)
    go func() {
        ticker := time.NewTicker(time.Duration(cfg.FlushInterval) * time.Millisecond)
        for {
            select {
            case rec := <-buffer:
                // Write to handlers (same multiHandler logic)
                // Flush ticker
            case <-ticker.C:
                // Force flush
            case <-ctx.Done():
                return
            }
        }
    }()
    
    return &Logger{...}, nil
}
```

**Benefits**:
- Logging doesn't block long-running operations
- Backpressure if disk is slow
- Batching reduces I/O

**Trade-offs**:
- Adds latency (buffer + ticker)
- More complex code
- Debug harder (logs out of order)
- **Risk**: Might not be needed for this use case

**Decidion**:
- defer, but keep it as a future expansion option. Add it to backlog.
---

### B4. Health Check Method

**Problem**: Can't tell if logger itself is broken (e.g., disk full, permission denied).

**Proposed**:
```go
func (l *Logger) Check() error {
    var buf bytes.Buffer
    handler := slog.NewTextHandler(&buf, nil)
    rec := slog.NewRecord(time.Now(), slog.LevelInfo, "health check", 0)
    err := handler.Handle(context.Background(), rec)
    return err
}
```

**Benefits**:
- Detects logger failures early
- Use in startup checks or periodic probes
- Can be added to health check endpoints

**Trade-offs**:
- Extra method to remember
- Minimal value unless health checks are critical

---

### B5. `Context()` Helper for Span Propagation

**Problem**: No way to propagate context from parent to child loggers.

**Proposed**:
```go
// Parent
logger := modules.Logger.Group("parent")
logger.Info("started")

// Child
childLogger := logger.Context("child_group").With("id", 42)
childLogger.Info("done")
```

**Implementation**:
```go
func (l *Logger) Context(name string) *Logger {
    return l.With(slog.String("context", name))
}
```

**Benefits**:
- Groups related logs without extra config
- Useful for request tracing

**Trade-offs**:
- Similar to `Attrs()`; might be redundant
- Naming could be clearer

**Decidion**:
- defer, I see no reason for this feature to exist for now. More analysis required.

---

## Priority Matrix

| Improvement | Effort | Impact | Priority |
|-------------|--------|---------|----------|
| A1. `Attrs()` | 10 min | High | ⭐⭐⭐ |
| A2. `Group()` | 5 min | High | ⭐⭐⭐ |
| A3. Format toggle | 15 min | Medium | ⭐⭐ |
| A4. Colors | 20 min | Medium | ⭐⭐ |
| B1. TraceID | 1 hour | High | ⭐⭐⭐ |
| B2. Rotation | 2 hours | High | ⭐⭐ |
| B3. Async buffer | 4 hours | Low/Med | ⭐ |
| B4. Health check | 30 min | Low | ⭐ |
| B5. Context helper | 10 min | Low | - |

---

## Recommended Implementation Order

### Phase 1: Quick Wins (30–60 min)
1. **A1. `Attrs()`** — improves ergonomics
2. **A2. `Group()`** — improves ergonomics
3. **A3. Format toggle** — improves readability

### Phase 2: Debuggability (2–4 hours)
4. **B1. TraceID/SpanID** — for distributed tracing
5. **A4. Colors** — improves terminal UX
6. **B2. Rotation** — prevents disk fill

### Phase 3: Optional (as needed)
- B3. Async buffer — only if high-frequency logging is needed
- B4. Health check — if startup health checks are required
- B5. Only if scaling to production

---

## Example: After Improvements

**Config**:
```toml
[logger]
level = "info"
path = "/tmp/lazyam3.log"
terminal = true
format = "text"
color = true
trace_id = "req-123"
rotate = true
```

**Usage**:
```go
// Contextual logging
logger.Attrs("unit_id", 42).Info("started")
logger.Attrs("unit_id", 42, "duration", 1.5).Info("done")

// Grouped contexts
logger.Group("command").Attrs("step", "scan").Info("scanning")
logger.Group("pipeline").Info("processing")

// With trace correlation
// All logs from this run include trace_id = "req-123"
```

---

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking changes | Default values preserve current behavior |
| Performance | Async buffer is opt-in; not default |
| Complexity | Start with small changes; measure impact |
| Testing | Add tests for new methods (already in `logger_test.go`) |

---

## Conclusion

The current logging is solid for a CLI tool. Quick wins (A1–A3) improve ergonomics and readability with minimal risk. TraceID (B1) and colors (A4) are high-impact for debugging. Rotation (B2) prevents disk issues.

Start with Phase 1, measure usage, then add as needed. Avoid C-level changes unless scaling to production.
