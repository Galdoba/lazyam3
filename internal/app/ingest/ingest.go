package ingest

import (
	"github.com/Galdoba/lazyam3/internal/domain/ingest"
)

// Ingest implements ingest.IngestPort with a minimal stub.
// Real implementation will scan, probe, and parse files.
type Ingest struct{}

// Ingest scans a directory and returns MediaUnits.
// TODO: implement real scan/probe/parse logic.
func (s *Ingest) Ingest(dir string) ([]ingest.MediaUnit, error) {
	// Stub: return empty for now.
	return nil, nil
}
