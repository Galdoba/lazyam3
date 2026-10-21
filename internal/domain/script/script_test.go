package script

import (
	"strings"
	"testing"

	"github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/Galdoba/lazyam3/internal/domain/metadata"
)

func TestBuildScript_GoldenFile(t *testing.T) {
	cfg := ScriptConfig{
		Filenames:   []string{"12254.mp4"},
		SRT:         "12254-rus_Russkii.srt",
		Destination: "/tmp/test.sh",
		Env: Environment{
			Buffer:        "/tmp/buffer",
			InProgress:    "/tmp/in_progress",
			Done:          "/tmp/done",
			Notifications: "/tmp/notifications",
			TargetRoot:    "/mnt/ROOT/EDIT",
			Priority:      6,
		},
		Product: metadata.Product{
			ID:            12061,
			Kind:          metadata.KindSeriesEpisode,
			Title:         "Любовь и смерть",
			ParentID:      ptrInt64(11393),
			SeasonNumber:  ptrInt(1),
			EpisodeNumber: ptrInt(1),
			SeriesTitle:   "Белая королева",
		},
		Streams: []ingest.Stream{
			{FileIndex: 0, Index: 0, Kind: ingest.StreamKindVideo, Codec: "h264", Width: 720, Height: 576, Interlaced: false, FrameRate: ingest.Rational{Num: 25, Denom: 1}},
			{FileIndex: 0, Index: 1, Kind: ingest.StreamKindAudio, Codec: "aac", Language: "rus", ChannelLayout: ingest.LayoutStereo, SampleRate: 48000},
			{FileIndex: 0, Index: 2, Kind: ingest.StreamKindAudio, Codec: "aac", Language: "jpn", ChannelLayout: ingest.LayoutMono, SampleRate: 44100},
		},
		CallNames: map[string]string{
			"12254.mp4":          "Arliss_s01e06_PRT261021140537_id12254.mp4",
			"12254-rus_Russkii.srt": "Arliss_s01e06_PRT261021140537_id12254-rus_Russkii.srt",
		},
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript failed: %v", err)
	}

	if script.Destination != "/tmp/test.sh" {
		t.Errorf("destination = %q, want %q", script.Destination, "/tmp/test.sh")
	}

	if !strings.HasPrefix(script.Content, "#!/bin/bash") {
		t.Errorf("content does not start with shebang, got:\n%s", script.Content)
	}

	// Check key placeholders were resolved
	// SOURCE must use call name from CallNames
	if !strings.Contains(script.Content, `SOURCE="Arliss_s01e06_PRT261021140537_id12254.mp4"`) {
		t.Error("SOURCE should be the call name from CallNames")
	}
	checkResolved(t, script.Content, "srt", "12254-rus_Russkii.srt")
	checkResolved(t, script.Content, "yadif", "") // progressive video, no yadif
	checkResolved(t, script.Content, "audio_filter", "[0:1]")
	checkResolved(t, script.Content, "audio_filter", "[0:2]")
	checkResolved(t, script.Content, "output_files", "AUDIO00_RUS20")
	checkResolved(t, script.Content, "output_files", "AUDIO01_JPN10")

	// Env paths must render as plain assignments (bash-valid)
	checkResolved(t, script.Content, "in_progress", `IN_PROGRESS="/tmp/in_progress"`)
	checkResolved(t, script.Content, "done", `DONE="/tmp/done"`)
	checkResolved(t, script.Content, "notifications", `NOTIFICATIONS="/tmp/notifications"`)
	if strings.Contains(script.Content, "${/tmp/") {
		t.Error("env paths rendered inside ${...} — invalid bash")
	}

	// Check no unresolved placeholders remain
	if unresolved := findUnresolved(script.Content); len(unresolved) > 0 {
		t.Errorf("unresolved placeholders in rendered script: %s", strings.Join(unresolved, ", "))
	}

	// Print for manual inspection
	t.Logf("Generated script (%d bytes):\n%s", len(script.Content), script.Content)
}

