package ingest

import (
	"encoding/json"
	"fmt"
)

// StreamInput represents the ffprobe JSON structure for streams.
type StreamInput struct {
	Streams []StreamJSON `json:"streams"`
}

// StreamJSON is the ffprobe JSON representation of a single stream.
type StreamJSON struct {
	Index       int    `json:"index"`
	CodecType   string `json:"codec_type"`
	CodecName   string `json:"codec_name"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	PixFmt      string `json:"pix_fmt"`
	RFrameRate  string `json:"r_frame_rate"`
	SampleRate  string `json:"sample_rate"`
	SampleFmt   string `json:"sample_fmt"`
	ChannelLayout string `json:"channel_layout"`
	BitRate     string `json:"bit_rate"`
	Duration    string `json:"duration"`
	Disposition Disp    `json:"disposition"`
	Language    string  `json:"language"`
	Tag         TagMap  `json:"tag"`
}

// Disp represents ffprobe disposition flags.
type Disp struct {
	Default          int `json:"default"`
	Forced           int `json:"forced"`
	Dub              int `json:"dub"`
	Original         int `json:"original"`
	Comment          int `json:"comment"`
	Lyrics           int `json:"lyrics"`
	Karaoke          int `json:"karaoke"`
	ProvidedBy       int `json:"provided_by"`
	Captions         int `json:"captions"`
	ClosedCaptions   int `json:"closed_captions"`
	VisualImpaired   int `json:"visual_impaired"`
	Descriptions     int `json:"descriptions"`
}

// TagMap represents key-value tags in a stream.
type TagMap map[string]string

// ParseStreams parses ffprobe JSON output into domain Stream objects.
func ParseStreams(ffprobeJSON string) ([]Stream, error) {
	var input StreamInput
	if err := json.Unmarshal([]byte(ffprobeJSON), &input); err != nil {
		return nil, fmt.Errorf("parse ffprobe JSON: %w", err)
	}

	var streams []Stream
	for _, s := range input.Streams {
		stream := Stream{
			Index:     s.Index,
			Kind:      streamKind(s.CodecType),
			Codec:     s.CodecName,
			Language:  Language(s.Language),
			Duration:  parseDuration(s.Duration),
		}

		// Video fields
		if stream.Kind == StreamKindVideo {
			stream.Width = s.Width
			stream.Height = s.Height
			stream.PixelFormat = s.PixFmt
			stream.FrameRate = parseRational(s.RFrameRate)
			stream.Interlaced = detectInterlaced(s)
		}

		// Audio fields
		if stream.Kind == StreamKindAudio {
			stream.SampleRate = parseInt(s.SampleRate)
			stream.SampleFormat = s.SampleFmt
			stream.ChannelLayout = Layout(s.ChannelLayout)
		}

		// Subtitle fields
		if stream.Kind == StreamKindSubtitle {
			stream.Forced = s.Disposition.Forced != 0
		}

		// Common fields
		stream.BitRate = parseBitRate(s.BitRate)

		streams = append(streams, stream)
	}

	return streams, nil
}

func parseDuration(s string) Rational {
	if s == "" || s == "N/A" {
		return Rational{}
	}
	var v float64
	fmt.Sscanf(s, "%f", &v) //nolint:errcheck // best-effort
	return Rational{Num: int(v * 1000), Denom: 1000}
}

func streamKind(codecType string) StreamKind {
	switch codecType {
	case "video":
		return StreamKindVideo
	case "audio":
		return StreamKindAudio
	case "subtitle":
		return StreamKindSubtitle
	default:
		return StreamKindData
	}
}

func parseRational(s string) Rational {
	if s == "" || s == "0/0" {
		return Rational{}
	}
	var num, den int
	if _, err := fmt.Sscanf(s, "%d/%d", &num, &den); err != nil {
		return Rational{}
	}
	return Rational{Num: num, Denom: den}
}

func parseInt(s string) int {
	var v int
	fmt.Sscanf(s, "%d", &v) //nolint:errcheck // best-effort
	return v
}

func parseBitRate(s string) int64 {
	if s == "" || s == "N/A" {
		return 0
	}
	var v float64
	fmt.Sscanf(s, "%f", &v) //nolint:errcheck // best-effort
	return int64(v)
}

func detectInterlaced(s StreamJSON) bool {
	if s.Tag["field_order"] != "" {
		return s.Tag["field_order"] != "progressive"
	}
	return false
}
