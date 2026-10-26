package logging

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Galdoba/lazyam3/internal/infra/config"
)

// Logger is a facade over *slog.Logger with an explicitly defined surface.
// No implicit method promotion from the embedded type —
// so With and Errorf are guaranteed to return the expected type.
type Logger struct {
	inner   *slog.Logger
	closers []io.Closer
}

func NewLogger(cfg config.Logging) (*Logger, error) {
	//log=info msg="logger created with level=%s, format=%s", cfg.Level, cfg.Format
	handlers, closers, err := createHandlers(cfg)
	if err != nil {
		//log=error msg="failed to create logger: %v", err
		return nil, fmt.Errorf("failed to collect handlers: %w", err)
	}
	return &Logger{
		inner:   slog.New(&multiHandler{handlers: handlers}),
		closers: closers,
	}, nil
}

// --- logging ---

func (l *Logger) Debug(msg string, args ...any) { l.inner.Debug(msg, args...) }
func (l *Logger) Info(msg string, args ...any)  { l.inner.Info(msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.inner.Warn(msg, args...) }
func (l *Logger) Error(msg string, args ...any) { l.inner.Error(msg, args...) }

// Errorf logs an error at Error level and returns it.
func (l *Logger) Errorf(format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	l.inner.Error(err.Error())
	return err
}

// With returns a child logger with attributes. The same *Logger type,
// so Errorf is also available on scoped loggers.
//
// The child logger owns no resources — closers is empty, Close on it
// is safe (no-op). Resources are closed only on the original *Logger
// through AppModules.Shutdown.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{inner: l.inner.With(args...)}
}

// Group returns a child logger that prefixes attribute keys with a group name,
// preserving the typed surface (*Logger, Errorf available).
//
// The child logger owns no resources — Close on it is a no-op. Resources are
// closed only on the original *Logger through AppModules.Shutdown.
func (l *Logger) Group(name string) *Logger {
	return &Logger{inner: l.inner.WithGroup(name)}
}

// Slog provides access to the inner *slog.Logger — an escape hatch in case
// you need to pass the logger to a third-party library that accepts *slog.Logger.
func (l *Logger) Slog() *slog.Logger { return l.inner }

// Close closes resources. Idempotent.
//log=info msg="logger closed"
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	var errs []error
	for _, c := range l.closers {
		if c == nil {
			continue
		}
		if err := c.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	l.closers = nil
	return errors.Join(errs...)
}

func createHandlers(cfg config.Logging) ([]slog.Handler, []io.Closer, error) {
	handlers := []slog.Handler{}
	closers := []io.Closer{}

	fileHandler, fileCloser, err := newFileHandler(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create file handler: %w", err)
	}
	if fileHandler != nil {
		handlers = append(handlers, fileHandler)
	}
	if fileCloser != nil {
		closers = append(closers, fileCloser)
	}

	termHandler := newTermHandler(cfg)
	if termHandler != nil {
		handlers = append(handlers, termHandler)
	}

	if len(handlers) == 0 {
		//log=error msg="no logging handlers created"
		return nil, nil, fmt.Errorf("created nil-handlers only")
	}

	return handlers, closers, nil
}

func newTermHandler(cfg config.Logging) slog.Handler {
	if !cfg.Terminal {
		return nil
	}
	opts := &slog.HandlerOptions{
		Level: levelFromString(cfg.Level),
	}
	if strings.EqualFold(cfg.Format, "text") {
		if cfg.FormatText != "" {
			// The custom handler colorizes the level itself — colorLevelWriter
			// stays reserved for the slog.TextHandler fallback.
			return newCustomTextHandler(textHandlerOptions{
				level:           levelFromString(cfg.Level),
				timestampFormat: cfg.TimestampFormat,
				template:        cfg.FormatText,
				color:           cfg.Color,
			}, os.Stderr)
		}
		var w io.Writer = os.Stderr
		if cfg.Color {
			// slog.TextHandler escapes control characters (strconv.Quote),
			// so ANSI codes cannot be injected via ReplaceAttr — the level
			// is colorized on the writer instead.
			w = &colorLevelWriter{w: w}
		}
		return slog.NewTextHandler(w, opts)
	}
	return slog.NewJSONHandler(os.Stderr, opts)
}

