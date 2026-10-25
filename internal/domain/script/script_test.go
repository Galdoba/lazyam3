package script

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Galdoba/lazyam3/internal/domain/callname"
	"github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/Galdoba/lazyam3/internal/domain/metadata"
)

// ---------------------------------------------------------------------------
// validateConfig
// ---------------------------------------------------------------------------

func TestValidateConfig(t *testing.T) {
	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	audioStream := ingest.Stream{Kind: ingest.StreamKindAudio, Index: 1, FileIndex: 0}

	tests := []struct {
		name    string
		cfg     ScriptConfig
		wantErr bool
	}{
		{
			name: "empty destination",
			cfg: ScriptConfig{
				Streams: []ingest.Stream{videoStream},
			},
			wantErr: true,
		},
		{
			name: "empty streams",
			cfg: ScriptConfig{
				Destination: "/tmp/test.sh",
			},
			wantErr: true,
		},
		{
			name: "no video stream",
			cfg: ScriptConfig{
				Destination: "/tmp/test.sh",
				Streams:     []ingest.Stream{audioStream},
			},
			wantErr: true,
		},
		{
			name: "valid minimal config",
			cfg: ScriptConfig{
				Destination: "/tmp/test.sh",
				Streams:     []ingest.Stream{videoStream},
			},
			wantErr: false,
		},
		{
			name: "selection references non-existent stream",
			cfg: ScriptConfig{
				Destination:      "/tmp/test.sh",
				Streams:          []ingest.Stream{videoStream},
				StreamSelections: []StreamSelection{{InputIndex: 0, StreamIndex: 99, Kind: "v"}},
			},
			wantErr: true,
		},
		{
			name: "valid with selections",
			cfg: ScriptConfig{
				Destination:      "/tmp/test.sh",
				Streams:          []ingest.Stream{videoStream},
				StreamSelections: []StreamSelection{{InputIndex: 0, StreamIndex: 0, Kind: "v"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// findUnresolved
// ---------------------------------------------------------------------------

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
			name:  "single placeholder",
			input: "hello |=name=| world",
			want:  []string{"name"},
		},
		{
			name:  "multiple placeholders",
			input: "a |=foo=| b |=bar=| c |=baz=|",
			want:  []string{"foo", "bar", "baz"},
		},
		{
			name:  "no matches",
			input: "no pipe equals here",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findUnresolved(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("findUnresolved() got %d results, want %d", len(got), len(tt.want))
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

// ---------------------------------------------------------------------------
// replaceAll
// ---------------------------------------------------------------------------

func TestReplaceAll(t *testing.T) {
	got := replaceAll("hello |=name=| and |=name=|", "name", "world")
	want := "hello world and world"
	if got != want {
		t.Errorf("replaceAll() = %q, want %q", got, want)
	}

	// No match — original unchanged
	got = replaceAll("hello world", "name", "world")
	if got != "hello world" {
		t.Errorf("replaceAll() with no match = %q, want unchanged", got)
	}
}

// ---------------------------------------------------------------------------
// buildSRTMove
// ---------------------------------------------------------------------------

func TestBuildSRTMove(t *testing.T) {
	got := buildSRTMove("")
	if got != "" {
		t.Errorf("buildSRTMove(\"\") = %q, want empty", got)
	}

	got = buildSRTMove("subs.srt")
	want := `mv "${BUFFER}/${SRT}" "${IN_PROGRESS}/${SRT}"`
	if got != want {
		t.Errorf("buildSRTMove() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// buildSRTSection
// ---------------------------------------------------------------------------

func TestBuildSRTSection(t *testing.T) {
	got := buildSRTSection("", nil)
	if got != "" {
		t.Errorf("buildSRTSection(\"\", nil) = %q, want empty", got)
	}

	got = buildSRTSection("subs.srt", nil)
	if got != "" {
		t.Errorf("buildSRTSection with empty predictions = %q, want empty", got)
	}

	preds := []callname.PredictedOutput{
		{MediaType: "video", Path: "/mnt/ROOT/EDIT/Show_s01e06/Show_HD.mp4"},
		{MediaType: "subtitle", Path: "/mnt/ROOT/EDIT/Show_s01e06/Show_HD.srt"},
	}
	got = buildSRTSection("subs.srt", preds)
	want := `cp "${IN_PROGRESS}/${SRT}" "/mnt/ROOT/EDIT/Show_s01e06/Show_HD.srt"`
	if got != want {
		t.Errorf("buildSRTSection() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// buildSRTCleanup
// ---------------------------------------------------------------------------

func TestBuildSRTCleanup(t *testing.T) {
	got := buildSRTCleanup("")
	if got != "" {
		t.Errorf("buildSRTCleanup(\"\") = %q, want empty", got)
	}

	got = buildSRTCleanup("subs.srt")
	want := `mv "${IN_PROGRESS}/${SRT}" "${DONE}/${SRT}"`
	if got != want {
		t.Errorf("buildSRTCleanup() = %q, want %q", got, want)
	}
}

// defaultHarsub mirrors the defaults in infra/config.defaultCFG().
var defaultHarsub = HarsubConfig{
	OriginalSize:    "1920x1080",
	FontName:        "Segoe UI",
	FontSize:        19.4,
	PrimaryColour:   "&H00EBEBEB",
	SecondaryColour: "&H000000FF",
	OutlineColour:   "&H00000000",
	BackColour:      "&H80000000",
	ScaleX:          1,
	ScaleY:          1,
	BorderStyle:     1,
	Outline:         0.5,
	Alignment:       2,
	MarginL:         5,
	MarginR:         5,
	MarginV:         20,
	Encoding:        1,
}

// ---------------------------------------------------------------------------
// buildHSUB
// ---------------------------------------------------------------------------

func TestBuildHSUB(t *testing.T) {
	got := buildHSUB("", "/tmp/in_progress", defaultHarsub)
	if got != "" {
		t.Errorf("buildHSUB(\"\", ...) = %q, want empty", got)
	}

	got = buildHSUB("hsub.srt", "/tmp/in_progress", defaultHarsub)
	if !strings.Contains(got, "subtitles=") {
		t.Errorf("buildHSUB() does not contain 'subtitles=': %q", got)
	}
	if !strings.Contains(got, "/tmp/in_progress/hsub.srt") {
		t.Errorf("buildHSUB() does not contain expected path: %q", got)
	}
}

func TestHSUBConfigValueChangesOutput(t *testing.T) {
	base := HarsubConfig{
		OriginalSize: "1920x1080",
		FontName:     "Arial",
		FontSize:     14,
		MarginV:      10,
		BorderStyle:  1,
		Outline:      1,
		Alignment:    2,
		MarginL:      5,
		MarginR:      5,
		Encoding:     1,
	}

	gotDefault := buildHSUB("sub.srt", "/tmp/ip", base)

	changed := base
	changed.MarginV = 99
	gotChanged := buildHSUB("sub.srt", "/tmp/ip", changed)

	if gotDefault == gotChanged {
		t.Error("buildHSUB with different HarsubConfig produced identical output")
	}
	if !strings.Contains(gotDefault, "MarginV=10") {
		t.Error("default output should contain 'MarginV=10'")
	}
	if !strings.Contains(gotChanged, "MarginV=99") {
		t.Error("changed output should contain 'MarginV=99'")
	}

	// Changing a string field also changes output
	changed2 := base
	changed2.FontName = "Verdana"
	gotFont := buildHSUB("sub.srt", "/tmp/ip", changed2)
	if gotDefault == gotFont {
		t.Error("buildHSUB with different FontName produced identical output")
	}
	if !strings.Contains(gotFont, "FontName=Verdana") {
		t.Error("font-changed output should contain 'FontName=Verdana'")
	}
}

// ---------------------------------------------------------------------------
// buildHSUBMove
// ---------------------------------------------------------------------------

func TestBuildHSUBMove(t *testing.T) {
	got := buildHSUBMove("")
	if got != "" {
		t.Errorf("buildHSUBMove(\"\") = %q, want empty", got)
	}

	got = buildHSUBMove("hsub.srt")
	want := `mv "${BUFFER}/${HSUBFILE}" "${IN_PROGRESS}/${HSUBFILE}"`
	if got != want {
		t.Errorf("buildHSUBMove() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// buildHSUBCleanup
// ---------------------------------------------------------------------------

func TestBuildHSUBCleanup(t *testing.T) {
	got := buildHSUBCleanup("")
	if got != "" {
		t.Errorf("buildHSUBCleanup(\"\") = %q, want empty", got)
	}

	got = buildHSUBCleanup("hsub.srt")
	want := `mv "${IN_PROGRESS}/${HSUBFILE}" "${DONE}/${HSUBFILE}"`
	if got != want {
		t.Errorf("buildHSUBCleanup() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// buildMKDirTarget
// ---------------------------------------------------------------------------

func TestBuildMKDirTarget(t *testing.T) {
	got := buildMKDirTarget(nil)
	if got != "" {
		t.Errorf("buildMKDirTarget(nil) = %q, want empty", got)
	}

	got = buildMKDirTarget([]callname.PredictedOutput{})
	if got != "" {
		t.Errorf("buildMKDirTarget([]) = %q, want empty", got)
	}

	// Fallback: non-video first prediction
	preds := []callname.PredictedOutput{{MediaType: "audio", Path: "/mnt/ROOT/EDIT/Show/H audio.m4a"}}
	got = buildMKDirTarget(preds)
	want := `mkdir -p "/mnt/ROOT/EDIT/Show"`
	if got != want {
		t.Errorf("buildMKDirTarget fallback = %q, want %q", got, want)
	}

	// Video prediction takes priority
	preds = []callname.PredictedOutput{
		{MediaType: "audio", Path: "/mnt/ROOT/EDIT/A/H_audio.m4a"},
		{MediaType: "video", Path: "/mnt/ROOT/EDIT/Show/H_video.mp4"},
	}
	got = buildMKDirTarget(preds)
	want = `mkdir -p "/mnt/ROOT/EDIT/Show"`
	if got != want {
		t.Errorf("buildMKDirTarget video priority = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// buildReadyFile
// ---------------------------------------------------------------------------

func TestBuildReadyFile(t *testing.T) {
	got := buildReadyFile(nil)
	if got != "" {
		t.Errorf("buildReadyFile(nil) = %q, want empty", got)
	}

	got = buildReadyFile([]callname.PredictedOutput{})
	if got != "" {
		t.Errorf("buildReadyFile([]) = %q, want empty", got)
	}

	preds := []callname.PredictedOutput{
		{MediaType: "video", Path: "/mnt/ROOT/EDIT/Show/H_HD.mp4"},
	}
	got = buildReadyFile(preds)
	want := `touch "/mnt/ROOT/EDIT/Show/H_HD.ready"`
	if got != want {
		t.Errorf("buildReadyFile() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// buildVideoFilter / getInterlaced
// ---------------------------------------------------------------------------

func TestBuildVideoFilter(t *testing.T) {
	got := buildVideoFilter(true)
	if got != "yadif," {
		t.Errorf("buildVideoFilter(true) = %q, want 'yadif,'", got)
	}

	got = buildVideoFilter(false)
	if got != "" {
		t.Errorf("buildVideoFilter(false) = %q, want empty", got)
	}
}

func TestGetInterlaced(t *testing.T) {
	streams := []ingest.Stream{
		{Kind: ingest.StreamKindVideo, Index: 0, Interlaced: false},
	}
	if getInterlaced(streams) {
		t.Error("getInterlaced(non-interlaced) = true, want false")
	}

	streams = []ingest.Stream{
		{Kind: ingest.StreamKindVideo, Index: 0, Interlaced: false},
		{Kind: ingest.StreamKindVideo, Index: 1, Interlaced: true},
	}
	if !getInterlaced(streams) {
		t.Error("getInterlaced(has-interlaced) = false, want true")
	}

	// Non-video interlaced should not count
	streams = []ingest.Stream{
		{Kind: ingest.StreamKindSubtitle, Index: 0, Interlaced: true},
	}
	if getInterlaced(streams) {
		t.Error("getInterlaced(subtitle interlaced) = true, want false")
	}
}

// ---------------------------------------------------------------------------
// buildAudioFilter
// ---------------------------------------------------------------------------

func TestBuildAudioFilter(t *testing.T) {
	got := buildAudioFilter(nil)
	if got != "" {
		t.Errorf("buildAudioFilter(nil) = %q, want empty", got)
	}

	tracks := []audioTrack{{StreamIndex: 1, Language: "rus", Layout: "20"}}
	got = buildAudioFilter(tracks)
	want := `;[0:1]aresample=48000,atempo=25/(25/1)[aud0]`
	if got != want {
		t.Errorf("buildAudioFilter(1 track) = %q, want %q", got, want)
	}

	tracks = []audioTrack{
		{StreamIndex: 1, Language: "rus", Layout: "20"},
		{StreamIndex: 2, Language: "eng", Layout: "51"},
	}
	got = buildAudioFilter(tracks)
	want = `;[0:1]aresample=48000,atempo=25/(25/1)[aud0];[0:2]aresample=48000,atempo=25/(25/1)[aud1]`
	if got != want {
		t.Errorf("buildAudioFilter(2 tracks) = %q, want %q", got, want)
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
		{ingest.Layout("unknown"), "20"},
	}
	for _, tt := range tests {
		got := layoutSuffix(tt.layout)
		if got != tt.want {
			t.Errorf("layoutSuffix(%q) = %q, want %q", tt.layout, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// buildOutputFilesPredictions
// ---------------------------------------------------------------------------

func TestBuildOutputFilesPredictions(t *testing.T) {
	got := buildOutputFilesPredictions(nil)
	if got != "" {
		t.Errorf("buildOutputFilesPredictions(nil) = %q, want empty", got)
	}

	preds := []callname.PredictedOutput{
		{MediaType: "video", Path: "/mnt/ROOT/EDIT/Show/H_HD.mp4"},
		{MediaType: "audio", Path: "/mnt/ROOT/EDIT/Show/H_HD_AUDIO00_RUS20.m4a", AudioIndex: 0},
	}
	got = buildOutputFilesPredictions(preds)

	if !strings.Contains(got, `-map "[vidHD]"`) {
		t.Error("buildOutputFilesPredictions() missing video map")
	}
	if !strings.Contains(got, `-map "[aud0]"`) {
		t.Error("buildOutputFilesPredictions() missing audio map [aud0]")
	}
	if !strings.Contains(got, "/mnt/ROOT/EDIT/Show/H_HD.mp4") {
		t.Error("buildOutputFilesPredictions() missing video path")
	}
	if !strings.Contains(got, "/mnt/ROOT/EDIT/Show/H_HD_AUDIO00_RUS20.m4a") {
		t.Error("buildOutputFilesPredictions() missing audio path")
	}
}

// ---------------------------------------------------------------------------
// selectAudioTracksFromPredictions
// ---------------------------------------------------------------------------

func TestSelectAudioTracksFromPredictions(t *testing.T) {
	audioStream := ingest.Stream{Kind: ingest.StreamKindAudio, Index: 1, FileIndex: 0, Language: ingest.Language("rus"), ChannelLayout: ingest.LayoutStereo}
	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}

	cfg := ScriptConfig{
		Streams: []ingest.Stream{videoStream, audioStream},
		Predictions: []callname.PredictedOutput{
			{MediaType: "video", Path: "/mnt/ROOT/EDIT/H_HD.mp4"},
			{MediaType: "audio", Path: "/mnt/ROOT/EDIT/H_AUDIO.m4a", Language: "rus", Layout: "20", AudioIndex: 0},
		},
	}
	tracks := selectAudioTracksFromPredictions(cfg)
	if len(tracks) != 1 {
		t.Fatalf("selectAudioTracksFromPredictions() got %d tracks, want 1", len(tracks))
	}
	if tracks[0].Language != "rus" {
		t.Errorf("track.Language = %q, want 'rus'", tracks[0].Language)
	}
	if tracks[0].Layout != "20" {
		t.Errorf("track.Layout = %q, want '20'", tracks[0].Layout)
	}

	// Default language when prediction has no language
	cfg.Predictions = []callname.PredictedOutput{
		{MediaType: "video", Path: "/mnt/ROOT/EDIT/H_HD.mp4"},
		{MediaType: "audio", Path: "/mnt/ROOT/EDIT/H_AUDIO.m4a", AudioIndex: 0},
	}
	tracks = selectAudioTracksFromPredictions(cfg)
	if len(tracks) != 1 {
		t.Fatalf("selectAudioTracksFromPredictions() got %d tracks, want 1", len(tracks))
	}
	if tracks[0].Language != "und" {
		t.Errorf("default track.Language = %q, want 'und'", tracks[0].Language)
	}

	// Default layout when prediction has no layout
	cfg.Predictions = []callname.PredictedOutput{
		{MediaType: "video", Path: "/mnt/ROOT/EDIT/H_HD.mp4"},
		{MediaType: "audio", Path: "/mnt/ROOT/EDIT/H_AUDIO.m4a", Language: "rus", AudioIndex: 0},
	}
	tracks = selectAudioTracksFromPredictions(cfg)
	if tracks[0].Layout != "20" {
		t.Errorf("default track.Layout = %q, want '20'", tracks[0].Layout)
	}
}

// ---------------------------------------------------------------------------
// abbrevKind
// ---------------------------------------------------------------------------

func TestAbbrevKind(t *testing.T) {
	tests := []struct {
		kind ingest.StreamKind
		want string
	}{
		{ingest.StreamKindVideo, "v"},
		{ingest.StreamKindAudio, "a"},
		{ingest.StreamKindSubtitle, "s"},
		{ingest.StreamKindData, "data"},
		{ingest.StreamKind("unknown"), "unknown"},
	}
	for _, tt := range tests {
		got := abbrevKind(tt.kind)
		if got != tt.want {
			t.Errorf("abbrevKind(%q) = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// derivePredictionsFromStreams
// ---------------------------------------------------------------------------

func TestDerivePredictionsFromStreams(t *testing.T) {
	prod := metadata.Product{
		ID:            12254,
		SeriesTitle:   "Арлисс",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(1),
		EpisodeNumber: ptrInt(6),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	audioStream := ingest.Stream{Kind: ingest.StreamKindAudio, Index: 1, FileIndex: 0, Language: ingest.Language("rus")}

	preds := derivePredictionsFromStreams(prod, []ingest.Stream{videoStream, audioStream}, "/mnt/ROOT/EDIT", "261023110725")

	if len(preds) == 0 {
		t.Fatal("derivePredictionsFromStreams() returned no predictions")
	}

	// First prediction should be video
	if preds[0].MediaType != "video" {
		t.Errorf("first prediction MediaType = %q, want 'video'", preds[0].MediaType)
	}

	// Check that product ID is embedded in output path (via _id12254)
	if !strings.Contains(preds[0].Path, "_id12254") {
		t.Errorf("prediction path does not contain product ID: %q", preds[0].Path)
	}

	// Check season directory prefix
	if !strings.Contains(preds[0].Path, "_s01") {
		t.Errorf("prediction path does not contain season tag: %q", preds[0].Path)
	}
}

func TestDerivePredictionsFromStreams_Empty(t *testing.T) {
	prod := metadata.Product{ID: 1, SeriesTitle: "Test"}
	preds := derivePredictionsFromStreams(prod, []ingest.Stream{}, "/mnt/ROOT", "00")
	// Even with empty streams, PredictOutputs creates at least a video prediction
	// since the product metadata drives the output name generation.
	if len(preds) < 1 {
		t.Errorf("derivePredictionsFromStreams(empty) got %d predictions, want at least 1 (video)", len(preds))
	}
}

// ---------------------------------------------------------------------------
// BuildScript — full integration within the domain
// ---------------------------------------------------------------------------

func TestBuildScript(t *testing.T) {
	prod := metadata.Product{
		ID:            12254,
		SeriesTitle:   "Арлисс",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(1),
		EpisodeNumber: ptrInt(6),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	audioStream := ingest.Stream{Kind: ingest.StreamKindAudio, Index: 1, FileIndex: 0, Language: ingest.Language("rus"), ChannelLayout: ingest.LayoutStereo}

	preds := callname.PredictOutputs(prod, ingest.MediaUnit{
		ID:           12254,
		PRTTimestamp: "261023110725",
	}, "/mnt/ROOT/EDIT", "", 12254)

	cfg := ScriptConfig{
		Filenames:    []string{"12254.mp4"},
		Streams:      []ingest.Stream{videoStream, audioStream},
		Product:      prod,
		Predictions:  preds,
		CallNames:    map[string]string{"12254.mp4": "Arliss_s01e06_PRT261023110725_id12254.mp4"},
		NameTag:      "_s01e06",
		SeasonDir:    "_s01",
		PRTTimestamp: "261023110725",
		Env: Environment{
			Buffer:        "/tmp/lazyam/buffer",
			InProgress:    "/tmp/lazyam/in_progress",
			Done:          "/tmp/lazyam/done",
			Notifications: "/tmp/lazyam/notifications",
			Archive:       "/tmp/lazyam/done/bash",
			TargetRoot:    "/mnt/ROOT/EDIT",
			Priority:      6,
		},
		Destination: "/tmp/lazyam/scripts/test.sh",
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript() error = %v", err)
	}

	if script.Destination != cfg.Destination {
		t.Errorf("script.Destination = %q, want %q", script.Destination, cfg.Destination)
	}

	if script.Content == "" {
		t.Fatal("BuildScript() returned empty content")
	}

	// Shebang present
	if !strings.HasPrefix(script.Content, "#!/bin/bash") {
		t.Error("BuildScript() content does not start with '#!/bin/bash'")
	}

	// No unresolved placeholders
	unresolved := findUnresolved(script.Content)
	if len(unresolved) > 0 {
		t.Errorf("BuildScript() has unresolved placeholders: %v", unresolved)
	}

	// Contains ffmpeg command
	if !strings.Contains(script.Content, "ffmpeg") {
		t.Error("BuildScript() content does not contain 'ffmpeg'")
	}
}

func TestBuildScript_WithSRT(t *testing.T) {
	prod := metadata.Product{
		ID:            12254,
		SeriesTitle:   "Арлисс",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(1),
		EpisodeNumber: ptrInt(6),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	srtStream := ingest.Stream{Kind: ingest.StreamKindSubtitle, Index: 2, FileIndex: 0}

	preds := callname.PredictOutputs(prod, ingest.MediaUnit{
		ID:           12254,
		PRTTimestamp: "261023110725",
	}, "/mnt/ROOT/EDIT", "", 12254)

	cfg := ScriptConfig{
		Filenames:    []string{"12254.mp4", "12254-rus.srt"},
		Streams:      []ingest.Stream{videoStream, srtStream},
		SRT:          "Arliss_s01e06_PRT261023110725_id12254_rus_Russkii.srt",
		Product:      prod,
		Predictions:  preds,
		CallNames:    map[string]string{"12254.mp4": "Arliss_s01e06_PRT261023110725_id12254.mp4"},
		NameTag:      "_s01e06",
		SeasonDir:    "_s01",
		PRTTimestamp: "261023110725",
		Env: Environment{
			Buffer:        "/tmp/lazyam/buffer",
			InProgress:    "/tmp/lazyam/in_progress",
			Done:          "/tmp/lazyam/done",
			Notifications: "/tmp/lazyam/notifications",
			Archive:       "/tmp/lazyam/done/bash",
			TargetRoot:    "/mnt/ROOT/EDIT",
			Priority:      6,
		},
		Destination: "/tmp/lazyam/scripts/test.sh",
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript() error = %v", err)
	}

	if !strings.Contains(script.Content, "cp") {
		t.Error("BuildScript() with SRT does not contain 'cp' for subtitle copy")
	}
	if !strings.Contains(script.Content, "DONE") {
		t.Error("BuildScript() with SRT does not contain DONE for subtitle cleanup")
	}
}

func TestBuildScript_WithHSUB(t *testing.T) {
	prod := metadata.Product{
		ID:            12254,
		SeriesTitle:   "Арлисс",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(1),
		EpisodeNumber: ptrInt(6),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}

	preds := callname.PredictOutputs(prod, ingest.MediaUnit{
		ID:           12254,
		PRTTimestamp: "261023110725",
	}, "/mnt/ROOT/EDIT", "", 12254)

	cfg := ScriptConfig{
		Filenames:   []string{"12254.mp4"},
		Streams:     []ingest.Stream{videoStream},
		HSUBFile:    "Arliss_s01e06_PRT261023110725_id12254_hardsub.srt",
		Product:     prod,
		Predictions: preds,
		CallNames:   map[string]string{"12254.mp4": "Arliss_s01e06_PRT261023110725_id12254.mp4"},
		NameTag:     "_s01e06",
		SeasonDir:   "_s01",
		Env: Environment{
			Buffer:        "/tmp/lazyam/buffer",
			InProgress:    "/tmp/lazyam/in_progress",
			Done:          "/tmp/lazyam/done",
			Notifications: "/tmp/lazyam/notifications",
			Archive:       "/tmp/lazyam/done/bash",
			TargetRoot:    "/mnt/ROOT/EDIT",
			Priority:      6,
		},
		Destination: "/tmp/lazyam/scripts/test.sh",
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript() error = %v", err)
	}

	if !strings.Contains(script.Content, "subtitles=") {
		t.Error("BuildScript() with HSUB does not contain 'subtitles=' filter")
	}
}

func TestBuildScript_WithInterlaced(t *testing.T) {
	prod := metadata.Product{
		ID:            12254,
		SeriesTitle:   "Арлисс",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(1),
		EpisodeNumber: ptrInt(6),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0, Interlaced: true}

	preds := callname.PredictOutputs(prod, ingest.MediaUnit{
		ID:           12254,
		PRTTimestamp: "261023110725",
	}, "/mnt/ROOT/EDIT", "", 12254)

	cfg := ScriptConfig{
		Filenames:    []string{"12254.mp4"},
		Streams:      []ingest.Stream{videoStream},
		Product:      prod,
		Predictions:  preds,
		CallNames:    map[string]string{"12254.mp4": "Arliss_s01e06_PRT261023110725_id12254.mp4"},
		NameTag:      "_s01e06",
		SeasonDir:    "_s01",
		PRTTimestamp: "261023110725",
		Env: Environment{
			Buffer:        "/tmp/lazyam/buffer",
			InProgress:    "/tmp/lazyam/in_progress",
			Done:          "/tmp/lazyam/done",
			Notifications: "/tmp/lazyam/notifications",
			Archive:       "/tmp/lazyam/done/bash",
			TargetRoot:    "/mnt/ROOT/EDIT",
			Priority:      6,
		},
		Destination: "/tmp/lazyam/scripts/test.sh",
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript() error = %v", err)
	}

	if !strings.Contains(script.Content, "yadif") {
		t.Error("BuildScript() with interlaced video does not contain 'yadif' filter")
	}
}

func TestBuildScript_DerivePredictionsFallback(t *testing.T) {
	// When predictions are not set, BuildScript should derive them from streams.
	prod := metadata.Product{
		ID:            99,
		SeriesTitle:   "TestShow",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(2),
		EpisodeNumber: ptrInt(3),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	audioStream := ingest.Stream{Kind: ingest.StreamKindAudio, Index: 1, FileIndex: 0, Language: ingest.Language("eng"), ChannelLayout: ingest.LayoutStereo}

	cfg := ScriptConfig{
		Filenames:    []string{"99.mp4"},
		Streams:      []ingest.Stream{videoStream, audioStream},
		Product:      prod,
		CallNames:    map[string]string{"99.mp4": "TestShow_s02e03_PRT261023110725_id99.mp4"},
		NameTag:      "_s02e03",
		SeasonDir:    "_s02",
		PRTTimestamp: "261023110725",
		Env: Environment{
			Buffer:        "/tmp/lazyam/buffer",
			InProgress:    "/tmp/lazyam/in_progress",
			Done:          "/tmp/lazyam/done",
			Notifications: "/tmp/lazyam/notifications",
			Archive:       "/tmp/lazyam/done/bash",
			TargetRoot:    "/mnt/ROOT/EDIT",
			Priority:      6,
		},
		Destination: "/tmp/lazyam/scripts/test.sh",
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript() derive fallback error = %v", err)
	}

	if script.Content == "" {
		t.Fatal("BuildScript() derive fallback returned empty content")
	}

	unresolved := findUnresolved(script.Content)
	if len(unresolved) > 0 {
		t.Errorf("BuildScript() derive fallback has unresolved placeholders: %v", unresolved)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func ptrInt(i int) *int { return &i }

// ---------------------------------------------------------------------------
// TestBuildScript_validation_errors
// ---------------------------------------------------------------------------

func TestBuildScript_ValidationErrors(t *testing.T) {
	// Empty destination
	prod := metadata.Product{ID: 1, SeriesTitle: "Test"}
	cfg := ScriptConfig{
		Product:     prod,
		Streams:     []ingest.Stream{{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}},
		Predictions: []callname.PredictedOutput{{MediaType: "video", Path: "/tmp/out.mp4"}},
		CallNames:   map[string]string{"test.mp4": "Test.mp4"},
		Env:         Environment{TargetRoot: "/tmp"},
	}
	_, err := BuildScript(cfg)
	if err == nil {
		t.Error("BuildScript(empty destination) expected error, got nil")
	}

	// No streams
	cfg.Destination = "/tmp/test.sh"
	_, err = BuildScript(ScriptConfig{
		Destination: "/tmp/test.sh",
		Product:     prod,
		Streams:     nil,
		CallNames:   map[string]string{},
		Env:         Environment{TargetRoot: "/tmp"},
	})
	if err == nil {
		t.Error("BuildScript(no streams) expected error, got nil")
	}

	// No video streams
	_, err = BuildScript(ScriptConfig{
		Destination: "/tmp/test.sh",
		Product:     prod,
		Streams:     []ingest.Stream{{Kind: ingest.StreamKindAudio, Index: 0, FileIndex: 0}},
		CallNames:   map[string]string{},
		Env:         Environment{TargetRoot: "/tmp"},
	})
	if err == nil {
		t.Error("BuildScript(no video) expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// TestBuildScript_output_contains_expected_elements
// ---------------------------------------------------------------------------

func TestBuildScript_OutputElements(t *testing.T) {
	prod := metadata.Product{
		ID:            42,
		SeriesTitle:   "TestShow",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(1),
		EpisodeNumber: ptrInt(1),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	audioStream := ingest.Stream{Kind: ingest.StreamKindAudio, Index: 1, FileIndex: 0, Language: ingest.Language("rus"), ChannelLayout: ingest.LayoutStereo}
	srtStream := ingest.Stream{Kind: ingest.StreamKindSubtitle, Index: 2, FileIndex: 0}

	preds := callname.PredictOutputs(prod, ingest.MediaUnit{
		ID:           42,
		PRTTimestamp: "261023110725",
	}, "/mnt/ROOT/EDIT", "", 42)

	cfg := ScriptConfig{
		Filenames:    []string{"42.mp4", "42-rus.srt"},
		Streams:      []ingest.Stream{videoStream, audioStream, srtStream},
		SRT:          "TestShow_s01e01_PRT261023110725_id42_rus_Russkii.srt",
		Product:      prod,
		Predictions:  preds,
		CallNames:    map[string]string{"42.mp4": "TestShow_s01e01_PRT261023110725_id42.mp4", "42-rus.srt": "TestShow_s01e01_PRT261023110725_id42_rus_Russkii.srt"},
		NameTag:      "_s01e01",
		SeasonDir:    "_s01",
		PRTTimestamp: "261023110725",
		Env: Environment{
			Buffer:        "/tmp/lazyam/buffer",
			InProgress:    "/tmp/lazyam/in_progress",
			Done:          "/tmp/lazyam/done",
			Notifications: "/tmp/lazyam/notifications",
			Archive:       "/tmp/lazyam/done/bash",
			TargetRoot:    "/mnt/ROOT/EDIT",
			Priority:      6,
		},
		Destination: "/tmp/lazyam/scripts/test.sh",
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript() error = %v", err)
	}

	// Check for expected script elements
	checks := []struct{ name, substr string }{
		{"shebang", "#!/bin/bash"},
		{"ffmpeg command", "ffmpeg"},
		{"source variable", "SOURCE="},
		{"buffer variable", "BUFFER="},
		{"in_progress variable", "IN_PROGRESS="},
		{"done variable", "DONE="},
		{"mkdir target", "mkdir -p"},
		{"priority", "PRIORITY="},
		{"archive", "ARCHIVE="},
		{"notifications", "NOTIFICATIONS="},
	}

	for _, chk := range checks {
		if !strings.Contains(script.Content, chk.substr) {
			t.Errorf("BuildScript() content missing %q: %q", chk.name, chk.substr)
		}
	}

	// Verify .ready file is generated
	if !strings.Contains(script.Content, ".ready") {
		t.Error("BuildScript() content missing .ready file generation")
	}

	// Verify source file moves to in_progress
	if !strings.Contains(script.Content, "mv") {
		t.Error("BuildScript() content missing 'mv' commands")
	}
}

// ---------------------------------------------------------------------------
// TestBuildScript_environment_variables
// ---------------------------------------------------------------------------

func TestBuildScript_EnvironmentVariables(t *testing.T) {
	prod := metadata.Product{
		ID:            1,
		SeriesTitle:   "EnvTest",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(1),
		EpisodeNumber: ptrInt(1),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	preds := callname.PredictOutputs(prod, ingest.MediaUnit{
		ID:           1,
		PRTTimestamp: "261023110725",
	}, "/mnt/ROOT/EDIT", "", 1)

	env := Environment{
		Buffer:        "/custom/buffer",
		InProgress:    "/custom/in_progress",
		Done:          "/custom/done",
		Notifications: "/custom/notifications",
		Archive:       "/custom/archive",
		TargetRoot:    "/custom/target",
		Priority:      99,
	}

	cfg := ScriptConfig{
		Filenames:    []string{"1.mp4"},
		Streams:      []ingest.Stream{videoStream},
		Product:      prod,
		Predictions:  preds,
		CallNames:    map[string]string{"1.mp4": "EnvTest_s01e01_PRT261023110725_id1.mp4"},
		NameTag:      "_s01e01",
		SeasonDir:    "_s01",
		PRTTimestamp: "261023110725",
		Env:          env,
		Destination:  "/tmp/test.sh",
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript() error = %v", err)
	}

	// Check that custom paths are embedded
	if !strings.Contains(script.Content, "/custom/buffer") {
		t.Error("BuildScript() does not contain custom Buffer path")
	}
	if !strings.Contains(script.Content, "/custom/in_progress") {
		t.Error("BuildScript() does not contain custom InProgress path")
	}
	if !strings.Contains(script.Content, "/custom/done") {
		t.Error("BuildScript() does not contain custom Done path")
	}
	if !strings.Contains(script.Content, "PRIORITY=99") {
		t.Error("BuildScript() does not contain custom priority")
	}
}

// ---------------------------------------------------------------------------
// TestDerivePredictionsFromStreams_stream_ordering
// ---------------------------------------------------------------------------

func TestDerivePredictionsFromStreams_Ordering(t *testing.T) {
	prod := metadata.Product{
		ID:            100,
		SeriesTitle:   "OrderTest",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(3),
		EpisodeNumber: ptrInt(7),
	}

	// Streams in non-sorted order
	streams := []ingest.Stream{
		{Kind: ingest.StreamKindAudio, Index: 2, FileIndex: 0, Language: ingest.Language("jpn")},
		{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0},
		{Kind: ingest.StreamKindAudio, Index: 1, FileIndex: 0, Language: ingest.Language("rus")},
	}

	preds := derivePredictionsFromStreams(prod, streams, "/mnt/ROOT/EDIT", "261023110725")

	// First should be video
	if len(preds) == 0 {
		t.Fatal("no predictions")
	}
	if preds[0].MediaType != "video" {
		t.Errorf("first prediction = %q, want 'video'", preds[0].MediaType)
	}

	// Should have 3 predictions total
	if len(preds) != 3 {
		t.Errorf("predictions count = %d, want 3", len(preds))
	}
}

// ---------------------------------------------------------------------------
// TestBuildSRTSection_edge_cases
// ---------------------------------------------------------------------------

func TestBuildSRTSection_EdgeCases(t *testing.T) {
	// SRT with prediction that has no video — falls back to first prediction, no subtitle prediction so uses input filename
	preds := []callname.PredictedOutput{
		{MediaType: "audio", Path: "/mnt/ROOT/EDIT/Show/H_AUDIO.m4a"},
	}
	got := buildSRTSection("subs.srt", preds)
	want := `cp "${IN_PROGRESS}/${SRT}" "/mnt/ROOT/EDIT/Show/subs.srt"`
	if got != want {
		t.Errorf("buildSRTSection fallback = %q, want %q", got, want)
	}

	// With subtitle prediction, uses prediction filename
	preds2 := []callname.PredictedOutput{
		{MediaType: "audio", Path: "/mnt/ROOT/EDIT/Show/H_AUDIO.m4a"},
		{MediaType: "subtitle", Path: "/mnt/ROOT/EDIT/Show/Show_HD.srt"},
	}
	got = buildSRTSection("subs.srt", preds2)
	want = `cp "${IN_PROGRESS}/${SRT}" "/mnt/ROOT/EDIT/Show/Show_HD.srt"`
	if got != want {
		t.Errorf("buildSRTSection with subtitle prediction = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// TestBuildReadyFile_edge_cases
// ---------------------------------------------------------------------------

func TestBuildReadyFile_EdgeCases(t *testing.T) {
	// Different file extensions
	tests := []struct {
		name string
		path string
		want string
	}{
		{"mp4", "/mnt/ROOT/EDIT/H_HD.mp4", `touch "/mnt/ROOT/EDIT/H_HD.ready"`},
		{"mkv", "/mnt/ROOT/EDIT/H_HD.mkv", `touch "/mnt/ROOT/EDIT/H_HD.ready"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preds := []callname.PredictedOutput{{MediaType: "video", Path: tt.path}}
			got := buildReadyFile(preds)
			if got != tt.want {
				t.Errorf("buildReadyFile(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestBuildScript_multiple_audio_tracks
// ---------------------------------------------------------------------------

func TestBuildScript_MultipleAudioTracks(t *testing.T) {
	prod := metadata.Product{
		ID:            200,
		SeriesTitle:   "MultiLang",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(1),
		EpisodeNumber: ptrInt(1),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	audio1 := ingest.Stream{Kind: ingest.StreamKindAudio, Index: 1, FileIndex: 0, Language: ingest.Language("rus"), ChannelLayout: ingest.LayoutStereo}
	audio2 := ingest.Stream{Kind: ingest.StreamKindAudio, Index: 2, FileIndex: 0, Language: ingest.Language("eng"), ChannelLayout: ingest.LayoutSurround51}

	// Add streams to the MediaUnit so PredictOutputs can create audio predictions
	mediaUnit := ingest.MediaUnit{
		ID:           200,
		PRTTimestamp: "261023110725",
		Streams: map[string]ingest.Stream{
			"[0:v:0]": videoStream,
			"[0:a:0]": audio1,
			"[0:a:1]": audio2,
		},
	}
	preds := callname.PredictOutputs(prod, mediaUnit, "/mnt/ROOT/EDIT", "", 200)

	cfg := ScriptConfig{
		Filenames:    []string{"200.mp4"},
		Streams:      []ingest.Stream{videoStream, audio1, audio2},
		Product:      prod,
		Predictions:  preds,
		CallNames:    map[string]string{"200.mp4": "MultiLang_s01e01_PRT261023110725_id200.mp4"},
		NameTag:      "_s01e01",
		SeasonDir:    "_s01",
		PRTTimestamp: "261023110725",
		Env: Environment{
			Buffer:        "/tmp/lazyam/buffer",
			InProgress:    "/tmp/lazyam/in_progress",
			Done:          "/tmp/lazyam/done",
			Notifications: "/tmp/lazyam/notifications",
			Archive:       "/tmp/lazyam/done/bash",
			TargetRoot:    "/mnt/ROOT/EDIT",
			Priority:      6,
		},
		Destination: "/tmp/lazyam/scripts/test.sh",
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript() error = %v", err)
	}

	// Both audio tracks should be in the filter_complex
	// Both audio tracks should produce [aud0] and [aud1] labels in filter_complex
	// The audio_filter placeholder contains the segments joined with semicolons
	audioFilter := buildAudioFilter(selectAudioTracksFromPredictions(cfg))
	if !strings.Contains(audioFilter, "[aud0]") || !strings.Contains(audioFilter, "[aud1]") {
		t.Errorf("audio filter missing [aud0] and/or [aud1]: %q", audioFilter)
	}

	// Check that stereo (20) and 5.1 (51) layouts are present
	if !strings.Contains(script.Content, "20") {
		t.Error("BuildScript() missing stereo layout suffix")
	}
}

// ---------------------------------------------------------------------------
// TestBuildSRTMove_HSUB_Move
// ---------------------------------------------------------------------------

func TestBuildSRTMove_HSUB_Move(t *testing.T) {
	// SRT and HSUB move should both produce shell commands
	srtMove := buildSRTMove("subs.srt")
	hsubMove := buildHSUBMove("hsub.srt")

	if srtMove == "" {
		t.Error("buildSRTMove with value should not be empty")
	}
	if hsubMove == "" {
		t.Error("buildHSUBMove with value should not be empty")
	}

	// Both should reference their respective shell variables
	if !strings.Contains(srtMove, "${SRT}") {
		t.Error("buildSRTMove should reference ${SRT}")
	}
	if !strings.Contains(hsubMove, "${HSUBFILE}") {
		t.Error("buildHSUBMove should reference ${HSUBFILE}")
	}
}

// ---------------------------------------------------------------------------
// TestBuildHSUB_path_resolution
// ---------------------------------------------------------------------------

func TestBuildHSUB_PathResolution(t *testing.T) {
	// With empty inProgress, should use hsubFile as-is
	got := buildHSUB("hardsub.srt", "", defaultHarsub)
	if !strings.Contains(got, "hardsub.srt") {
		t.Error("buildHSUB with empty inProgress should contain filename")
	}

	// With non-empty inProgress, should join paths
	got = buildHSUB("hardsub.srt", "/tmp/in_progress", defaultHarsub)
	expectedPath := filepath.ToSlash(filepath.Join("/tmp/in_progress", "hardsub.srt"))
	if !strings.Contains(got, expectedPath) {
		t.Errorf("buildHSUB with inProgress should contain %q, got: %q", expectedPath, got)
	}
}

// ---------------------------------------------------------------------------
// TestBuildMKDirTarget_path_formatting
// ---------------------------------------------------------------------------

func TestBuildMKDirTarget_PathFormatting(t *testing.T) {
	// Windows-style path should be converted to forward slashes
	preds := []callname.PredictedOutput{
		{MediaType: "video", Path: filepath.Join("C:", "Users", "Test", "out.mp4")},
	}
	got := buildMKDirTarget(preds)
	// Should use forward slashes
	if strings.Contains(got, "\\") {
		t.Error("buildMKDirTarget should use forward slashes, got backslashes")
	}
}

// ---------------------------------------------------------------------------
// TestBuildScript_predictable_output
// ---------------------------------------------------------------------------

func TestBuildScript_PredictableOutput(t *testing.T) {
	// Two identical configs should produce identical scripts
	prod := metadata.Product{
		ID:            300,
		SeriesTitle:   "Deterministic",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(1),
		EpisodeNumber: ptrInt(1),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	preds := callname.PredictOutputs(prod, ingest.MediaUnit{
		ID:           300,
		PRTTimestamp: "261023110725",
	}, "/mnt/ROOT/EDIT", "", 300)

	cfg1 := ScriptConfig{
		Filenames:    []string{"300.mp4"},
		Streams:      []ingest.Stream{videoStream},
		Product:      prod,
		Predictions:  preds,
		CallNames:    map[string]string{"300.mp4": "Deterministic_s01e01_PRT261023110725_id300.mp4"},
		NameTag:      "_s01e01",
		SeasonDir:    "_s01",
		PRTTimestamp: "261023110725",
		Env: Environment{
			Buffer:        "/tmp/lazyam/buffer",
			InProgress:    "/tmp/lazyam/in_progress",
			Done:          "/tmp/lazyam/done",
			Notifications: "/tmp/lazyam/notifications",
			Archive:       "/tmp/lazyam/done/bash",
			TargetRoot:    "/mnt/ROOT/EDIT",
			Priority:      6,
		},
		Destination: "/tmp/lazyam/scripts/test.sh",
	}

	cfg2 := cfg1 // exact copy

	script1, err1 := BuildScript(cfg1)
	script2, err2 := BuildScript(cfg2)

	if err1 != nil || err2 != nil {
		t.Fatalf("BuildScript() errors: %v, %v", err1, err2)
	}

	if script1.Content != script2.Content {
		t.Error("BuildScript() is not deterministic: identical configs produced different output")
	}
}

// ---------------------------------------------------------------------------
// TestBuildScript_ReadyFile_path
// ---------------------------------------------------------------------------

func TestBuildScript_ReadyFilePath(t *testing.T) {
	prod := metadata.Product{
		ID:            400,
		SeriesTitle:   "ReadyTest",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(1),
		EpisodeNumber: ptrInt(1),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	preds := callname.PredictOutputs(prod, ingest.MediaUnit{
		ID:           400,
		PRTTimestamp: "261023110725",
	}, "/mnt/ROOT/EDIT", "", 400)

	cfg := ScriptConfig{
		Filenames:    []string{"400.mp4"},
		Streams:      []ingest.Stream{videoStream},
		Product:      prod,
		Predictions:  preds,
		CallNames:    map[string]string{"400.mp4": "ReadyTest_s01e01_PRT261023110725_id400.mp4"},
		NameTag:      "_s01e01",
		SeasonDir:    "_s01",
		PRTTimestamp: "261023110725",
		Env: Environment{
			Buffer:        "/tmp/lazyam/buffer",
			InProgress:    "/tmp/lazyam/in_progress",
			Done:          "/tmp/lazyam/done",
			Notifications: "/tmp/lazyam/notifications",
			Archive:       "/tmp/lazyam/done/bash",
			TargetRoot:    "/mnt/ROOT/EDIT",
			Priority:      6,
		},
		Destination: "/tmp/lazyam/scripts/test.sh",
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript() error = %v", err)
	}

	// The ready file should be in the same directory as the video
	if !strings.Contains(script.Content, ".ready") {
		t.Error("BuildScript() missing .ready file touch command")
	}
}

// ---------------------------------------------------------------------------
// TestFindUnresolved_edge_cases
// ---------------------------------------------------------------------------

func TestFindUnresolved_EdgeCases(t *testing.T) {
	// Similar but non-matching patterns
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"no pipes", "hello world", 0},
		{"pipe but no equals", "|foo|", 0},
		{"equals but no pipe", "=foo=", 0},
		{"nested placeholders", "|=|=foo=||=", 1},
		{"multiple spaces", "|= foo |= |= bar |=|", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findUnresolved(tt.input)
			if len(got) != tt.want {
				t.Errorf("findUnresolved(%q) got %d results, want %d", tt.input, len(got), tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestBuildScript_empty_predictions_with_fallback
// ---------------------------------------------------------------------------

func TestBuildScript_EmptyPredictionsFallback(t *testing.T) {
	// When predictions are empty and streams provide enough info,
	// BuildScript should derive them and still produce a valid script.
	prod := metadata.Product{
		ID:            500,
		SeriesTitle:   "FallbackTest",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(1),
		EpisodeNumber: ptrInt(1),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	audioStream := ingest.Stream{Kind: ingest.StreamKindAudio, Index: 1, FileIndex: 0, Language: ingest.Language("rus"), ChannelLayout: ingest.LayoutStereo}

	// Empty predictions — should trigger derivePredictionsFromStreams
	cfg := ScriptConfig{
		Filenames:    []string{"500.mp4"},
		Streams:      []ingest.Stream{videoStream, audioStream},
		Product:      prod,
		CallNames:    map[string]string{"500.mp4": "FallbackTest_s01e01_PRT261023110725_id500.mp4"},
		NameTag:      "_s01e01",
		SeasonDir:    "_s01",
		PRTTimestamp: "261023110725",
		Env: Environment{
			Buffer:        "/tmp/lazyam/buffer",
			InProgress:    "/tmp/lazyam/in_progress",
			Done:          "/tmp/lazyam/done",
			Notifications: "/tmp/lazyam/notifications",
			Archive:       "/tmp/lazyam/done/bash",
			TargetRoot:    "/mnt/ROOT/EDIT",
			Priority:      6,
		},
		Destination: "/tmp/lazyam/scripts/test.sh",
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript() empty predictions fallback error = %v", err)
	}

	if script.Content == "" {
		t.Fatal("BuildScript() empty predictions fallback returned empty content")
	}

	unresolved := findUnresolved(script.Content)
	if len(unresolved) > 0 {
		t.Errorf("BuildScript() fallback has unresolved placeholders: %v", unresolved)
	}
}

// ---------------------------------------------------------------------------
// TestBuildScript_MovieProduct
// ---------------------------------------------------------------------------

func TestBuildScript_MovieProduct(t *testing.T) {
	prod := metadata.Product{
		ID:          600,
		SeriesTitle: "MovieTitle",
		Kind:        metadata.KindMovie,
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	preds := callname.PredictOutputs(prod, ingest.MediaUnit{
		ID:           600,
		PRTTimestamp: "261023110725",
	}, "/mnt/ROOT/EDIT", "", 600)

	cfg := ScriptConfig{
		Filenames:    []string{"600.mp4"},
		Streams:      []ingest.Stream{videoStream},
		Product:      prod,
		Predictions:  preds,
		CallNames:    map[string]string{"600.mp4": "MovieTitle_PRT261023110725_id600.mp4"},
		NameTag:      "",
		SeasonDir:    "",
		PRTTimestamp: "261023110725",
		Env: Environment{
			Buffer:        "/tmp/lazyam/buffer",
			InProgress:    "/tmp/lazyam/in_progress",
			Done:          "/tmp/lazyam/done",
			Notifications: "/tmp/lazyam/notifications",
			Archive:       "/tmp/lazyam/done/bash",
			TargetRoot:    "/mnt/ROOT/EDIT",
			Priority:      6,
		},
		Destination: "/tmp/lazyam/scripts/test.sh",
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript(movie) error = %v", err)
	}

	if script.Content == "" {
		t.Fatal("BuildScript(movie) returned empty content")
	}

	unresolved := findUnresolved(script.Content)
	if len(unresolved) > 0 {
		t.Errorf("BuildScript(movie) has unresolved placeholders: %v", unresolved)
	}
}

// ---------------------------------------------------------------------------
// TestBuildScript_EnvironmentVariables_Unique
// ---------------------------------------------------------------------------

func TestBuildScript_EnvironmentVariables_Unique(t *testing.T) {
	prod := metadata.Product{
		ID:            700,
		SeriesTitle:   "EnvUnique",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(1),
		EpisodeNumber: ptrInt(1),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	preds := callname.PredictOutputs(prod, ingest.MediaUnit{
		ID:           700,
		PRTTimestamp: "261023110725",
	}, "/mnt/ROOT/EDIT", "", 700)

	// Use paths that are unlikely to appear by accident
	cfg := ScriptConfig{
		Filenames:    []string{"700.mp4"},
		Streams:      []ingest.Stream{videoStream},
		Product:      prod,
		Predictions:  preds,
		CallNames:    map[string]string{"700.mp4": "EnvUnique_s01e01_PRT261023110725_id700.mp4"},
		NameTag:      "_s01e01",
		SeasonDir:    "_s01",
		PRTTimestamp: "261023110725",
		Env: Environment{
			Buffer:        "/unique/buffer/path",
			InProgress:    "/unique/in_progress/path",
			Done:          "/unique/done/path",
			Notifications: "/unique/notifications/path",
			Archive:       "/unique/archive/path",
			TargetRoot:    "/unique/target/path",
			Priority:      42,
		},
		Destination: "/tmp/lazyam/scripts/test.sh",
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript() error = %v", err)
	}

	// Verify all custom paths appear in the script
	uniquePaths := []string{
		"/unique/buffer/path",
		"/unique/in_progress/path",
		"/unique/done/path",
		"/unique/notifications/path",
		"/unique/archive/path",
	}
	for _, p := range uniquePaths {
		if !strings.Contains(script.Content, p) {
			t.Errorf("BuildScript() missing custom path %q", p)
		}
	}

	// Priority should be embedded
	if !strings.Contains(script.Content, "PRIORITY=42") {
		t.Error("BuildScript() missing custom priority value")
	}
}

// ---------------------------------------------------------------------------
// TestBuildScript_CallNamesUsed
// ---------------------------------------------------------------------------

func TestBuildScript_CallNamesUsed(t *testing.T) {
	prod := metadata.Product{
		ID:            800,
		SeriesTitle:   "CallNameTest",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(1),
		EpisodeNumber: ptrInt(1),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	preds := callname.PredictOutputs(prod, ingest.MediaUnit{
		ID:           800,
		PRTTimestamp: "261023110725",
	}, "/mnt/ROOT/EDIT", "", 800)

	callName := "CallNameTest_s01e01_PRT261023110725_id800.mp4"
	cfg := ScriptConfig{
		Filenames:    []string{"original_800.mp4"},
		Streams:      []ingest.Stream{videoStream},
		Product:      prod,
		Predictions:  preds,
		CallNames:    map[string]string{"original_800.mp4": callName},
		NameTag:      "_s01e01",
		SeasonDir:    "_s01",
		PRTTimestamp: "261023110725",
		Env: Environment{
			Buffer:        "/tmp/lazyam/buffer",
			InProgress:    "/tmp/lazyam/in_progress",
			Done:          "/tmp/lazyam/done",
			Notifications: "/tmp/lazyam/notifications",
			Archive:       "/tmp/lazyam/done/bash",
			TargetRoot:    "/mnt/ROOT/EDIT",
			Priority:      6,
		},
		Destination: "/tmp/lazyam/scripts/test.sh",
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript() error = %v", err)
	}

	// SOURCE should be the call name, not the original filename
	if !strings.Contains(script.Content, fmt.Sprintf(`SOURCE="%s"`, callName)) {
		t.Errorf("BuildScript() SOURCE not set to call name: expected %q in content", callName)
	}

	// Original filename should NOT appear as SOURCE
	if strings.Contains(script.Content, `SOURCE="original_800.mp4"`) {
		t.Error("BuildScript() SOURCE should not be the original filename")
	}
}

// ---------------------------------------------------------------------------
// TestBuildScript_ReadyFile_DirectoryMatch
// ---------------------------------------------------------------------------

func TestBuildScript_ReadyFile_DirectoryMatch(t *testing.T) {
	prod := metadata.Product{
		ID:            900,
		SeriesTitle:   "ReadyDirTest",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(1),
		EpisodeNumber: ptrInt(1),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	preds := callname.PredictOutputs(prod, ingest.MediaUnit{
		ID:           900,
		PRTTimestamp: "261023110725",
	}, "/mnt/ROOT/EDIT", "", 900)

	cfg := ScriptConfig{
		Filenames:    []string{"900.mp4"},
		Streams:      []ingest.Stream{videoStream},
		Product:      prod,
		Predictions:  preds,
		CallNames:    map[string]string{"900.mp4": "ReadyDirTest_s01e01_PRT261023110725_id900.mp4"},
		NameTag:      "_s01e01",
		SeasonDir:    "_s01",
		PRTTimestamp: "261023110725",
		Env: Environment{
			Buffer:        "/tmp/lazyam/buffer",
			InProgress:    "/tmp/lazyam/in_progress",
			Done:          "/tmp/lazyam/done",
			Notifications: "/tmp/lazyam/notifications",
			Archive:       "/tmp/lazyam/done/bash",
			TargetRoot:    "/mnt/ROOT/EDIT",
			Priority:      6,
		},
		Destination: "/tmp/lazyam/scripts/test.sh",
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript() error = %v", err)
	}

	// The ready file should be created in the same directory as the video
	// Look for the touch command and verify it ends with .ready
	if !strings.Contains(script.Content, ".ready") {
		t.Error("BuildScript() missing .ready file touch command")
	}
}

func TestBuildScript_FfmpegBinary(t *testing.T) {
	prod := metadata.Product{
		ID:            900,
		SeriesTitle:   "FfmpegTest",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(1),
		EpisodeNumber: ptrInt(1),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	preds := callname.PredictOutputs(prod, ingest.MediaUnit{
		ID:           900,
		PRTTimestamp: "261023110725",
	}, "/mnt/ROOT/EDIT", "", 900)

	callName := "FfmpegTest_s01e01_PRT261023110725_id900.mp4"

	// Custom ffmpeg path
	cfg := ScriptConfig{
		Filenames:    []string{"900.mp4"},
		Streams:      []ingest.Stream{videoStream},
		Product:      prod,
		Predictions:  preds,
		CallNames:    map[string]string{"900.mp4": callName},
		NameTag:      "_s01e01",
		SeasonDir:    "_s01",
		PRTTimestamp: "261023110725",
		FfmpegBinary: "/opt/bin/ffmpeg-nvenc",
		Env: Environment{
			Buffer:        "/tmp/lazyam/buffer",
			InProgress:    "/tmp/lazyam/in_progress",
			Done:          "/tmp/lazyam/done",
			Notifications: "/tmp/lazyam/notifications",
			Archive:       "/tmp/lazyam/done/bash",
			TargetRoot:    "/mnt/ROOT/EDIT",
			Priority:      6,
		},
		Destination: "/tmp/lazyam/scripts/test.sh",
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript() error = %v", err)
	}

	if !strings.Contains(script.Content, "/opt/bin/ffmpeg-nvenc") {
		t.Error("BuildScript() with custom FfmpegBinary should contain custom path")
	}
}

func TestBuildScript_FfmpegBinaryFallback(t *testing.T) {
	prod := metadata.Product{
		ID:            950,
		SeriesTitle:   "FfmpegFallback",
		Kind:          metadata.KindSeriesEpisode,
		SeasonNumber:  ptrInt(1),
		EpisodeNumber: ptrInt(1),
	}

	videoStream := ingest.Stream{Kind: ingest.StreamKindVideo, Index: 0, FileIndex: 0}
	preds := callname.PredictOutputs(prod, ingest.MediaUnit{
		ID:           950,
		PRTTimestamp: "261023110725",
	}, "/mnt/ROOT/EDIT", "", 950)

	callName := "FfmpegFallback_s01e01_PRT261023110725_id950.mp4"

	// Empty ffmpeg binary — should fallback to "ffmpeg"
	cfg := ScriptConfig{
		Filenames:    []string{"950.mp4"},
		Streams:      []ingest.Stream{videoStream},
		Product:      prod,
		Predictions:  preds,
		CallNames:    map[string]string{"950.mp4": callName},
		NameTag:      "_s01e01",
		SeasonDir:    "_s01",
		PRTTimestamp: "261023110725",
		FfmpegBinary: "",
		Env: Environment{
			Buffer:        "/tmp/lazyam/buffer",
			InProgress:    "/tmp/lazyam/in_progress",
			Done:          "/tmp/lazyam/done",
			Notifications: "/tmp/lazyam/notifications",
			Archive:       "/tmp/lazyam/done/bash",
			TargetRoot:    "/mnt/ROOT/EDIT",
			Priority:      6,
		},
		Destination: "/tmp/lazyam/scripts/test.sh",
	}

	script, err := BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript() error = %v", err)
	}

	if !strings.Contains(script.Content, "ffmpeg") {
		t.Error("BuildScript() with empty FfmpegBinary should fallback to 'ffmpeg'")
	}

	// No unresolved placeholders
	unresolved := findUnresolved(script.Content)
	if len(unresolved) > 0 {
		t.Errorf("BuildScript() has unresolved placeholders: %v", unresolved)
	}
}
