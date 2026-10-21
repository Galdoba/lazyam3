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
			got := tt.stream.coordinate()
			if got != tt.expected {
				t.Errorf("Stream.coordinate() = %q, want %q", got, tt.expected)
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
				"tags": {"language": "eng"}
			},
			{
				"index": 2,
				"codec_type": "subtitle",
				"codec_name": "mov_text",
				"disposition": {"default": 1, "forced": 1},
				"tags": {"language": "rus"}
			}
		]
	}`

	streams, err := parseStreams([]byte(input))
	if err != nil {
		t.Fatalf("parseStreams error: %v", err)
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
		name    string
		streams []Stream
		wantN   int
	}{
		{
			name:    "no issues",
			streams: []Stream{{Kind: StreamKindVideo, BitRate: 5000000}},
			wantN:   0,
		},
		{
			name:    "video low bitrate",
			streams: []Stream{{Kind: StreamKindVideo, BitRate: 50000}},
			wantN:   1,
		},
		{
			name:    "audio missing bitrate (ok)",
			streams: []Stream{{Kind: StreamKindAudio, BitRate: 0}},
			wantN:   0,
		},
		{
			name: "mixed streams",
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

func TestNewMediaUnit(t *testing.T) {
	videoProbe := `{"streams":[{"index":0,"codec_type":"video","codec_name":"h264","width":1920,"height":1080,"pix_fmt":"yuv420p","r_frame_rate":"25/1","bit_rate":"5000000","tag":{"field_order":"progressive"}}]}`
	audioProbe := `{"streams":[{"index":0,"codec_type":"audio","codec_name":"aac","sample_rate":"48000","sample_fmt":"fltp","channel_layout":"stereo","bit_rate":"128000","tags":{"language":"eng"}}]}`
	subProbe := `{"streams":[{"index":0,"codec_type":"subtitle","codec_name":"mov_text","disposition":{"forced":1},"tags":{"language":"rus"}}]}`
	dataProbe := `{"streams":[{"index":0,"codec_type":"data","codec_name":"none"}]}`
	badJSON := `not json`

	tests := []struct {
		name        string
		data        map[string][]byte
		wantErr     bool
		wantID      int64
		wantFiles   []string
		wantStreams int
	}{
		{
			name:        "single video file",
			data:        map[string][]byte{"63603.mp4": []byte(videoProbe)},
			wantErr:     false,
			wantID:      63603,
			wantFiles:   []string{"63603.mp4"},
			wantStreams: 1,
		},
		{
			name: "multi-file unit",
			data: map[string][]byte{
				"63603.mp4":       []byte(videoProbe),
				"63603_audio.mkv": []byte(audioProbe),
				"63603_sub.srt":   []byte(subProbe),
			},
			wantErr:     false,
			wantID:      63603,
			wantFiles:   []string{"63603.mp4", "63603_audio.mkv", "63603_sub.srt"},
			wantStreams: 3,
		},
		{
			name: "data-only file filtered",
			data: map[string][]byte{
				"63603.mp4":      []byte(videoProbe),
				"63603_data.dat": []byte(dataProbe),
			},
			wantErr:     false,
			wantID:      63603,
			wantFiles:   []string{"63603.mp4"},
			wantStreams: 1,
		},
		{
			name: "video file first despite lexicographic srt order",
			data: map[string][]byte{
				"63603-rus.srt": []byte(subProbe),
				"63603.mp4":     []byte(videoProbe),
			},
			wantErr:     false,
			wantID:      63603,
			wantFiles:   []string{"63603.mp4", "63603-rus.srt"},
			wantStreams: 2,
		},
		{
			name:    "invalid json",
			data:    map[string][]byte{"63603.mp4": []byte(badJSON)},
			wantErr: true,
		},
		{
			name: "inconsistent IDs",
			data: map[string][]byte{
				"63603.mp4": []byte(videoProbe),
				"99999.mp4": []byte(videoProbe),
			},
			wantErr: true,
		},
		{
			name: "only data files",
			data: map[string][]byte{
				"data1.dat": []byte(dataProbe),
				"data2.dat": []byte(dataProbe),
			},
			wantErr:     false,
			wantID:      0,
			wantFiles:   []string{},
			wantStreams: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mu, err := NewMediaUnit(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewMediaUnit() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("NewMediaUnit() unexpected error: %v", err)
				return
			}
			if mu.ID != tt.wantID {
				t.Errorf("ID = %d, want %d", mu.ID, tt.wantID)
			}
			if len(mu.Filenames) != len(tt.wantFiles) {
				t.Errorf("Filenames: got %d files, want %d", len(mu.Filenames), len(tt.wantFiles))
			}
			for i, f := range tt.wantFiles {
				if i < len(mu.Filenames) && mu.Filenames[i] != f {
					t.Errorf("Filenames[%d] = %q, want %q", i, mu.Filenames[i], f)
				}
			}
			if len(mu.Streams) != tt.wantStreams {
				t.Errorf("Streams: got %d, want %d", len(mu.Streams), tt.wantStreams)
			}
		})
	}
}

func TestNewMediaUnit_Empty(t *testing.T) {
	mu, err := NewMediaUnit(map[string][]byte{})
	if err != nil {
		t.Fatalf("NewMediaUnit() unexpected error: %v", err)
	}
	if mu.ID != 0 {
		t.Errorf("ID = %d, want 0", mu.ID)
	}
	if len(mu.Filenames) != 0 {
		t.Errorf("Filenames: got %d files, want 0", len(mu.Filenames))
	}
	if len(mu.Streams) != 0 {
		t.Errorf("Streams: got %d, want 0", len(mu.Streams))
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		wantN int
		wantD int
	}{
		{"10.5", 10500, 1000},
		{"0", 0, 1000},
		{"", 0, 0},
		{"N/A", 0, 0},
		{"3.14159", 3141, 1000},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseDuration(tt.input)
			if got.Num != tt.wantN || got.Denom != tt.wantD {
				t.Errorf("parseDuration(%q) = %d/%d, want %d/%d", tt.input, got.Num, got.Denom, tt.wantN, tt.wantD)
			}
		})
	}
}

func TestParseRational(t *testing.T) {
	tests := []struct {
		input string
		wantN int
		wantD int
	}{
		{"25/1", 25, 1},
		{"30000/1001", 30000, 1001},
		{"0/0", 0, 0},
		{"", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseRational(tt.input)
			if got.Num != tt.wantN || got.Denom != tt.wantD {
				t.Errorf("parseRational(%q) = %d/%d, want %d/%d", tt.input, got.Num, got.Denom, tt.wantN, tt.wantD)
			}
		})
	}
}

func TestParseBitRate(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"5000000", 5000000},
		{"128000", 128000},
		{"", 0},
		{"N/A", 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseBitRate(tt.input)
			if got != tt.want {
				t.Errorf("parseBitRate(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestStreamKind(t *testing.T) {
	tests := []struct {
		input string
		want  StreamKind
	}{
		{"video", StreamKindVideo},
		{"audio", StreamKindAudio},
		{"subtitle", StreamKindSubtitle},
		{"data", StreamKindData},
		{"unknown", StreamKindData},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := streamKind(tt.input)
			if got != tt.want {
				t.Errorf("streamKind(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDetectInterlaced(t *testing.T) {
	tests := []struct {
		name string
		tag  map[string]string
		want bool
	}{
		{"progressive", map[string]string{"field_order": "progressive"}, false},
		{"top_field_first", map[string]string{"field_order": "tt"}, true},
		{"bottom_field_first", map[string]string{"field_order": "bb"}, true},
		{"no field_order", map[string]string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectInterlaced(StreamJSON{Tags: StreamTags{FieldOrder: tt.tag["field_order"]}})
			if got != tt.want {
				t.Errorf("detectInterlaced(%v) = %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestValidateMediaUnit_Comprehensive(t *testing.T) {
	video := Stream{Kind: StreamKindVideo, Width: 1920, Height: 1080, BitRate: 5000000, Duration: Rational{Num: 10, Denom: 1}}
	audio := Stream{Kind: StreamKindAudio, BitRate: 128000, Duration: Rational{Num: 10, Denom: 1}}
	sub := Stream{Kind: StreamKindSubtitle}

	tests := []struct {
		name     string
		streams  []Stream
		wantN    int
		wantMsgs []string
	}{
		{
			name:     "all good",
			streams:  []Stream{video, audio},
			wantN:    0,
			wantMsgs: []string{},
		},
		{
			name: "low res video",
			streams: []Stream{
				{Kind: StreamKindVideo, Width: 320, Height: 240, BitRate: 500000, Duration: Rational{Num: 10, Denom: 1}},
			},
			wantN:    1,
			wantMsgs: []string{"resolution"},
		},
		{
			name: "low bitrate video",
			streams: []Stream{
				{Kind: StreamKindVideo, Width: 1920, Height: 1080, BitRate: 50000, Duration: Rational{Num: 10, Denom: 1}},
			},
			wantN:    1,
			wantMsgs: []string{"bitrate"},
		},
		{
			name: "low bitrate audio",
			streams: []Stream{
				{Kind: StreamKindAudio, BitRate: 16000, Duration: Rational{Num: 10, Denom: 1}},
			},
			wantN:    1,
			wantMsgs: []string{"bitrate"},
		},
		{
			name: "negative duration video",
			streams: []Stream{
				{Kind: StreamKindVideo, Width: 1920, Height: 1080, BitRate: 5000000, Duration: Rational{Num: -1, Denom: 1}},
			},
			wantN:    1,
			wantMsgs: []string{"duration"},
		},
		{
			name: "negative duration audio",
			streams: []Stream{
				{Kind: StreamKindAudio, BitRate: 128000, Duration: Rational{Num: -5, Denom: 1}},
			},
			wantN:    1,
			wantMsgs: []string{"duration"},
		},
		{
			name:     "zero duration ok for subtitles",
			streams:  []Stream{sub},
			wantN:    0,
			wantMsgs: []string{},
		},
		{
			name: "duplicate streams",
			streams: []Stream{
				{Kind: StreamKindVideo, FileIndex: 0, Index: 0, BitRate: 5000000, Duration: Rational{Num: 10, Denom: 1}},
				{Kind: StreamKindVideo, FileIndex: 0, Index: 0, BitRate: 5000000, Duration: Rational{Num: 10, Denom: 1}},
			},
			wantN:    1,
			wantMsgs: []string{"coordinate"},
		},
		{
			name: "duration inconsistency",
			streams: []Stream{
				{Kind: StreamKindVideo, BitRate: 5000000, Duration: Rational{Num: 1000, Denom: 1}},
				{Kind: StreamKindAudio, BitRate: 128000, Duration: Rational{Num: 100, Denom: 1}},
			},
			wantN:    1,
			wantMsgs: []string{"duration_consistency"},
		},
		{
			name: "multiple issues",
			streams: []Stream{
				{Kind: StreamKindVideo, Width: 320, Height: 240, BitRate: 50000, Duration: Rational{Num: -1, Denom: 1}},
			},
			wantN:    3,
			wantMsgs: []string{"resolution", "bitrate", "duration"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := ValidateMediaUnit(tt.streams)
			if len(issues) != tt.wantN {
				t.Errorf("got %d issues, want %d", len(issues), tt.wantN)
				for _, iss := range issues {
					t.Logf("  issue: %s.%s: %s", iss.Field, iss.Problem, iss.Problem)
				}
			}
			for _, expected := range tt.wantMsgs {
				found := false
				for _, iss := range issues {
					if iss.Field == expected || (expected == "duration" && iss.Field == "duration") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected issue field %q not found", expected)
				}
			}
		})
	}
}
