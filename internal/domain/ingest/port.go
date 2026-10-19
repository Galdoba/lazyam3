package ingest

// IngestPort defines the contract for file ingestion.
// Infrastructure implements this interface; domain only depends on the interface.
type IngestPort interface {
	// Ingest scans a directory, groups files by product ID, probes streams,
	// and returns MediaUnit. Infrastructure handles the actual IO.
	Ingest(dir string) (MediaUnit, error)
}
