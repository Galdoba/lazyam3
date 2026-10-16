package logging

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/Galdoba/lazyam3/internal/app/config"
)

// Logger — фасад над *slog.Logger с явно определённой поверхностью.
// Никакого неявного продвижения методов из встроенного типа —
// поэтому With и Errorf гарантированно возвращают ожидаемое.
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
		inner:   slog.New(slog.NewMultiHandler(handlers...)),
		closers: closers,
	}, nil
}

// --- логирование ---

func (l *Logger) Debug(msg string, args ...any) { l.inner.Debug(msg, args...) }
func (l *Logger) Info(msg string, args ...any)  { l.inner.Info(msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.inner.Warn(msg, args...) }
func (l *Logger) Error(msg string, args ...any) { l.inner.Error(msg, args...) }

// Errorf логирует ошибку на уровне Error и возвращает её же.
func (l *Logger) Errorf(format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	l.inner.Error(err.Error())
	return err
}

// With возвращает дочерний логгер с атрибутами. Тот же тип *Logger,
// поэтому Errorf доступен и на scoped-логгерах.
//
// Дочерний логгер не владеет ресурсами — closers пустой, Close на нём
// безопасен (no-op). Ресурсы закрываются только на исходном *Logger
// через AppModules.Shutdown.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{inner: l.inner.With(args...)}
}

// Slog даёт доступ к внутреннему *slog.Logger — escape hatch на случай,
// если нужно передать логгер в стороннюю библиотеку, принимающую *slog.Logger.
func (l *Logger) Slog() *slog.Logger { return l.inner }

// Close закрывает ресурсы. Идемпотентен.
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

	termHandler, err := newTermHandler(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create term handler: %w", err)
	}
	if termHandler != nil {
		handlers = append(handlers, termHandler)
	}

	if len(handlers) == 0 {
		return nil, nil, fmt.Errorf("created nil-handlers only")
	}

	return handlers, closers, nil
}

func newTermHandler(cfg config.Logging) (slog.Handler, error) {
	if !cfg.Terminal {
		return nil, nil
	}
	opts := &slog.HandlerOptions{
		Level: levelFromString(cfg.Level),
	}
	return slog.NewJSONHandler(os.Stderr, opts), nil
}

func newFileHandler(cfg config.Logging) (slog.Handler, io.Closer, error) {
	if cfg.Path == "" {
		return nil, nil, nil
	}
	logfile, err := os.OpenFile(cfg.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
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
