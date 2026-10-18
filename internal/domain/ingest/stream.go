package ingest

import (
	"fmt"
)

// StreamKind identifies the type of a media stream.
type StreamKind string

const (
	StreamKindVideo    StreamKind = "video"
	StreamKindAudio    StreamKind = "audio"
	StreamKindSubtitle StreamKind = "subtitle"
	StreamKindData     StreamKind = "data"
)

// Rational represents a rational number (numerator / denominator).
type Rational struct {
	Num   int
	Denom int
}

// Layout describes the channel layout of an audio stream.
type Layout string

const (
	LayoutMono       Layout = "mono"
	LayoutStereo     Layout = "stereo"
	LayoutSurround51 Layout = "5.1"
	LayoutSurround71 Layout = "7.1"
)

// Language represents an IETF language tag.
type Language string

// Stream describes a single media stream (video, audio, subtitles, data).
type Stream struct {
	// Source
	FileIndex int
	Index     int

	// Common
	Kind  StreamKind
	Codec string

	// Video
	Width       int
	Height      int
	FrameRate   Rational
	PixelFormat string
	Interlaced  bool

	// Audio
	ChannelLayout Layout
	Language      Language
	SampleRate    int
	SampleFormat  string

	// Subtitles
	Forced bool

	// Common (all streams)
	BitRate  int64
	Duration Rational
}

// Coordinate returns the ffmpeg-style stream coordinate,
// e.g. "[0:v:0]" or "[1:s:0]".
func (s Stream) Coordinate() string {
	kind := abbreviateKind(s.Kind)
	return fmt.Sprintf("[%d:%s:%d]", s.FileIndex, kind, s.Index)
}

func abbreviateKind(kind StreamKind) string {
	switch kind {
	case StreamKindVideo:
		return "v"
	case StreamKindAudio:
		return "a"
	case StreamKindSubtitle:
		return "s"
	default:
		return string(kind)
	}
}
