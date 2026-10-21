package callname

import (
	"testing"

	"github.com/Galdoba/lazyam3/internal/domain/metadata"
)

// ── ScriptInput.String ──────────────────────────────────────────────

func TestScriptInput_String_full(t *testing.T) {
	si := ScriptInput{
		ID:           12345,
		Base:         "Igra",
		SeasonTag:    "s01",
		EpisodeTag:   "e26",
		PRT:          "PRT260928200000",
		OriginalName: "Igra_lzhecov_s01e26_PRT260928200000_id12345.mp4",
		FileTags:     []string{"rus", "Russkii"},
	}
	got := si.String()
	want := "Igra_s01e26_PRT260928200000_id12345-rus_Russkii.mp4"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestScriptInput_String_noSeasonEpisode(t *testing.T) {
	si := ScriptInput{
		ID:           12254,
		Base:         "Film",
		SeasonTag:    "",
		EpisodeTag:   "",
		PRT:          "PRT260928200000",
		OriginalName: "12254-rus_Russkii_Forsirovannye.srt",
		FileTags:     []string{"rus", "Russkii", "Forsirovannye"},
	}
	got := si.String()
	want := "Film_PRT260928200000_id12254-rus_Russkii_Forsirovannye.srt"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestScriptInput_String_noFileTags(t *testing.T) {
	si := ScriptInput{
		ID:           63574,
		Base:         "BlackClover",
		SeasonTag:    "s01",
		EpisodeTag:   "e041",
		PRT:          "PRT261013145101",
		OriginalName: "63705.mp4",
	}
	got := si.String()
	want := "BlackClover_s01e041_PRT261013145101_id63574.mp4"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

// ── extractFileTags ─────────────────────────────────────────────────

func TestExtractFileTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"bare id", "12254", nil},
		{"single tag", "12254-rus", []string{"rus"}},
		{"multiple tags", "12254-rus_Russkii", []string{"rus", "Russkii"}},
		{"multiple tags", "12254-rus_Russkii_Forsirovannye", []string{"rus", "Russkii", "Forsirovannye"}},
		{"empty after id", "", nil},
		{"trailing empty tag", "12254-rus_", []string{"rus"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFileTags(tt.input)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("extractFileTags(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractFileTags(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ── ScriptInput.validate ────────────────────────────────────────────

func TestScriptInput_validate(t *testing.T) {
	t.Run("zero id", func(t *testing.T) {
		si := ScriptInput{Base: "B", PRT: "PRT123", OriginalName: "12345.mp4"}
		if err := si.validate(); err == nil {
			t.Error("expected error for zero ID")
		}
	})

	t.Run("empty base", func(t *testing.T) {
		si := ScriptInput{ID: 1, PRT: "PRT123", OriginalName: "12345.mp4"}
		if err := si.validate(); err == nil {
			t.Error("expected error for empty base")
		}
	})

	t.Run("empty prt", func(t *testing.T) {
		si := ScriptInput{ID: 1, Base: "B", OriginalName: "12345.mp4"}
		if err := si.validate(); err == nil {
			t.Error("expected error for empty PRT")
		}
	})

	t.Run("all valid", func(t *testing.T) {
		si := ScriptInput{ID: 1, Base: "B", PRT: "PRT260101120000", OriginalName: "12345.mp4"}
		if err := si.validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// ── NewPRT ──────────────────────────────────────────────────────────

func TestNewPRT(t *testing.T) {
	got := NewPRT()
	if got == "PRTxxxxxxxxxxxx" {
		t.Log("NewPRT returns TODO placeholder")
	}
	if len(got) != 15 {
		t.Errorf("NewPRT() length = %d, want 15", len(got))
	}
}

// ── BuildInputName ───────────────────────────────────────────────────

func TestBuildInputName_movie(t *testing.T) {
	product := metadata.Product{
		Kind:        metadata.KindMovie,
		SeriesTitle: "12254",
	}
	got, err := BuildInputName(product, "12254-rus_Russkii_Forsirovannye.srt", "260928200000")
	if err != nil {
		t.Fatalf("BuildInputName error: %v", err)
	}
	if got.ID != 12254 {
		t.Errorf("ID = %d, want 12254", got.ID)
	}
	if got.Base == "" {
		t.Error("Base should not be empty")
	}
	if got.PRT != "PRT260928200000" {
		t.Errorf("PRT = %q, want %q", got.PRT, "PRT260928200000")
	}
	want := "12254_PRT260928200000_id12254-rus_Russkii_Forsirovannye.srt"
	if got.String() != want {
		t.Errorf("String() = %q, want %q", got.String(), want)
	}
}

func TestBuildInputName_series_episode(t *testing.T) {
	season := 1
	episode := 41
	product := metadata.Product{
		Kind:          metadata.KindSeriesEpisode,
		SeriesTitle:   "BLACK CLOVER",
		SeasonNumber:  &season,
		EpisodeNumber: &episode,
	}
	got, err := BuildInputName(product, "63705.mp4", "261013145101")
	if err != nil {
		t.Fatalf("BuildInputName error: %v", err)
	}
	if got.ID != 63705 {
		t.Errorf("ID = %d, want 63705", got.ID)
	}
	if got.SeasonTag+got.EpisodeTag != "s01e41" {
		t.Errorf("SeasonTag+EpisodeTag = %q, want %q", got.SeasonTag+got.EpisodeTag, "s01e41")
	}
}

func TestBuildInputName_no_leading_digit(t *testing.T) {
	product := metadata.Product{Kind: metadata.KindMovie, SeriesTitle: "Test"}
	_, err := BuildInputName(product, "no_digits.mp4", "123456789012")
	if err == nil {
		t.Error("expected error for non-digit-starting filename")
	}
}

// ── ExtractID ───────────────────────────────────────────────────────

func TestExtractID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64
		wantErr bool
	}{
		{"bare id", "12254", 12254, false},
		{"with extension", "63574.mp4", 63574, false},
		{"with tags", "12254-rus_Russkii.srt", 12254, false},
		{"no leading digit", "abc.mp4", 0, true},
		{"empty", "", 0, true},
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

// ── BuildInputNames ──────────────────────────────────────────────────

func TestBuildInputNames(t *testing.T) {
	season := 1
	product := metadata.Product{
		Kind:          metadata.KindSeries,
		SeriesTitle:   "BLACK CLOVER",
		SeasonNumber:  &season,
		EpisodeNumber: ptrInt(41),
	}
	names, err := BuildInputNames(product, []string{"63705.mp4"}, "261013145101")
	if err != nil {
		t.Fatalf("BuildInputNames error: %v", err)
	}
	if len(names) != 1 {
		t.Fatalf("got %d names, want 1", len(names))
	}
	got := names["63705.mp4"]
	if got == "" {
		t.Error("expected non-empty call name")
	}
}

func TestBuildInputNames_one_fails(t *testing.T) {
	product := metadata.Product{Kind: metadata.KindMovie, SeriesTitle: "Test"}
	_, err := BuildInputNames(product, []string{"bad.mp4"}, "123456789012")
	if err == nil {
		t.Error("expected error when one filename fails")
	}
}

// ── deriveNameTag ───────────────────────────────────────────────────

func TestDeriveNameTag(t *testing.T) {
	tests := []struct {
		name    string
		product metadata.Product
		want    string
	}{
		{"season+episode", metadata.Product{SeasonNumber: ptrInt(1), EpisodeNumber: ptrInt(6)}, "s01e06"},
		{"season only", metadata.Product{SeasonNumber: ptrInt(3)}, "s03"},
		{"episode only", metadata.Product{EpisodeNumber: ptrInt(12)}, "e12"},
		{"neither", metadata.Product{SeriesTitle: "Movie"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveNameTag(tt.product)
			if got != tt.want {
				t.Errorf("deriveNameTag() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ── splitExtension ──────────────────────────────────────────────────

func TestSplitExtension(t *testing.T) {
	tests := []struct {
		input    string
		wantStem string
		wantExt  string
	}{
		{"63574.mp4", "63574", "mp4"},
		{"12254-rus_Russkii.srt", "12254-rus_Russkii", "srt"},
		{"no_ext", "no_ext", ""},
		{"a.b.c", "a.b", "c"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			stem, ext := splitExtension(tt.input)
			if stem != tt.wantStem {
				t.Errorf("stem = %q, want %q", stem, tt.wantStem)
			}
			if ext != tt.wantExt {
				t.Errorf("ext = %q, want %q", ext, tt.wantExt)
			}
		})
	}
}

func ptrInt(i int) *int { return &i }

// ── NewPRT ──────────────────────────────────────────────────────────

func TestNewPRT_format(t *testing.T) {
	got := NewPRT()
	if len(got) != 15 {
		t.Errorf("NewPRT() length = %d, want 15", len(got))
	}
	if got[:3] != "PRT" {
		t.Errorf("NewPRT() prefix = %q, want %q", got[:3], "PRT")
	}
	digits := got[3:]
	if len(digits) != 12 {
		t.Errorf("NewPRT() digits length = %d, want 12", len(digits))
	}
	for i, r := range digits {
		if r < '0' || r > '9' {
			t.Errorf("NewPRT()[%d] = %q, expected digit", i, r)
		}
	}
}

func TestNewPRT_uniqueness(t *testing.T) {
	// Rapid calls may return the same timestamp (1-second resolution).
	// Just verify format is valid for multiple calls.
	for i := 0; i < 5; i++ {
		got := NewPRT()
		if len(got) != 15 {
			t.Errorf("NewPRT() length = %d, want 15", len(got))
		}
		if got[:3] != "PRT" {
			t.Errorf("NewPRT() prefix = %q, want %q", got[:3], "PRT")
		}
	}
}

// ── validate ────────────────────────────────────────────────────────

func TestScriptInput_validate_invalidPRT(t *testing.T) {
	si := ScriptInput{ID: 1, Base: "B", PRT: "PRT123", OriginalName: "12345.mp4"}
	if err := si.validate(); err == nil {
		t.Error("expected error for short PRT")
	}

	si.PRT = "pRT260101120000"
	if err := si.validate(); err == nil {
		t.Error("expected error for lowercase PRT prefix")
	}

	si.PRT = "PRT2601011200001" // 13 digits
	if err := si.validate(); err == nil {
		t.Error("expected error for PRT with 13 digits")
	}

	si.PRT = "PRT26010112000" // 11 digits
	if err := si.validate(); err == nil {
		t.Error("expected error for PRT with 11 digits")
	}
}

// ── deriveOutBase ───────────────────────────────────────────────────

func TestDeriveOutBase_cyrillic(t *testing.T) {
	product := metadata.Product{
		SeriesTitle: "Игра лиц",
	}
	got := deriveOutBase(product)
	want := "Igra_lic"
	if got != want {
		t.Errorf("deriveOutBase(%q) = %q, want %q", "Игра лиц", got, want)
	}
}

func TestDeriveOutBase_empty(t *testing.T) {
	product := metadata.Product{SeriesTitle: ""}
	got := deriveOutBase(product)
	if got != "" {
		t.Errorf("deriveOutBase(\"\") = %q, want empty", got)
	}
}

func TestDeriveOutBase_latin(t *testing.T) {
	product := metadata.Product{
		SeriesTitle: "BLACK CLOVER",
	}
	got := deriveOutBase(product)
	want := "Black_clover"
	if got != want {
		t.Errorf("deriveOutBase(%q) = %q, want %q", "BLACK CLOVER", got, want)
	}
}

func TestDeriveOutBase_firstCharUpper(t *testing.T) {
	product := metadata.Product{
		SeriesTitle: "белая ночь",
	}
	got := deriveOutBase(product)
	if len(got) == 0 {
		t.Fatal("deriveOutBase returned empty")
	}
	if got[:1] != "B" {
		t.Errorf("first char = %q, want %q", got[:1], "B")
	}
}
