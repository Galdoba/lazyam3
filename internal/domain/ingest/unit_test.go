package ingest

import (
	"testing"
)

func TestExtractID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{"pure numeric", "63603.mp4", 63603, false},
		{"mkv extension", "63603.mkv", 63603, false},
		{"underscore suffix", "63604_PRT251205.mp4", 63604, false},
		{"hyphen suffix", "63603-rus_Russkii.srt", 63603, false},
		{"underscore then digits", "63604_1080.mp4", 63604, false},
		{"no extension", "63603", 63603, false},
		{"single digit", "1.mp4", 1, false},
		{"alpha prefix", "Copy 2 of 63604.mp4", 0, true},
		{"empty", "", 0, true},
		{"alpha only", "unknown.mp4", 0, true},
		{"underscore first", "_63603.mp4", 0, true},
		{"dot first", ".63603.mp4", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ExtractID(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ExtractID(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractID(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestStreamCoordinate(t *testing.T) {
	tests := []struct {
		name     string
		stream   Stream
		expected string
	}{
		{"video stream of file 0", Stream{FileIndex: 0, Index: 0, Kind: StreamKindVideo}, "[0:v:0]"},
		{"audio stream of file 1", Stream{FileIndex: 1, Index: 2, Kind: StreamKindAudio}, "[1:a:2]"},
		{"subtitle stream of file 2", Stream{FileIndex: 2, Index: 0, Kind: StreamKindSubtitle}, "[2:s:0]"},
		{"data stream of file 0", Stream{FileIndex: 0, Index: 5, Kind: StreamKindData}, "[0:data:5]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.stream.Coordinate()
			if got != tt.expected {
				t.Errorf("Stream.Coordinate() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseStreams(t *testing.T) {
	input := `{
		"streams": [
			{
				"index": 0,
				"codec_type": "video",
				"codec_name": "h264",
				"width": 1920,
				"height": 1080,
				"pix_fmt": "yuv420p",
				"r_frame_rate": "25/1",
				"bit_rate": "5000000",
				"tag": {"field_order": "progressive"}
			},
			{
				"index": 1,
				"codec_type": "audio",
				"codec_name": "aac",
				"sample_rate": "48000",
				"sample_fmt": "fltp",
				"channel_layout": "stereo",
				"bit_rate": "128000",
				"language": "eng"
			},
			{
				"index": 2,
				"codec_type": "subtitle",
				"codec_name": "mov_text",
				"disposition": {"default": 1, "forced": 1},
				"language": "rus"
			}
		]
	}`

	streams, err := ParseStreams(input)
	if err != nil {
		t.Fatalf("ParseStreams error: %v", err)
	}

	if len(streams) != 3 {
		t.Fatalf("expected 3 streams, got %d", len(streams))
	}

	// Video stream
	v := streams[0]
	if v.Kind != StreamKindVideo {
		t.Errorf("stream[0] kind = %v, want video", v.Kind)
	}
	if v.Codec != "h264" {
		t.Errorf("stream[0] codec = %q, want h264", v.Codec)
	}
	if v.Width != 1920 || v.Height != 1080 {
		t.Errorf("stream[0] size = %dx%d, want 1920x1080", v.Width, v.Height)
	}
	if v.FrameRate.Num != 25 || v.FrameRate.Denom != 1 {
		t.Errorf("stream[0] frame rate = %d/%d, want 25/1", v.FrameRate.Num, v.FrameRate.Denom)
	}
	if v.Interlaced {
		t.Errorf("stream[0] interlaced = true, want false")
	}
	if v.BitRate != 5000000 {
		t.Errorf("stream[0] bitrate = %d, want 5000000", v.BitRate)
	}

	// Audio stream
	a := streams[1]
	if a.Kind != StreamKindAudio {
		t.Errorf("stream[1] kind = %v, want audio", a.Kind)
	}
	if a.SampleRate != 48000 {
		t.Errorf("stream[1] sample rate = %d, want 48000", a.SampleRate)
	}
	if a.Language != "eng" {
		t.Errorf("stream[1] language = %q, want eng", a.Language)
	}

	// Subtitle stream
	s := streams[2]
	if s.Kind != StreamKindSubtitle {
		t.Errorf("stream[2] kind = %v, want subtitle", s.Kind)
	}
	if !s.Forced {
		t.Errorf("stream[2] forced = false, want true")
	}
	if s.Language != "rus" {
		t.Errorf("stream[2] language = %q, want rus", s.Language)
	}
}

func TestValidateMediaUnit(t *testing.T) {
	tests := []struct {
		name     string
		streams  []Stream
		wantN    int
	}{
		{
			name:  "no issues",
			streams: []Stream{{Kind: StreamKindVideo, BitRate: 5000000}},
			wantN: 0,
		},
		{
			name:  "video missing bitrate",
			streams: []Stream{{Kind: StreamKindVideo, BitRate: 0}},
			wantN: 1,
		},
		{
			name:  "audio missing bitrate (ok)",
			streams: []Stream{{Kind: StreamKindAudio, BitRate: 0}},
			wantN: 0,
		},
		{
			name:  "mixed streams",
			streams: []Stream{
				{Kind: StreamKindVideo, BitRate: 5000000},
				{Kind: StreamKindVideo, BitRate: 0},
				{Kind: StreamKindAudio, BitRate: 0},
			},
			wantN: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := ValidateMediaUnit(tt.streams)
			if len(issues) != tt.wantN {
				t.Errorf("ValidateMediaUnit() returned %d issues, want %d", len(issues), tt.wantN)
			}
		})
	}
}
