package logging

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Galdoba/lazyam3/internal/infra/config"
)

func TestNewLogger_TerminalOnly(t *testing.T) {
	cfg := config.Logging{Terminal: true, Level: "info", Path: ""}
	l, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()
	if l.inner == nil {
		t.Fatal("expected non-nil inner logger")
	}
}

func TestNewLogger_FileOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	cfg := config.Logging{Terminal: false, Level: "info", Path: path}
	l, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()
	if l.inner == nil {
		t.Fatal("expected non-nil inner logger")
	}
}

func TestNewLogger_Both(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	cfg := config.Logging{Terminal: true, Level: "info", Path: path}
	l, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()
	if l.inner == nil {
		t.Fatal("expected non-nil inner logger")
	}
}

func TestNewLogger_NilHandlers(t *testing.T) {
	cfg := config.Logging{Terminal: false, Path: ""}
	_, err := NewLogger(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewLogger_InvalidLogLevel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	cfg := config.Logging{Terminal: false, Path: path, Level: "bogus"}
	// Should still succeed — levelFromString defaults to Info for unknown levels.
	l, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()
	if l.inner == nil {
		t.Fatal("expected non-nil inner logger")
	}
}

func TestLogger_Close(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	cfg := config.Logging{Terminal: false, Path: path}
	l, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}

	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Idempotent — closing again should be safe.
	if err := l.Close(); err != nil {
		t.Fatalf("Close (idempotent): %v", err)
	}
}

func TestLogger_CloseNil(t *testing.T) {
	var l *Logger
	if err := l.Close(); err != nil {
		t.Fatalf("Close on nil: %v", err)
	}
}

func TestLogger_With(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	cfg := config.Logging{Terminal: false, Path: path}
	l, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()

	child := l.With("key", "value")
	if child == l {
		t.Error("With should return a different Logger instance")
	}
}

func TestLogger_Group(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	cfg := config.Logging{Terminal: false, Path: path}
	l, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()

	child := l.Group("command")
	if child == l {
		t.Error("Group should return a different Logger instance")
	}

	child.Info("processed", "unit_id", 42)
	errVal := child.Errorf("grouped errorf: %d", 42)
	if errVal == nil {
		t.Fatal("Errorf should be available on grouped child logger")
	}

	chained := l.Group("pipeline").With("id", 1)
	chained.Warn("chained")

	if err := child.Close(); err != nil {
		t.Fatalf("Close on grouped child: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	for _, want := range []string{
		`"command":{"unit_id":42}`,
		`"msg":"grouped errorf: 42"`,
		`"pipeline":{"id":1}`,
	} {
		if !bytes.Contains(data, []byte(want)) {
			t.Errorf("output missing %s:\n%s", want, data)
		}
	}
}

func TestLogger_Methods(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	cfg := config.Logging{Terminal: false, Path: path}
	l, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()

	// Verify methods don't panic.
	l.Debug("debug message")
	l.Info("info message")
	l.Warn("warn message")
	l.Error("error message")

	errVal := l.Errorf("errorf: %d", 42)
	if errVal == nil {
		t.Fatal("Errorf should return a non-nil error")
	}
}

func TestLogger_Slog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	cfg := config.Logging{Terminal: false, Path: path}
	l, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()

	s := l.Slog()
	if s == nil {
		t.Fatal("Slog() returned nil")
	}
}

func TestMultiHandler_Enabled(t *testing.T) {
	mh := multiHandler{handlers: []slog.Handler{
		slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}),
	}}

	if !mh.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("Expected Enabled to return true for Debug")
	}
	if !mh.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Expected Enabled to return true for Info")
	}
	if !mh.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("Expected Enabled to return true for Warn")
	}
	if !mh.Enabled(context.Background(), slog.LevelError) {
		t.Error("Expected Enabled to return true for Error")
	}
}

func TestMultiHandler_Handle(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	s := time.Now()
	mh := multiHandler{handlers: []slog.Handler{
		slog.NewTextHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelDebug}),
		slog.NewTextHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelDebug}),
	}}

	rec := slog.NewRecord(s, slog.LevelInfo, "test message", 0)
	err := mh.Handle(context.Background(), rec)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if buf1.Len() == 0 {
		t.Error("buf1 should have written output")
	}
	if buf2.Len() == 0 {
		t.Error("buf2 should have written output")
	}
}

