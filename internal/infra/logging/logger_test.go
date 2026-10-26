package logging

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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

func TestNewTermHandler_ColorText(t *testing.T) {
	data := captureTermOutput(t, config.Logging{Terminal: true, Format: "text", Color: true})
	if !bytes.Contains(data, []byte("level=\x1b[32mINFO\x1b[0m")) {
		t.Errorf("color=true text output missing colorized level:\n%q", data)
	}
}

func TestNewTermHandler_ColorDisabled(t *testing.T) {
	data := captureTermOutput(t, config.Logging{Terminal: true, Format: "text", Color: false})
	if bytes.Contains(data, []byte("\x1b")) {
		t.Errorf("color=false text output contains ANSI escapes:\n%q", data)
	}
	if !bytes.Contains(data, []byte("level=INFO")) {
		t.Errorf("expected plain level field, got:\n%q", data)
	}
}

func TestNewTermHandler_ColorJSONIgnored(t *testing.T) {
	data := captureTermOutput(t, config.Logging{Terminal: true, Format: "json", Color: true})
	if bytes.Contains(data, []byte("\x1b")) {
		t.Errorf("json output contains ANSI escapes:\n%q", data)
	}
	if !bytes.Contains(data, []byte(`"msg":"format test message"`)) {
		t.Errorf("expected JSON output, got:\n%s", data)
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

const testTemplate = "{time} [{level}] {module}: {message} {attrs}"

func testRecordTime() time.Time {
	return time.Date(2026, time.October, 25, 22, 9, 52, 794000000, time.UTC)
}

type renderOpts struct {
	template        string
	timestampFormat string
	color           bool
	groups          []string
	preAttrs        []slog.Attr
	msg             string
	attrs           []slog.Attr
}

func renderCustom(t *testing.T, o renderOpts) string {
	t.Helper()
	var buf bytes.Buffer
	h := newCustomTextHandler(textHandlerOptions{
		level:           slog.LevelDebug,
		timestampFormat: o.timestampFormat,
		template:        o.template,
		color:           o.color,
	}, &buf)
	handler := h
	for _, g := range o.groups {
		handler = handler.WithGroup(g)
	}
	if len(o.preAttrs) > 0 {
		handler = handler.WithAttrs(o.preAttrs)
	}
	rec := slog.NewRecord(testRecordTime(), slog.LevelInfo, o.msg, 0)
	rec.AddAttrs(o.attrs...)
	if err := handler.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	return buf.String()
}

func TestCustomHandler_RenderBasic(t *testing.T) {
	got := renderCustom(t, renderOpts{
		template:        testTemplate,
		timestampFormat: "06-01-02 15:04:05.000",
		groups:          []string{"pipeline"},
		msg:             "processing",
		attrs:           []slog.Attr{slog.String("dir", "/path")},
	})
	want := "26-10-25 22:09:52.794 [INFO] pipeline: processing dir=\"/path\"\n"
	if got != want {
		t.Errorf("render:\n got: %q\nwant: %q", got, want)
	}
}

func TestCustomHandler_RenderAttrs(t *testing.T) {
	got := renderCustom(t, renderOpts{
		template: "{attrs}",
		msg:      "unused",
		attrs: []slog.Attr{
			slog.String("name", "x"),
			slog.Int("id", 42),
			slog.Bool("flag", true),
			slog.Float64("ratio", 1.5),
		},
	})
	want := "name=\"x\" id=42 flag=true ratio=1.5\n"
	if got != want {
		t.Errorf("render:\n got: %q\nwant: %q", got, want)
	}
}

func TestCustomHandler_AccumulatedAttrs(t *testing.T) {
	got := renderCustom(t, renderOpts{
		template: "{message} {attrs}",
		preAttrs: []slog.Attr{slog.String("app", "cli")},
		msg:      "start",
		attrs:    []slog.Attr{slog.Int("id", 7)},
	})
	want := "start app=\"cli\" id=7\n"
	if got != want {
		t.Errorf("render:\n got: %q\nwant: %q", got, want)
	}
}

func TestCustomHandler_Module(t *testing.T) {
	tests := []struct {
		name   string
		groups []string
		want   string
	}{
		{"no group", nil, "processing\n"},
		{"one group", []string{"pipeline"}, "pipeline: processing\n"},
		{"nested groups", []string{"pipeline", "command"}, "pipeline.command: processing\n"},
	}
	for _, tc := range tests {
		got := renderCustom(t, renderOpts{
			template: "{module}: {message}",
			groups:   tc.groups,
			msg:      "processing",
		})
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestCustomHandler_GroupPrefixStripping(t *testing.T) {
	got := renderCustom(t, renderOpts{
		template: testTemplate,
		groups:   []string{"pipeline"},
		msg:      "processing",
		attrs:    []slog.Attr{slog.String("dir", "/path")},
	})
	if strings.Contains(got, "pipeline.dir") {
		t.Errorf("attr key must not repeat the group prefix: %q", got)
	}
	if !strings.Contains(got, "pipeline: processing dir=\"/path\"") {
		t.Errorf("expected module + unprefixed attr, got %q", got)
	}
}

func TestCustomHandler_Color(t *testing.T) {
	colored := renderCustom(t, renderOpts{template: "{level}", color: true, msg: "m"})
	if !strings.Contains(colored, "\x1b[32mINFO\x1b[0m") {
		t.Errorf("color=true: expected colorized level, got %q", colored)
	}
	plain := renderCustom(t, renderOpts{template: "{level}", color: false, msg: "m"})
	if plain != "INFO\n" {
		t.Errorf("color=false: got %q, want %q", plain, "INFO\n")
	}
}

func TestCustomHandler_TimestampFormat(t *testing.T) {
	short := renderCustom(t, renderOpts{
		template:        "{time} {level}",
		timestampFormat: "15:04:05",
		msg:             "m",
	})
	if short != "22:09:52 INFO\n" {
		t.Errorf("custom layout: got %q", short)
	}
	def := renderCustom(t, renderOpts{template: "{time} {level}", msg: "m"})
	if def != "2026/10/25 22:09:52.794 INFO\n" {
		t.Errorf("default layout: got %q", def)
	}
}

func TestCustomHandler_Message(t *testing.T) {
	got := renderCustom(t, renderOpts{
		template: testTemplate,
		groups:   []string{"pipeline"},
		msg:      "scan failed, using fallback",
	})
	// message is plain text; with the default timestamp layout the time differs —
	// check structure instead of the exact timestamp.
	if !strings.HasSuffix(got, "[INFO] pipeline: scan failed, using fallback\n") {
		t.Errorf("message with separators: got %q", got)
	}
}

func TestCustomHandler_EmptyMessageAndAttrsSwallowSeparators(t *testing.T) {
	got := renderCustom(t, renderOpts{
		template:        testTemplate,
		timestampFormat: "15:04:05",
		groups:          []string{"pipeline"},
		msg:             "",
	})
	want := "22:09:52 [INFO] pipeline\n"
	if got != want {
		t.Errorf("empty message+attrs: got %q, want %q", got, want)
	}

	noGroup := renderCustom(t, renderOpts{
		template:        testTemplate,
		timestampFormat: "15:04:05",
		msg:             "processing",
	})
	want = "22:09:52 [INFO] processing\n"
	if noGroup != want {
		t.Errorf("empty module+attrs: got %q, want %q", noGroup, want)
	}

	trailing := renderCustom(t, renderOpts{
		template: "{message} {attrs}",
		msg:      "processing",
	})
	if trailing != "processing\n" {
		t.Errorf("empty attrs must not leave a trailing space: %q", trailing)
	}
}

func TestCustomHandler_TemplateReordering(t *testing.T) {
	got := renderCustom(t, renderOpts{
		template:        "{message} {level} {time}",
		timestampFormat: "15:04:05",
		msg:             "processing",
	})
	want := "processing INFO 22:09:52\n"
	if got != want {
		t.Errorf("reordered template: got %q, want %q", got, want)
	}
}

func TestCustomHandler_UnknownPlaceholder(t *testing.T) {
	got := renderCustom(t, renderOpts{
		template:        "{time} {foo} {level}",
		timestampFormat: "15:04:05",
		msg:             "m",
	})
	want := "22:09:52 {foo} INFO\n"
	if got != want {
		t.Errorf("unknown placeholder: got %q, want %q", got, want)
	}
}

func TestCustomHandler_NestedGroupAttr(t *testing.T) {
	got := renderCustom(t, renderOpts{
		template: "{attrs}",
		msg:      "m",
		attrs:    []slog.Attr{slog.Group("inner", slog.String("k", "v"))},
	})
	want := "inner.k=\"v\"\n"
	if got != want {
		t.Errorf("nested group attr: got %q, want %q", got, want)
	}
}

type syncWriter struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (w *syncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

func TestCustomHandler_ConcurrentHandle(t *testing.T) {
	w := &syncWriter{}
	h := newCustomTextHandler(textHandlerOptions{
		level:    slog.LevelDebug,
		template: testTemplate,
	}, w)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := slog.NewRecord(testRecordTime(), slog.LevelInfo, "concurrent", 0)
			rec.AddAttrs(slog.Int("id", i))
			_ = h.Handle(context.Background(), rec)
		}()
	}
	wg.Wait()
	if got := strings.Count(w.b.String(), "concurrent"); got != 50 {
		t.Errorf("expected 50 rendered lines, got %d", got)
	}
}

func TestNewTermHandler_CustomFormatText(t *testing.T) {
	data := captureTermOutput(t, config.Logging{
		Terminal:        true,
		Format:          "text",
		Color:           false,
		FormatText:      "{time} {level} {message} {attrs}",
		TimestampFormat: "15:04:05",
	})
	if !regexp.MustCompile(`^\d{2}:\d{2}:\d{2} INFO format test message key="value"\n$`).Match(data) {
		t.Errorf("custom format output: %q", data)
	}
}

func TestNewTermHandler_FallbackToTextHandler(t *testing.T) {
	data := captureTermOutput(t, config.Logging{Terminal: true, Format: "text", Color: false})
	if !bytes.Contains(data, []byte("level=INFO")) {
		t.Errorf("empty format_text must fall back to slog.TextHandler, got: %q", data)
	}
}

func TestNewTermHandler_FormatTextIgnoredForJSON(t *testing.T) {
	data := captureTermOutput(t, config.Logging{
		Terminal:   true,
		Format:     "json",
		FormatText: "{time} {level} {message}",
	})
	if !bytes.Contains(data, []byte(`"msg":"format test message"`)) {
		t.Errorf("format_text must be ignored for JSON output, got: %q", data)
	}
	if bytes.Contains(data, []byte("INFO format test message")) {
		t.Errorf("format_text must not switch JSON output to text, got: %q", data)
	}
}