func TestBuildScript_MissingVideo(t *testing.T) {
	cfg := ScriptConfig{
		Filenames:   []string{"001_video.mkv"},
		Destination: "/tmp/test.sh",
		Streams: []ingest.Stream{
			{FileIndex: 0, Index: 1, Kind: ingest.StreamKindAudio, Codec: "aac"},
		},
	}

	_, err := BuildScript(cfg)
	if err == nil {
		t.Fatal("expected error for missing video stream, got nil")
	}
}

func TestBuildScript_EmptyDestination(t *testing.T) {
	cfg := ScriptConfig{
		Filenames: []string{"001_video.mkv"},
		Streams: []ingest.Stream{
			{FileIndex: 0, Index: 0, Kind: ingest.StreamKindVideo, Codec: "h264"},
		},
	}

	_, err := BuildScript(cfg)
	if err == nil {
		t.Fatal("expected error for empty destination, got nil")
	}
}

func TestBuildScript_EmptyStreams(t *testing.T) {
	cfg := ScriptConfig{
		Filenames:   []string{"001_video.mkv"},
		Destination: "/tmp/test.sh",
		Streams:     []ingest.Stream{},
	}

	_, err := BuildScript(cfg)
	if err == nil {
		t.Fatal("expected error for empty streams, got nil")
	}
}

func TestBuildScript_NoSRT(t *testing.T) {
	cfg := ScriptConfig{
		Filenames:   []string{"001_video.mkv"},
		Destination: "/tmp/test.sh",
		Env: Environment{
			Buffer:     "/tmp/buffer",
			InProgress: "/tmp/in_progress",
			Done:       "/tmp/done",
			TargetRoot: "/mnt/ROOT/EDIT",
			Priority:   6,
		},
		Streams: []ingest.Stream{
			{FileIndex: 0, Index: 0, Kind: ingest.StreamKindVideo, Codec: "h264"},
		},
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript failed: %v", err)
	}

	// SRT-related placeholders should be empty strings
	if strings.Contains(script.Content, "mv") && strings.Contains(script.Content, "001_sub") {
		t.Error("SRT move line present despite no SRT filename")
	}
}

func TestBuildScript_Interlaced(t *testing.T) {
	cfg := ScriptConfig{
		Filenames:   []string{"001_video.mkv"},
		Destination: "/tmp/test.sh",
		Env: Environment{
			Buffer:     "/tmp/buffer",
			InProgress: "/tmp/in_progress",
			Done:       "/tmp/done",
			TargetRoot: "/mnt/ROOT/EDIT",
			Priority:   6,
		},
		Streams: []ingest.Stream{
			{FileIndex: 0, Index: 0, Kind: ingest.StreamKindVideo, Codec: "h264", Interlaced: true},
		},
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript failed: %v", err)
	}

	if !strings.Contains(script.Content, "yadif,") {
		t.Error("expected yadif filter for interlaced video, not found")
	}
}

func TestBuildScript_NotInterlaced(t *testing.T) {
	cfg := ScriptConfig{
		Filenames:   []string{"001_video.mkv"},
		Destination: "/tmp/test.sh",
		Env: Environment{
			Buffer:     "/tmp/buffer",
			InProgress: "/tmp/in_progress",
			Done:       "/tmp/done",
			TargetRoot: "/mnt/ROOT/EDIT",
			Priority:   6,
		},
		Streams: []ingest.Stream{
			{FileIndex: 0, Index: 0, Kind: ingest.StreamKindVideo, Codec: "h264", Interlaced: false},
		},
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript failed: %v", err)
	}

	if strings.Contains(script.Content, "yadif,") {
		t.Error("unexpected yadif filter for non-interlaced video")
	}
}

