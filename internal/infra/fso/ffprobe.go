package fso

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/Galdoba/lazyam3/internal/infra/logging"
)

// ---------------------------------------------------------------------------
// Type definitions
// ---------------------------------------------------------------------------

type Prober struct {
	binPath string
	logger  *logging.Logger
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

// Default ffprobe binary name.
const defaultBinPath = "ffprobe"

var ffprobeScanArgs = []string{
	"-v", "quiet",
	"-print_format", "json",
	"-show_streams",
	"-show_format",
}

// NewProber creates a Prober using the given ffprobe binary path.
func NewProber(binPath string, logger *logging.Logger) *Prober {
	if binPath == "" {
		binPath = defaultBinPath
	}
	return &Prober{binPath: binPath, logger: logger}
}

func (p *Prober) ProbeFile(filePath string) (json.RawMessage, error) {
	if p.logger != nil {
		p.logger.Debug("ffprobe probe start", "file", filePath)
	}
	if _, err := os.Stat(filePath); err != nil {
		if p.logger != nil {
			p.logger.Debug("ffprobe probe stat failed", "file", filePath, "error", err)
		}
		return nil, fmt.Errorf("ffprobe: stat %q: %w", filePath, err)
	}

	cmd := exec.Command(p.binPath, append(ffprobeScanArgs, filePath)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if p.logger != nil {
			p.logger.Debug("ffprobe probe failed", "file", filePath, "error", err)
		}
		return nil, fmt.Errorf("ffprobe: %w (stderr: %s)", err, stderr.String())
	}

	out := stdout.Bytes()
	if !json.Valid(out) {
		if p.logger != nil {
			p.logger.Debug("ffprobe probe invalid json", "file", filePath)
		}
		return nil, fmt.Errorf("ffprobe: invalid JSON on stdout (stderr: %s)", stderr.String())
	}
	if p.logger != nil {
		p.logger.Debug("ffprobe probe success", "file", filePath)
	}
	return json.RawMessage(out), nil
}