var levelFieldRe = regexp.MustCompile(`^(time=\S+ )?level=([A-Z]+)`)

var levelColors = map[string]string{
	"DEBUG": "\x1b[36m",
	"INFO":  "\x1b[32m",
	"WARN":  "\x1b[33m",
	"ERROR": "\x1b[31m",
}

type colorLevelWriter struct{ w io.Writer }

func (cw *colorLevelWriter) Write(p []byte) (int, error) {
	if _, err := cw.w.Write(colorizeLevel(p)); err != nil {
		return 0, err
	}
	return len(p), nil
}

func colorizeLevel(p []byte) []byte {
	loc := levelFieldRe.FindSubmatchIndex(p)
	if loc == nil {
		return p
	}
	color, ok := levelColors[string(p[loc[4]:loc[5]])]
	if !ok {
		return p
	}
	out := make([]byte, 0, len(p)+20)
	out = append(out, p[:loc[4]]...)
	out = append(out, color...)
	out = append(out, p[loc[4]:loc[5]]...)
	out = append(out, "\x1b[0m"...)
	out = append(out, p[loc[5]:]...)
	return out
}

const defaultTimestampLayout = "2006/01/02 15:04:05.000"

var templatePlaceholders = map[string]bool{
	"time":    true,
	"level":   true,
	"module":  true,
	"message": true,
	"attrs":   true,
}

type textHandlerOptions struct {
	level           slog.Level
	timestampFormat string
	template        string
	color           bool
}

type textTemplateSegment struct {
	text        string
	placeholder string
}

// customTextHandler renders records from a template of literal text and
// {time} {level} {module} {message} {attrs} placeholders. The handler is
// state-immutable after construction — Handle uses only local state, so
// concurrent calls are safe without a mutex.
type customTextHandler struct {
	out    io.Writer
	opts   textHandlerOptions
	tmpl   []textTemplateSegment
	attrs  []slog.Attr
	groups []string
}

func newCustomTextHandler(opts textHandlerOptions, out io.Writer) slog.Handler {
	return &customTextHandler{
		out:  out,
		opts: opts,
		tmpl: parseTemplate(opts.template),
	}
}

// parseTemplate splits a template into literal and placeholder segments.
// Unknown placeholders stay literal text.
func parseTemplate(s string) []textTemplateSegment {
	var segs []textTemplateSegment
	var lit strings.Builder
	flush := func() {
		if lit.Len() > 0 {
			segs = append(segs, textTemplateSegment{text: lit.String()})
			lit.Reset()
		}
	}
	for i := 0; i < len(s); {
		if s[i] != '{' {
			lit.WriteByte(s[i])
			i++
			continue
		}
		j := strings.IndexByte(s[i+1:], '}')
		if j < 0 {
			lit.WriteByte(s[i])
			i++
			continue
		}
		name := s[i+1 : i+1+j]
		if !templatePlaceholders[name] {
			lit.WriteString(s[i : i+j+2])
			i += j + 2
			continue
		}
		flush()
		segs = append(segs, textTemplateSegment{placeholder: name})
		i += j + 2
	}
	flush()
	return segs
}

func (h *customTextHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.opts.level
}

func (h *customTextHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	pending := ""
	dropNext := false
	for _, seg := range h.tmpl {
		if seg.placeholder == "" {
			if dropNext && isSeparatorLiteral(seg.text) {
				dropNext = false
				continue
			}
			dropNext = false
			pending += seg.text
			continue
		}
		dropNext = false
		val := h.renderPlaceholder(seg.placeholder, r)
		if val == "" {
			if isSeparatorLiteral(pending) {
				pending = ""
			}
			dropNext = true
			continue
		}
		b.WriteString(pending)
		b.WriteString(val)
		pending = ""
	}
	b.WriteString(pending)
	b.WriteByte('\n')
	_, err := io.WriteString(h.out, b.String())
	return err
}