func TestBuildScript_WithHSUB(t *testing.T) {
	cfg := ScriptConfig{
		Filenames:   []string{"12094.mp4"},
		SRT:         "12094-rus_Russkii.srt",
		HSUBFile:    "12094-rus_Russkii.ass",
		Destination: "/tmp/test.sh",
		Env: Environment{
			Buffer:     "/tmp/buffer",
			InProgress: "/tmp/in_progress",
			Done:       "/tmp/done",
			TargetRoot: "/mnt/ROOT/EDIT",
			Priority:   6,
		},
		Streams: []ingest.Stream{
			{FileIndex: 0, Index: 0, Kind: ingest.StreamKindVideo, Codec: "h264", Interlaced: false},
		},
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript failed: %v", err)
	}

	if !strings.Contains(script.Content, "subtitles=filename=/tmp/in_progress/12094-rus_Russkii.ass:original_size=1920x1080") {
		t.Errorf("expected subtitles hardsub filter with absolute path, not found in:\n%s", script.Content)
	}
	if !strings.Contains(script.Content, `HSUBFILE="12094-rus_Russkii.ass"`) {
		t.Error("expected HSUBFILE variable resolved, not found")
	}
	if unresolved := findUnresolved(script.Content); len(unresolved) > 0 {
		t.Errorf("unresolved placeholders in rendered script: %s", strings.Join(unresolved, ", "))
	}
}

func TestBuildHSUB(t *testing.T) {
	if got := buildHSUB("", ""); got != "" {
		t.Errorf("buildHSUB(\"\") = %q, want empty", got)
	}
	got := buildHSUB("subs.ass", "/tmp/in_progress")
	if !strings.HasPrefix(got, "subtitles=filename=/tmp/in_progress/subs.ass:original_size=1920x1080") {
		t.Errorf("buildHSUB() = %q, want subtitles filter with absolute path", got)
	}
}

func TestBuildScript_AudioSelection(t *testing.T) {
	cfg := ScriptConfig{
		Filenames:   []string{"001_video.mkv"},
		Destination: "/tmp/test.sh",
		Env: Environment{
			Buffer:     "/tmp/buffer",
			InProgress: "/tmp/in_progress",
			Done:       "/tmp/done",
			TargetRoot: "/mnt/ROOT/EDIT",
			Priority:   6,
		},
		StreamSelections: []StreamSelection{
			{InputIndex: 0, StreamIndex: 1, Kind: "a"},
			{InputIndex: 0, StreamIndex: 2, Kind: "a"},
		},
		Streams: []ingest.Stream{
			{FileIndex: 0, Index: 0, Kind: ingest.StreamKindVideo, Codec: "h264"},
			{FileIndex: 0, Index: 1, Kind: ingest.StreamKindAudio, Codec: "aac", Language: "rus", ChannelLayout: ingest.LayoutStereo},
			{FileIndex: 0, Index: 2, Kind: ingest.StreamKindAudio, Codec: "aac", Language: "eng", ChannelLayout: ingest.LayoutStereo},
		},
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript failed: %v", err)
	}

	if !strings.Contains(script.Content, "AUDIO00_RUS20") {
		t.Error("expected AUDIO00_RUS20 in audio outputs")
	}
	if !strings.Contains(script.Content, "AUDIO01_ENG20") {
		t.Error("expected AUDIO01_ENG20 in audio outputs")
	}
}

func TestBuildScript_MissingSelection(t *testing.T) {
	cfg := ScriptConfig{
		Filenames:   []string{"001_video.mkv"},
		Destination: "/tmp/test.sh",
		StreamSelections: []StreamSelection{
			{InputIndex: 0, StreamIndex: 99, Kind: "a"},
		},
		Streams: []ingest.Stream{
			{FileIndex: 0, Index: 0, Kind: ingest.StreamKindVideo, Codec: "h264"},
		},
	}

	_, err := BuildScript(cfg)
	if err == nil {
		t.Fatal("expected error for missing stream selection, got nil")
	}
}