func TestMultiHandler_Handle_Error(t *testing.T) {
	mh := multiHandler{handlers: []slog.Handler{
		slog.NewTextHandler(&errWriter{}, &slog.HandlerOptions{Level: slog.LevelDebug}),
	}}

	rec := slog.NewRecord(time.Now(), slog.LevelError, "error message", 0)
	err := mh.Handle(context.Background(), rec)
	if err == nil {
		t.Fatal("expected error from Handle")
	}
}

func TestMultiHandler_WithAttrs(t *testing.T) {
	mh := multiHandler{handlers: []slog.Handler{
		slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}),
	}}

	attrs := mh.WithAttrs([]slog.Attr{slog.String("key", "val")})
	if attrs == nil {
		t.Fatal("WithAttrs returned nil")
	}
	// The returned handler should still be a multiHandler.
	if _, ok := attrs.(*multiHandler); !ok {
		t.Errorf("WithAttrs returned %T, want *multiHandler", attrs)
	}
}

func TestMultiHandler_WithGroup(t *testing.T) {
	mh := multiHandler{handlers: []slog.Handler{
		slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}),
	}}

	group := mh.WithGroup("mygroup")
	if group == nil {
		t.Fatal("WithGroup returned nil")
	}
	if _, ok := group.(*multiHandler); !ok {
		t.Errorf("WithGroup returned %T, want *multiHandler", group)
	}
}

func TestNewLogger_LevelDebug(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	cfg := config.Logging{Terminal: false, Path: path, Level: "debug"}
	l, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()
	// Debug records should be visible.
	l.Debug("debug test")
}

func TestNewLogger_LevelError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	cfg := config.Logging{Terminal: false, Path: path, Level: "error"}
	l, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()
	// Debug should be filtered at error level.
	l.Debug("debug filtered")
	l.Error("error visible")
}

func TestNewLogger_LevelWarn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	cfg := config.Logging{Terminal: false, Path: path, Level: "warn"}
	l, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()
	l.Warn("warn visible")
	l.Error("error visible")
}

// errWriter is an io.Writer that always returns an error.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, os.ErrInvalid
}

func captureTermOutput(t *testing.T, cfg config.Logging) []byte {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = old }()

	handler := newTermHandler(cfg)
	if handler == nil {
		w.Close()
		t.Fatal("newTermHandler returned nil for Terminal=true")
	}
	logger := slog.New(handler)
	logger.Info("format test message", "key", "value")

	w.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return data
}

func TestNewTermHandler_FormatText(t *testing.T) {
	data := captureTermOutput(t, config.Logging{Terminal: true, Format: "text"})
	if !bytes.Contains(data, []byte("format test message")) {
		t.Errorf("text output missing message:\n%s", data)
	}
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
		t.Errorf("text format produced JSON output:\n%s", data)
	}
	if bytes.Contains(data, []byte(`"msg":`)) {
		t.Errorf("text format produced JSON key msg:\n%s", data)
	}
}

func TestNewTermHandler_FormatJSON(t *testing.T) {
	for _, format := range []string{"json", ""} {
		data := captureTermOutput(t, config.Logging{Terminal: true, Format: format})
		if !bytes.Contains(data, []byte(`"msg":"format test message"`)) {
			t.Errorf("format %q: expected JSON output, got:\n%s", format, data)
		}
	}
}

func TestNewTermHandler_FormatCaseInsensitive(t *testing.T) {
	data := captureTermOutput(t, config.Logging{Terminal: true, Format: "TEXT"})
	if bytes.Contains(data, []byte(`"msg":`)) {
		t.Errorf("format TEXT should produce text output, got:\n%s", data)
	}
}

func TestNewTermHandler_NoTerminal(t *testing.T) {
	if h := newTermHandler(config.Logging{Terminal: false, Format: "text"}); h != nil {
		t.Error("expected nil handler when Terminal=false")
	}
}

func TestNewLogger_TerminalTextFormat(t *testing.T) {
	// NewLogger with only terminal + text format should succeed and write text to stderr.
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	os.Stderr = w

	cfg := config.Logging{Terminal: true, Level: "info", Path: "", Format: "text"}
	l, err := NewLogger(cfg)
	if err != nil {
		w.Close()
		os.Stderr = old
		t.Fatalf("NewLogger: %v", err)
	}
	l.Info("pipeline start")

	w.Close()
	os.Stderr = old
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if l.closers != nil {
		t.Errorf("expected closers to be nil after Close")
	}
	if !bytes.Contains(data, []byte("pipeline start")) {
		t.Errorf("expected text line in terminal output, got:\n%s", data)
	}
}
