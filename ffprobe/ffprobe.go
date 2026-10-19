package ffprobe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// ---------------------------------------------------------------------------
// Type definitions
// ---------------------------------------------------------------------------

type Prober struct {
	binPath string
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

// ffprobe command and default scan arguments.
const (
	ffprobeCmd = "ffprobe"
)

var ffprobeScanArgs = []string{
	"-v", "quiet",
	"-print_format", "json",
	"-show_streams",
	"-show_format",
}

func NewProber(binPath string) *Prober {
	if binPath == "" {
		binPath = "ffprobe"
	}
	return &Prober{binPath: binPath}
}

func (p *Prober) ProbeFile(ctx context.Context, filePath string) (json.RawMessage, error) {
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("ffprobe: stat %q: %w", filePath, err)
	}

	cmd := exec.CommandContext(ctx, p.binPath, append(ffprobeScanArgs, filePath)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffprobe: %w (stderr: %s)", err, stderr.String())
	}

	out := stdout.Bytes()
	if !json.Valid(out) {
		return nil, fmt.Errorf("ffprobe: invalid JSON on stdout (stderr: %s)", stderr.String())
	}
	return json.RawMessage(out), nil
}