func TestDeriveOutBase(t *testing.T) {
	tests := []struct {
		name    string
		product metadata.Product
		want    string
	}{
		{
			name: "full episode",
			product: metadata.Product{
				SeriesTitle:   "Белая королева",
				SeasonNumber:  ptrInt(1),
				EpisodeNumber: ptrInt(5),
				Title:         "Любовь и смерть",
			},
			want: "Belaya_koroleva_s01e05",
		},
		{
			name: "series only",
			product: metadata.Product{
				SeriesTitle: "Белая королева",
			},
			want: "Belaya_koroleva",
		},
		{
			name: "series with season no episode",
			product: metadata.Product{
				SeriesTitle:  "Белая королева",
				SeasonNumber: ptrInt(1),
			},
			want: "Belaya_koroleva_s01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveOutBase(tt.product)
			if got != tt.want {
				t.Errorf("deriveOutBase() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLayoutSuffix(t *testing.T) {
	tests := []struct {
		layout ingest.Layout
		want   string
	}{
		{ingest.LayoutStereo, "20"},
		{ingest.LayoutSurround51, "51"},
		{ingest.LayoutSurround71, "70"},
		{ingest.LayoutMono, "10"},
		{"unknown", "20"},
	}

	for _, tt := range tests {
		t.Run(string(tt.layout), func(t *testing.T) {
			got := layoutSuffix(tt.layout)
			if got != tt.want {
				t.Errorf("layoutSuffix(%q) = %q, want %q", tt.layout, got, tt.want)
			}
		})
	}
}

func TestBuildAudioFilter(t *testing.T) {
	tracks := []audioTrack{
		{StreamIndex: 1, Language: "rus", Layout: "20", Suffix: "AUDIO_00_RUS20"},
		{StreamIndex: 2, Language: "eng", Layout: "20", Suffix: "AUDIO_01_ENG20"},
	}

	got := buildAudioFilter(tracks)
	want := ";[0:1]aresample=48000,atempo=25/(25/1)[aud0];[0:2]aresample=48000,atempo=25/(25/1)[aud1]"

	if got != want {
		t.Errorf("buildAudioFilter() = %q, want %q", got, want)
	}
}

func TestBuildOutputFiles(t *testing.T) {
	videoLine := `  -map "[vidHD]" -c:v libx264 -preset medium -crf 21 -pix_fmt yuv420p -profile:v high -g 0 -map_metadata -1 -map_chapters -1 "${TARGET_DIR}/${OUTBASE}_HD.mp4"`

	t.Run("with audio", func(t *testing.T) {
		tracks := []audioTrack{
			{StreamIndex: 1, Language: "rus", Layout: "20", Suffix: "AUDIO_00_RUS20"},
		}
		got := buildOutputFiles(tracks)
		want := videoLine + ` \` + "\n" +
			`  -map "[aud0]" -c:a alac -compression_level 0 "${TARGET_DIR}/${OUTBASE}_HD_AUDIO_00_RUS20.m4a"`
		if got != want {
			t.Errorf("buildOutputFiles() = %q, want %q", got, want)
		}
	})

	t.Run("video only has no trailing continuation", func(t *testing.T) {
		got := buildOutputFiles(nil)
		if got != videoLine {
			t.Errorf("buildOutputFiles(nil) = %q, want %q", got, videoLine)
		}
		if strings.HasSuffix(strings.TrimSpace(got), `\`) {
			t.Error("video-only output must not end with a line continuation")
		}
	})
}

func TestFindUnresolved(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "no placeholders",
			input: "hello world",
			want:  nil,
		},
		{
			name:  "one placeholder",
			input: "hello |=source=| world",
			want:  []string{"source"},
		},
		{
			name:  "multiple placeholders",
			input: "|=source=| and |=outbase=| and |=missing=|",
			want:  []string{"source", "outbase", "missing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findUnresolved(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("findUnresolved() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("findUnresolved()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestReplaceAll(t *testing.T) {
	input := "hello |=name=|, your |=age=| years old"
	got := replaceAll(input, "name", "Alice")
	got = replaceAll(got, "age", "30")
	want := "hello Alice, your 30 years old"

	if got != want {
		t.Errorf("replaceAll() = %q, want %q", got, want)
	}
}

func ptrInt(i int) *int       { return &i }
func ptrInt64(i int64) *int64 { return &i }

func checkResolved(t *testing.T, content, key, value string) {
	t.Helper()
	if !strings.Contains(content, value) {
		t.Errorf("content does not contain resolved value for %q: %q", key, value)
	}
}
