package ffprobe

import (
	"bytes"
	"encoding/json"
	"os/exec"

	"github.com/Galdoba/lazyam3/internal/domain/ingest"
)

// Client wraps ffprobe execution.
type Client struct {
	Path string
}

// RawData represents the full ffprobe JSON output.
type RawData struct {
	Format  Format   `json:"format"`
	Streams []Stream `json:"streams"`
}

// Format describes the container format.
type Format struct {
	Filename   string            `json:"filename"`
	FormatName string            `json:"format_name"`
	Duration   string            `json:"duration"`
	BitRate    string            `json:"bit_rate"`
	Tags       map[string]string `json:"tags,omitempty"`
}

// Stream describes a single stream from ffprobe output.
type Stream struct {
	Index      int               `json:"index"`
	Codectype  string            `json:"codec_type"`
	CodecName  string            `json:"codec_name"`
	Width      int               `json:"width,omitempty"`
	Height     int               `json:"height,omitempty"`
	SampleRate string            `json:"sample_rate,omitempty"`
	BitRate    string            `json:"bit_rate,omitempty"`
	Language   string            `json:"language,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
}

// Probe runs ffprobe and returns the parsed JSON data.
func (c *Client) Probe(path string) (*RawData, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	}

	cmd := exec.Command(c.Path, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var data RawData
	if err := json.Unmarshal(stdout.Bytes(), &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// MapStreams converts ffprobe RawData to domain ingest.Streams.
func MapStreams(raw *RawData) []ingest.Stream {
	var streams []ingest.Stream
	for _, s := range raw.Streams {
		streams = append(streams, ingest.Stream{
			Index:    s.Index,
			Kind:     mapKind(s.Codectype),
			Codec:    s.CodecName,
			Language: ingest.Language(s.Language),
			Width:    s.Width,
			Height:   s.Height,
		})
	}
	return streams
}

func mapKind(codecType string) ingest.StreamKind {
	switch codecType {
	case "video":
		return ingest.StreamKindVideo
	case "audio":
		return ingest.StreamKindAudio
	case "subtitle":
		return ingest.StreamKindSubtitle
	default:
		return ingest.StreamKindData
	}
}
