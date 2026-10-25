package ingest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/Galdoba/lazyam3/internal/infra/fso"
)

// Ingest implements ingest.Organizer by scanning a directory, probing each
// regular file with ffprobe, and returning a MediaUnit.
type Ingest struct {
	ffprobeBinary string
}

// New creates an Ingest that uses the given ffprobe binary path.
//log=debug msg="ingest created with ffprobe=%s", ffprobeBinaryPath
func New(ffprobeBinaryPath string) *Ingest {
	return &Ingest{ffprobeBinary: ffprobeBinaryPath}
}

// Ingest scans dir for regular files, probes each with ffprobe, and returns
// a MediaUnit. Non-regular entries (directories, symlinks, etc.) are skipped.
// Subtitle sidecar files (.srt, .ass, .ssa) that ffprobe cannot probe are
// recorded in Filenames so the pipeline can later classify them as SRT or HSUB.
//log=info msg="ingesting directory: %s", dir
func (ing *Ingest) Ingest(dir string) (ingest.MediaUnit, error) {
	fls, err := os.ReadDir(dir)
	if err != nil {
		//log=error msg="failed to read directory %s: %v", dir, err
		return ingest.MediaUnit{}, err
	}
	dirData := make(map[string][]byte)
	sidecars := make([]string, 0)
	prober := fso.NewProber(ing.ffprobeBinary)
	for _, f := range fls {
		if !f.Type().IsRegular() {
			continue
		}
		path := filepath.Join(dir, f.Name())
		data, err := prober.ProbeFile(path)
		if err != nil {
			if isSubtitle(f.Name()) {
				sidecars = append(sidecars, f.Name())
				continue
			}
			return ingest.MediaUnit{}, fmt.Errorf("failed to probe file %q: %w", path, err)
		//log=error msg="failed to probe file %q: %v", path, err
		}
		dirData[f.Name()] = data
	}
	mu, err := ingest.NewMediaUnit(dirData)
	if err != nil {
		//log=error msg="failed to parse data for unit: %v", err
		return ingest.MediaUnit{}, err
	}
	mu.Filenames = append(mu.Filenames, sidecars...)
	return mu, nil
}

// isSubtitle returns true when the filename ends with a known subtitle extension.
func isSubtitle(name string) bool {
	l := strings.ToLower(name)
	return strings.HasSuffix(l, ".srt") || strings.HasSuffix(l, ".ass") || strings.HasSuffix(l, ".ssa")
}
