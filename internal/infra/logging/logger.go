package logging

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

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
	handlers, closers, err := createHandlers(cfg)
	if err != nil {
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

// Slog provides access to the inner *slog.Logger — an escape hatch in case
// you need to pass the logger to a third-party library that accepts *slog.Logger.
func (l *Logger) Slog() *slog.Logger { return l.inner }

// Close closes resources. Idempotent.
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
	return slog.NewJSONHandler(os.Stderr, opts)
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
