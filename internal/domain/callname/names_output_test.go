package callname

import (
	"testing"

	"github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/Galdoba/lazyam3/internal/domain/metadata"
)

// ── PredictOutputs ──────────────────────────────────────────────────

func newUnit(product metadata.Product, filenames []string, streams []ingest.Stream) ingest.MediaUnit {
	streamMap := make(map[string]ingest.Stream)
	fileIdx := 0
	for _, s := range streams {
		s.FileIndex = fileIdx
		streamMap[string(s.Kind)+string(rune(s.Index))] = s
	}
	return ingest.MediaUnit{
		ID:        product.ID,
		Filenames: filenames,
		Streams:   streamMap,
	}
}

func TestPredictOutputs_series_episode(t *testing.T) {
	season := 1
	episode := 6
	product := metadata.Product{
		ID:            12345,
		Kind:          metadata.KindSeriesEpisode,
		SeriesTitle:   "Arliss",
		SeasonNumber:  &season,
		EpisodeNumber: &episode,
	}
	streams := []ingest.Stream{
		{Kind: ingest.StreamKindVideo, Index: 0},
		{Kind: ingest.StreamKindAudio, Index: 1, Language: "rus", ChannelLayout: ingest.LayoutStereo},
		{Kind: ingest.StreamKindAudio, Index: 2, Language: "jpn", ChannelLayout: ingest.LayoutSurround51},
		{Kind: ingest.StreamKindSubtitle, Index: 3, Language: "rus"},
	}
	unit := newUnit(product, []string{"12345.mp4", "12345-rus.srt"}, streams)

	got := PredictOutputs(product, unit, "/mnt/ROOT/EDIT", "")

	if len(got) != 4 {
		t.Fatalf("PredictOutputs() = %d outputs, want 4", len(got))
	}

	// Video
	if got[0].MediaType != "video" {
		t.Errorf("[0] MediaType = %q, want %q", got[0].MediaType, "video")
	}
	if got[0].HsubFile != "" {
		t.Errorf("[0] HsubFile = %q, want %q", got[0].HsubFile, "")
	}
	wantVideo := "/mnt/ROOT/EDIT/Arliss_s01/Arliss_s01e06_HD.mp4"
	if got[0].Path != wantVideo {
		t.Errorf("[0] Path = %q, want %q", got[0].Path, wantVideo)
	}

	// Audio 0
	if got[1].MediaType != "audio" {
		t.Errorf("[1] MediaType = %q, want %q", got[1].MediaType, "audio")
	}
	if got[1].AudioIndex != 0 {
		t.Errorf("[1] AudioIndex = %d, want 0", got[1].AudioIndex)
	}
	if got[1].Language != "rus" {
		t.Errorf("[1] Language = %q, want %q", got[1].Language, "rus")
	}
	if got[1].Layout != "20" {
		t.Errorf("[1] Layout = %q, want %q", got[1].Layout, "20")
	}
	wantAudio0 := "/mnt/ROOT/EDIT/Arliss_s01/Arliss_s01e06_HD_AUDIO00_RUS20.m4a"
	if got[1].Path != wantAudio0 {
		t.Errorf("[1] Path = %q, want %q", got[1].Path, wantAudio0)
	}

	// Audio 1
	if got[2].Layout != "51" {
		t.Errorf("[2] Layout = %q, want %q", got[2].Layout, "51")
	}
	if got[2].Language != "jpn" {
		t.Errorf("[2] Language = %q, want %q", got[2].Language, "jpn")
	}
	wantAudio1 := "/mnt/ROOT/EDIT/Arliss_s01/Arliss_s01e06_HD_AUDIO01_JPN51.m4a"
	if got[2].Path != wantAudio1 {
		t.Errorf("[2] Path = %q, want %q", got[2].Path, wantAudio1)
	}

	// Subtitle
	if got[3].MediaType != "subtitle" {
		t.Errorf("[3] MediaType = %q, want %q", got[3].MediaType, "subtitle")
	}
	wantSub := "/mnt/ROOT/EDIT/Arliss_s01/Arliss_s01e06.srt"
	if got[3].Path != wantSub {
		t.Errorf("[3] Path = %q, want %q", got[3].Path, wantSub)
	}
}

