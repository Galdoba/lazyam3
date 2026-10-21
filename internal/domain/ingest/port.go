package ingest

// Organizer defines the contract for organizing files into MediaUnits.
// Infrastructure implements this interface; domain only depends on the interface.
type Organizer interface {
	// Ingest scans a directory, groups files by product ID, probes streams,
	// and returns MediaUnit. Infrastructure handles the actual IO.
	Ingest(dir string) (MediaUnit, error)
}
