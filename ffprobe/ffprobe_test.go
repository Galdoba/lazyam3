package ffprobe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestHelperProcess — не настоящий тест. Он запускается как дочерний процесс
// (см. helperProber) и эмулирует поведение ffprobe в зависимости от HELPER_MODE.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	switch os.Getenv("HELPER_MODE") {
	case "success":
		fmt.Print(`{"streams":[{"codec_name":"h264"}],"format":{"duration":"10.0"}}`)
	case "invalid_json":
		fmt.Print("this is not json")
	case "stderr_error":
		fmt.Fprint(os.Stderr, "boom: input file is corrupted")
		os.Exit(1)
	case "json_and_stderr_error":
		fmt.Print(`{"streams":[]}`)
		fmt.Fprint(os.Stderr, "boom")
		os.Exit(1)
	case "sleep":
		time.Sleep(30 * time.Second)
	default:
		fmt.Fprintf(os.Stderr, "unknown HELPER_MODE %q\n", os.Getenv("HELPER_MODE"))
		os.Exit(2)
	}
	// Все «успешные» ветки завершаем через os.Exit, чтобы тестовый раннер
	// не допечатал "PASS"/"ok" в stdout и не сломал JSON.
	os.Exit(0)
}

// helperProber возвращает Prober, чей бинарь — сам тестовый бинарь,
// запускаемый в режиме «эмулятора ffprobe».
func helperProber(t *testing.T, mode string) *Prober {
	t.Helper()
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	t.Setenv("HELPER_MODE", mode)

	// Подменяем «флаги ffprobe» на флаги go test: дочерний процесс должен
	// выполнить только TestHelperProcess. "--" отделяет флаги от filePath.
	old := ffprobeScanArgs
	ffprobeScanArgs = []string{"-test.run=TestHelperProcess", "--"}
	t.Cleanup(func() { ffprobeScanArgs = old })

	return NewProber(os.Args[0])
}

func mkFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.mkv")
	if err := os.WriteFile(path, []byte("not a real video"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNewProber_DefaultPath(t *testing.T) {
	if got := NewProber("").binPath; got != "ffprobe" {
		t.Fatalf("binPath = %q, want %q", got, "ffprobe")
	}
}

func TestNewProber_CustomPath(t *testing.T) {
	const custom = "/opt/bin/ffprobe"
	if got := NewProber(custom).binPath; got != custom {
		t.Fatalf("binPath = %q, want %q", got, custom)
	}
}

func TestProbeFile_StatError(t *testing.T) {
	p := NewProber("ffprobe")
	missing := filepath.Join(t.TempDir(), "nope.mkv")

	_, err := p.ProbeFile(context.Background(), missing)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "ffprobe: stat") {
		t.Fatalf("unexpected error text: %v", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want os.ErrNotExist, got %v", err)
	}
}

func TestProbeFile_Success(t *testing.T) {
	p := helperProber(t, "success")
	file := mkFile(t)

	out, err := p.ProbeFile(context.Background(), file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !json.Valid(out) {
		t.Fatalf("invalid json: %s", out)
	}

	var got struct {
		Streams []struct {
			CodecName string `json:"codec_name"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Streams) != 1 || got.Streams[0].CodecName != "h264" {
		t.Fatalf("unexpected payload: %s", out)
	}
}

func TestProbeFile_InvalidJSON(t *testing.T) {
	p := helperProber(t, "invalid_json")
	file := mkFile(t)

	_, err := p.ProbeFile(context.Background(), file)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("unexpected error text: %v", err)
	}
}

func TestProbeFile_CommandError(t *testing.T) {
	p := helperProber(t, "stderr_error")
	file := mkFile(t)

	_, err := p.ProbeFile(context.Background(), file)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "boom: input file is corrupted") {
		t.Fatalf("stderr not propagated to error: %v", err)
	}
}

func TestProbeFile_ExitCodeTakesPrecedenceOverStdout(t *testing.T) {
	p := helperProber(t, "json_and_stderr_error")
	file := mkFile(t)

	_, err := p.ProbeFile(context.Background(), file)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	// Даже если stdout — валидный JSON, ненулевой код возврата — ошибка.
	if !strings.Contains(err.Error(), "ffprobe:") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProbeFile_ContextCancelled(t *testing.T) {
	p := helperProber(t, "sleep")
	file := mkFile(t)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := p.ProbeFile(ctx, file)
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("want error, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ProbeFile did not return after context cancellation")
	}
}