func TestPredictOutputs_series_episode_with_hardsub(t *testing.T) {
	season := 1
	episode := 6
	product := metadata.Product{
		ID:            12345,
		Kind:          metadata.KindSeriesEpisode,
		SeriesTitle:   "Arliss",
		SeasonNumber:  &season,
		EpisodeNumber: &episode,
	}
	streams := []ingest.Stream{
		{Kind: ingest.StreamKindVideo, Index: 0},
		{Kind: ingest.StreamKindAudio, Index: 1, Language: "rus", ChannelLayout: ingest.LayoutStereo},
		{Kind: ingest.StreamKindSubtitle, Index: 2, Language: "rus"},
	}
	unit := newUnit(product, []string{"12345.mp4", "12345-hardsub-rus.srt"}, streams)

	got := PredictOutputs(product, unit, "/mnt/ROOT/EDIT", "")

	if got[0].HsubFile != "12345-hardsub-rus.srt" {
		t.Errorf("[0] HsubFile = %q, want %q", got[0].HsubFile, "12345-hardsub-rus.srt")
	}
}

func TestPredictOutputs_movie_no_season(t *testing.T) {
	product := metadata.Product{
		ID:      12254,
		Kind:    metadata.KindMovie,
		SeriesTitle: "12254",
	}
	streams := []ingest.Stream{
		{Kind: ingest.StreamKindVideo, Index: 0},
		{Kind: ingest.StreamKindAudio, Index: 1, Language: "rus", ChannelLayout: ingest.LayoutSurround51},
	}
	unit := newUnit(product, []string{"12254.mp4"}, streams)

	got := PredictOutputs(product, unit, "/mnt/ROOT/EDIT", "")

	if len(got) != 2 {
		t.Fatalf("PredictOutputs() = %d outputs, want 2", len(got))
	}

	// Movie: NameTag is "", so directory is targetRoot/BaseName/
	wantVideo := "/mnt/ROOT/EDIT/12254/12254_HD.mp4"
	if got[0].Path != wantVideo {
		t.Errorf("[0] Path = %q, want %q", got[0].Path, wantVideo)
	}
	wantAudio := "/mnt/ROOT/EDIT/12254/12254_HD_AUDIO00_RUS51.m4a"
	if got[1].Path != wantAudio {
		t.Errorf("[1] Path = %q, want %q", got[1].Path, wantAudio)
	}
}

func TestPredictOutputs_no_audio(t *testing.T) {
	product := metadata.Product{
		ID:           1,
		Kind:         metadata.KindSeriesEpisode,
		SeriesTitle:  "Show",
		SeasonNumber: ptrInt(2),
		EpisodeNumber: ptrInt(5),
	}
	streams := []ingest.Stream{
		{Kind: ingest.StreamKindVideo, Index: 0},
	}
	unit := newUnit(product, []string{"1.mp4"}, streams)

	got := PredictOutputs(product, unit, "/mnt/ROOT/EDIT", "")

	if len(got) != 1 {
		t.Fatalf("PredictOutputs() = %d outputs, want 1", len(got))
	}
	if got[0].MediaType != "video" {
		t.Errorf("[0] MediaType = %q, want %q", got[0].MediaType, "video")
	}
}

func TestPredictOutputs_no_subtitle(t *testing.T) {
	product := metadata.Product{
		ID:           1,
		Kind:         metadata.KindSeriesEpisode,
		SeriesTitle:  "Show",
		SeasonNumber: ptrInt(1),
		EpisodeNumber: ptrInt(1),
	}
	streams := []ingest.Stream{
		{Kind: ingest.StreamKindVideo, Index: 0},
		{Kind: ingest.StreamKindAudio, Index: 1, Language: "eng", ChannelLayout: ingest.LayoutStereo},
	}
	unit := newUnit(product, []string{"1.mp4"}, streams)

	got := PredictOutputs(product, unit, "/mnt/ROOT/EDIT", "")

	for _, o := range got {
		if o.MediaType == "subtitle" {
			t.Errorf("unexpected subtitle output")
		}
	}
}

func TestPredictOutputs_unknown_language(t *testing.T) {
	product := metadata.Product{
		ID:           1,
		Kind:         metadata.KindSeriesEpisode,
		SeriesTitle:  "X",
		SeasonNumber: ptrInt(1),
		EpisodeNumber: ptrInt(1),
	}
	streams := []ingest.Stream{
		{Kind: ingest.StreamKindVideo, Index: 0},
		{Kind: ingest.StreamKindAudio, Index: 1}, // no language
	}
	unit := newUnit(product, []string{"1.mp4"}, streams)

	got := PredictOutputs(product, unit, "/mnt/ROOT/EDIT", "")

	if got[1].Language != "und" {
		t.Errorf("[1] Language = %q, want %q", got[1].Language, "und")
	}
}

