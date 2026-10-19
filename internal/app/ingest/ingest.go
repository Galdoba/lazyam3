package ingest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Galdoba/lazyam3/ffprobe"
	"github.com/Galdoba/lazyam3/internal/domain/ingest"
)

// Ingest implements ingest.IngestPort with a minimal stub.
// Real implementation will scan, probe, and parse files.
type Ingest struct{}

// Ingest scans a directory and returns MediaUnits.
// TODO: implement real scan/probe/parse logic.
func (s *Ingest) Ingest(dir string) (ingest.MediaUnit, error) {
	// scetch: return empty for now.
	// TODO: sort filenames alphabeticaly
	fmt.Println("ingesting", dir)
	fmt.Println("get files")
	fls, err := os.ReadDir(dir)
	if err != nil {
		return ingest.MediaUnit{}, err
	}
	dirData := make(map[string][]byte)
	fmt.Println("probe files")
	for _, f := range fls {
		data, err := ffprobe.NewProber("").ProbeFile(context.TODO(), filepath.Join(dir, f.Name()))
		if err != nil {
			return ingest.MediaUnit{}, err
		}
		fmt.Println("aquired data for", f.Name())
		dirData[f.Name()] = data
	}
	fmt.Println("create MediaUnit")
	mu, err := ingest.NewMediaUnit(dirData)

	return mu, err
}
