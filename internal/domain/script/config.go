package script

import "github.com/Galdoba/lazyam3/internal/domain/ingest"

// ScriptConfig is the domain input for building a rendering script.
type ScriptConfig struct {
	Files            []ingest.Stream
	StreamSelections []StreamSelection
	Destination      string
}