func TestPredictOutputs_empty_streams(t *testing.T) {
	product := metadata.Product{
		ID:           1,
		Kind:         metadata.KindSeriesEpisode,
		SeriesTitle:  "Test",
		SeasonNumber: ptrInt(1),
		EpisodeNumber: ptrInt(1),
	}
	unit := newUnit(product, []string{"1.mp4"}, nil)

	got := PredictOutputs(product, unit, "/mnt/ROOT/EDIT", "")

	if len(got) != 1 {
		t.Fatalf("PredictOutputs() = %d outputs, want 1", len(got))
	}
	if got[0].MediaType != "video" {
		t.Errorf("[0] MediaType = %q, want %q", got[0].MediaType, "video")
	}
}

func TestPredictOutputs_season_only(t *testing.T) {
	product := metadata.Product{
		ID:           1,
		Kind:         metadata.KindSeason,
		SeriesTitle:  "Show",
		SeasonNumber: ptrInt(3),
	}
	streams := []ingest.Stream{
		{Kind: ingest.StreamKindVideo, Index: 0},
		{Kind: ingest.StreamKindAudio, Index: 1, Language: "eng", ChannelLayout: ingest.LayoutStereo},
	}
	unit := newUnit(product, []string{"1.mp4"}, streams)

	got := PredictOutputs(product, unit, "/mnt/ROOT/EDIT", "")

	if len(got) != 2 {
		t.Fatalf("PredictOutputs() = %d outputs, want 2", len(got))
	}
	want := "/mnt/ROOT/EDIT/Show_s03/Show_s03_HD.mp4"
	if got[0].Path != want {
		t.Errorf("[0] Path = %q, want %q", got[0].Path, want)
	}
}

func TestPredictOutputs_episode_only(t *testing.T) {
	product := metadata.Product{
		ID:            1,
		Kind:          metadata.KindMovieEpisode,
		SeriesTitle:   "Show",
		EpisodeNumber: ptrInt(7),
	}
	streams := []ingest.Stream{
		{Kind: ingest.StreamKindVideo, Index: 0},
	}
	unit := newUnit(product, []string{"1.mp4"}, streams)

	got := PredictOutputs(product, unit, "/mnt/ROOT/EDIT", "")

	want := "/mnt/ROOT/EDIT/Show/Show_e07_HD.mp4"

	if got[0].Path != want {
		t.Errorf("[0] Path = %q, want %q", got[0].Path, want)
	}
}

// ── detectHSUB ──────────────────────────────────────────────────────

func TestDetectHSUB(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		tag      string
		want     string
	}{
		{"no hardsub", []string{"video.mp4", "sub.srt"}, "", ""},
		{"legacy hardsub tag", []string{"video.mp4", "12345-hardsub-eng.srt"}, "", "12345-hardsub-eng.srt"},
		{"custom tag Forsirovannye", []string{"video.mp4", "12254-rus_Russkii_Forsirovannye.srt"}, "Forsirovannye", "12254-rus_Russkii_Forsirovannye.srt"},
		{"multiple hardsub, first wins", []string{"a-hardsub.srt", "b-hardsub.srt"}, "", "a-hardsub.srt"},
		{"hardsub in middle", []string{"video.mp4", "a.srt", "12345-hardsub-eng.srt"}, "", "12345-hardsub-eng.srt"},
		{"case insensitive", []string{"video.mp4", "12345-HARDSUB-eng.srt"}, "", "12345-HARDSUB-eng.srt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectHSUB(tt.files, tt.tag)
			if got != tt.want {
				t.Errorf("detectHSUB(%v, %q) = %q, want %q", tt.files, tt.tag, got, tt.want)
			}
		})
	}
}

// ── layoutTag ───────────────────────────────────────────────────────

func TestLayoutTag(t *testing.T) {
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
		t.Run(string(tt.layout), func(t *testing.T) {
			got := layoutTag(tt.layout)
			if got != tt.want {
				t.Errorf("layoutTag(%q) = %q, want %q", tt.layout, got, tt.want)
			}
		})
	}
}
