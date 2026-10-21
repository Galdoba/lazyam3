package ingest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/Galdoba/lazyam3/internal/infra/fso"
)

// Ingest implements ingest.Organizer by scanning a directory, probing each
// regular file with ffprobe, and returning a MediaUnit.
type Ingest struct {
	ffprobeBinary string
}

// New creates an Ingest that uses the given ffprobe binary path.
func New(ffprobeBinaryPath string) *Ingest {
	return &Ingest{ffprobeBinary: ffprobeBinaryPath}
}

// Ingest scans dir for regular files, probes each with ffprobe, and returns
// a MediaUnit. Non-regular entries (directories, symlinks, etc.) are skipped.
// A probe failure fails the entire directory immediately.
func (ing *Ingest) Ingest(dir string) (ingest.MediaUnit, error) {
	fls, err := os.ReadDir(dir)
	if err != nil {
		return ingest.MediaUnit{}, err
	}
	dirData := make(map[string][]byte)
	prober := fso.NewProber(ing.ffprobeBinary)
	for _, f := range fls {
		if !f.Type().IsRegular() {
			continue
		}
		path := filepath.Join(dir, f.Name())
		data, err := prober.ProbeFile(path)
		if err != nil {
			return ingest.MediaUnit{}, fmt.Errorf("failed to probe file %q: %w", path, err)
		}
		dirData[f.Name()] = data
	}
	return ingest.NewMediaUnit(dirData)
}
