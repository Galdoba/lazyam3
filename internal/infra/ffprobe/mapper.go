package ffprobe

import (
	"github.com/Galdoba/lazyam3/internal/domain/ingest"
)

// Mapper converts ffprobe domain types to internal domain types.
// This is a stub — the real mapper will handle complex transformations.
type Mapper struct{}

// MapStream converts an ffprobe stream to a domain stream.
func (m *Mapper) MapStream(raw Stream) ingest.Stream {
	return ingest.Stream{
		Kind:     mapKind(raw.Codectype),
		Codec:    raw.CodecName,
		Language: ingest.Language(raw.Language),
		Width:    raw.Width,
		Height:   raw.Height,
	}
}
