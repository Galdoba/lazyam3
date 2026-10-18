package ffprobe

import (
	"github.com/Galdoba/lazyam3/internal/domain/ingest"
)

// ProbeClient wraps ffprobe.Client and implements ingest.TechnicalProbe.
type ProbeClient struct {
	Client *Client
}

// Probe implements ingest.TechnicalProbe.
func (p *ProbeClient) Probe(path string) ([]ingest.Stream, error) {
	raw, err := p.Client.Probe(path)
	if err != nil {
		return nil, err
	}
	return MapStreams(raw), nil
}
