package fso

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHelperProcess is not a real test. It runs as a child process
// (see helperProber) and emulates ffprobe behavior depending on HELPER_MODE.
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
	default:
		fmt.Fprintf(os.Stderr, "unknown HELPER_MODE %q\n", os.Getenv("HELPER_MODE"))
		os.Exit(2)
	}
	// All "successful" branches exit via os.Exit so the test runner
	// doesn't append "PASS"/"ok" to stdout and break JSON.
	os.Exit(0)
}

// helperProber returns a Prober whose binary is the test binary itself,
// run in "ffprobe emulator" mode.
func helperProber(t *testing.T, mode string) *Prober {
	t.Helper()
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	t.Setenv("HELPER_MODE", mode)

	// Replace "ffprobe flags" with go test flags: the child process must
	// run only TestHelperProcess. "--" separates flags from filePath.
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

	_, err := p.ProbeFile(missing)
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

	out, err := p.ProbeFile(file)
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

	_, err := p.ProbeFile(file)
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

	_, err := p.ProbeFile(file)
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

	_, err := p.ProbeFile(file)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	// Even if stdout is valid JSON, a non-zero exit code is an error.
	if !strings.Contains(err.Error(), "ffprobe:") {
		t.Fatalf("unexpected error: %v", err)
	}
}