// isSeparatorLiteral reports whether text consists only of spaces and colons,
// making it droppable when an adjacent placeholder renders empty.
func isSeparatorLiteral(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != ':' {
			return false
		}
	}
	return true
}

func (h *customTextHandler) renderPlaceholder(name string, r slog.Record) string {
	switch name {
	case "time":
		layout := h.opts.timestampFormat
		if layout == "" {
			layout = defaultTimestampLayout
		}
		return r.Time.Format(layout)
	case "level":
		s := r.Level.String()
		if h.opts.color {
			if c, ok := levelColors[s]; ok {
				return c + s + "\x1b[0m"
			}
		}
		return s
	case "module":
		return strings.Join(h.groups, ".")
	case "message":
		return r.Message
	case "attrs":
		return h.renderAttrs(r)
	}
	return ""
}

// renderAttrs formats accumulated and record attrs as space-separated
// key=value pairs without the group prefix — the group stack is shown
// by the {module} placeholder instead.
func (h *customTextHandler) renderAttrs(r slog.Record) string {
	var b strings.Builder
	var write func(string, slog.Value)
	write = func(k string, v slog.Value) {
		if k == "" {
			return
		}
		v = v.Resolve()
		if v.Kind() == slog.KindGroup {
			for _, a := range v.Group() {
				write(k+"."+a.Key, a.Value)
			}
			return
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(formatAttrValue(v))
	}
	for _, a := range h.attrs {
		write(a.Key, a.Value)
	}
	r.Attrs(func(a slog.Attr) bool {
		write(a.Key, a.Value)
		return true
	})
	return b.String()
}

func (h *customTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	nh := *h
	nh.attrs = make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	nh.attrs = append(nh.attrs, h.attrs...)
	nh.attrs = append(nh.attrs, attrs...)
	return &nh
}

func (h *customTextHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	nh := *h
	nh.groups = make([]string, 0, len(h.groups)+1)
	nh.groups = append(nh.groups, h.groups...)
	nh.groups = append(nh.groups, name)
	return &nh
}

func formatAttrValue(v slog.Value) string {
	if v.Kind() == slog.KindString {
		return strconv.Quote(v.String())
	}
	return fmt.Sprintf("%v", v.Any())
}

func newFileHandler(cfg config.Logging) (slog.Handler, io.Closer, error) {
	if cfg.Path == "" {
		return nil, nil, nil
	}
	logfile, err := os.OpenFile(cfg.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open log file: %w", err)
	}
	opts := &slog.HandlerOptions{
		Level: levelFromString(cfg.Level),
	}
	handler := slog.NewJSONHandler(logfile, opts)
	return handler, logfile, nil
}

func levelFromString(s string) slog.Level {
	switch s {
	case "error":
		return slog.LevelError
	case "warn":
		return slog.LevelWarn
	case "debug":
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

// multiHandler dispatches log records to multiple underlying handlers.
type multiHandler struct {
	handlers []slog.Handler
}

func (h *multiHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	for _, hh := range h.handlers {
		if hh.Enabled(ctx, lvl) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, hh := range h.handlers {
		if err := hh.Handle(ctx, r); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nh := &multiHandler{handlers: make([]slog.Handler, len(h.handlers))}
	for i, hh := range h.handlers {
		nh.handlers[i] = hh.WithAttrs(attrs)
	}
	return nh
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	nh := &multiHandler{handlers: make([]slog.Handler, len(h.handlers))}
	for i, hh := range h.handlers {
		nh.handlers[i] = hh.WithGroup(name)
	}
	return nh
}
