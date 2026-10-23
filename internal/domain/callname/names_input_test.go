package callname

import (
	"strings"
	"testing"

	"github.com/Galdoba/lazyam3/internal/domain/metadata"
)

func TestScriptInput_String(t *testing.T) {
	tests := []struct {
		name     string
		input    ScriptInput
		expected string
	}{
		{
			name: "full with season/episode and file tags",
			input: ScriptInput{
				ID:           12345,
				Base:         "Igra_lzhecov",
				SeasonTag:    "s01",
				EpisodeTag:   "e26",
				PRT:          "PRT260928200000",
				OriginalName: "12345.mp4",
				FileTags:     []string{"rus", "Russkii", "Forsirovannye"},
			},
			expected: "Igra_lzhecov_s01e26_PRT260928200000_id12345-rus_Russkii_Forsirovannye.mp4",
		},
		{
			name: "no season/episode, no file tags",
			input: ScriptInput{
				ID:           999,
				Base:         "Movie",
				PRT:          "PRT261001120000",
				OriginalName: "999.mkv",
			},
			expected: "Movie_PRT261001120000_id999.mkv",
		},
		{
			name: "with file tags only",
			input: ScriptInput{
				ID:           100,
				Base:         "Show",
				SeasonTag:    "s02",
				EpisodeTag:   "e05",
				PRT:          "PRT261015080000",
				OriginalName: "100-video.mp4",
				FileTags:     []string{"eng", "original"},
			},
			expected: "Show_s02e05_PRT261015080000_id100-eng_original.mp4",
		},
		{
			name: "only season tag, no episode",
			input: ScriptInput{
				ID:        50,
				Base:      "Series",
				SeasonTag: "s03",
				PRT:       "PRT261020000000",
			},
			expected: "Series_s03_PRT261020000000_id50",
		},
		{
			name: "only episode tag, no season",
			input: ScriptInput{
				ID:        50,
				Base:      "Series",
				EpisodeTag: "e10",
				PRT:       "PRT261020000000",
			},
			expected: "Series_e10_PRT261020000000_id50",
		},
		{
			name:     "empty base with season/episode",
			input:    ScriptInput{},
			expected: "__id0",
		},
		{
			name:     "empty base produces flat output",
			input:    ScriptInput{ID: 12345, PRT: "PRT260928200000", OriginalName: "12345.mp4"},
			expected: "_PRT260928200000_id12345.mp4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.String()
			if got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func Test_extractFileTags(t *testing.T) {
	tests := []struct {
		name     string
		stem     string
		expected []string
	}{
		{
			name:     "basic tags after ID",
			stem:     "12345-rus_Russkii",
			expected: []string{"rus", "Russkii"},
		},
		{
			name:     "single tag",
			stem:     "100-eng",
			expected: []string{"eng"},
		},
		{
			name:     "no tags",
			stem:     "12345",
			expected: nil,
		},
		{
			name:     "tags with empty trailing",
			stem:     "100-eng_original_",
			expected: []string{"eng", "original"},
		},
		{
			name:     "no leading digits",
			stem:     "abc123-rus",
			expected: []string{"abc123-rus"},
		},
		{
			name:     "empty stem",
			stem:     "",
			expected: nil,
		},
		{
			name:     "dash only after digits",
			stem:     "42-",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFileTags(tt.stem)
			if len(got) != len(tt.expected) {
				t.Errorf("extractFileTags(%q) = %v (len=%d), want %v (len=%d)",
					tt.stem, got, len(got), tt.expected, len(tt.expected))
			} else {
				for i := range got {
					if got[i] != tt.expected[i] {
						t.Errorf("extractFileTags(%q)[%d] = %q, want %q",
							tt.stem, i, got[i], tt.expected[i])
					}
				}
			}
		})
	}
}

func TestScriptInput_validate(t *testing.T) {
	tests := []struct {
		name    string
		input   ScriptInput
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid input",
			input:   ScriptInput{ID: 1, Base: "Show", PRT: "PRT260928200000"},
			wantErr: false,
		},
		{
			name:    "missing ID",
			input:   ScriptInput{Base: "Show", PRT: "PRT260928200000"},
			wantErr: true,
			errMsg:  "id must be set",
		},
		{
			name:    "missing base",
			input:   ScriptInput{ID: 1, PRT: "PRT260928200000"},
			wantErr: true,
			errMsg:  "base must be set",
		},
		{
			name:    "missing PRT",
			input:   ScriptInput{ID: 1, Base: "Show"},
			wantErr: true,
			errMsg:  "PRT must be set",
		},
		{
			name:    "invalid PRT format",
			input:   ScriptInput{ID: 1, Base: "Show", PRT: "BAD"},
			wantErr: true,
			errMsg:  "PRT invalid",
		},
		{
			name:    "PRT too short",
			input:   ScriptInput{ID: 1, Base: "Show", PRT: "PRT123"},
			wantErr: true,
			errMsg:  "PRT invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("validate() returned nil, want error")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validate() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBuildInputName(t *testing.T) {
	product := metadata.Product{
		ID:          12345,
		Kind:        metadata.KindSeries,
		Title:       "Test Episode",
		SeriesTitle: "Игра лицевых",
		SeasonNumber: intPtr(1),
		EpisodeNumber: intPtr(26),
	}

	tests := []struct {
		name        string
		product     metadata.Product
		filename    string
		timestamp   string
		wantErr     bool
		validate    func(ScriptInput) bool
	}{
		{
			name:      "basic video file",
			product:   product,
			filename:  "12345.mp4",
			timestamp: "260928200000",
			wantErr:   false,
			validate: func(si ScriptInput) bool {
				return si.ID == 12345 &&
					si.SeasonTag == "s01e26" &&
					si.PRT == "PRT260928200000"
			},
		},
		{
			name:      "filename without leading digits",
			product:   product,
			filename:  "no_digits.mp4",
			timestamp: "260928200000",
			wantErr:   true,
		},
		{
			name:      "with file tags",
			product:   product,
			filename:  "12345-rus_Russkii.mp4",
			timestamp: "260928200000",
			wantErr:   false,
			validate: func(si ScriptInput) bool {
				return len(si.FileTags) == 2 && si.FileTags[0] == "rus" && si.FileTags[1] == "Russkii"
			},
		},
		{
			name:      "movie product no season/episode",
			product:   metadata.Product{ID: 999, Kind: metadata.KindMovie, SeriesTitle: "Movie Title"},
			filename:  "999.mp4",
			timestamp: "261001120000",
			wantErr:   false,
			validate: func(si ScriptInput) bool {
				return si.SeasonTag == "" && si.EpisodeTag == "" && si.ID == 999
			},
		},
		{
			name:      "empty series title is error",
			product:   metadata.Product{ID: 100, SeriesTitle: ""},
			filename:  "100.mp4",
			timestamp: "261001120000",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildInputName(tt.product, tt.filename, tt.timestamp)
			if tt.wantErr {
				if err == nil {
					t.Errorf("BuildInputName() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("BuildInputName() unexpected error: %v", err)
				return
			}
			if tt.validate != nil && !tt.validate(got) {
				t.Errorf("BuildInputName() validation failed for result: %+v", got)
			}
		})
	}
}

func TestBuildInputNames(t *testing.T) {
	product := metadata.Product{
		ID:          12345,
		Kind:        metadata.KindSeries,
		SeriesTitle: "Show",
		SeasonNumber: intPtr(1),
		EpisodeNumber: intPtr(1),
	}

	inputs := []string{"12345.mp4", "12345-rus_Russkii.srt"}
	names, err := BuildInputNames(product, inputs, "261001120000")
	if err != nil {
		t.Fatalf("BuildInputNames() error: %v", err)
	}

	if len(names) != 2 {
		t.Fatalf("BuildInputNames() returned %d names, want 2", len(names))
	}

	// Check video
	videoName := names["12345.mp4"]
	if videoName == "" {
		t.Error("Expected video name, got empty")
	}
	if !strings.HasSuffix(videoName, ".mp4") {
		t.Errorf("Expected video to end with .mp4, got %q", videoName)
	}

	// Check subtitle
	subName := names["12345-rus_Russkii.srt"]
	if subName == "" {
		t.Error("Expected subtitle name, got empty")
	}
	if !strings.HasSuffix(subName, ".srt") {
		t.Errorf("Expected subtitle to end with .srt, got %q", subName)
	}
}

func TestExtractID(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantID    int64
		wantErr   bool
	}{
		{"basic", "12345.mp4", 12345, false},
		{"with extension", "99999.mkv", 99999, false},
		{"no leading digit", "abc.mp4", 0, true},
		{"empty", "", 0, true},
		{"only extension", ".mp4", 0, true},
		{"large id", "9999999999999.mp4", 9999999999999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractID(tt.filename)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ExtractID(%q) expected error, got %d", tt.filename, got)
				}
				return
			}
			if err != nil {
				t.Errorf("ExtractID(%q) unexpected error: %v", tt.filename, err)
				return
			}
			if got != tt.wantID {
				t.Errorf("ExtractID(%q) = %d, want %d", tt.filename, got, tt.wantID)
			}
		})
	}
}

func TestDeriveOutBase(t *testing.T) {
	tests := []struct {
		name     string
		product  metadata.Product
		expected string
	}{
		{
			name:    "cyrillic series title",
			product: metadata.Product{SeriesTitle: "Игра лицевых"},
			expected: "Igra_licevyh",
		},
		{
			name:    "latin series title",
			product: metadata.Product{SeriesTitle: "Test Show"},
			expected: "Test_show",
		},
		{
			name:    "empty series title",
			product: metadata.Product{},
			expected: "",
		},
		{
			name:    "single word cyrillic",
			product: metadata.Product{SeriesTitle: "Кино"},
			expected: "Kino",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveOutBase(tt.product)
			if got != tt.expected {
				t.Errorf("deriveOutBase(%q) = %q, want %q", tt.product.SeriesTitle, got, tt.expected)
			}
		})
	}
}

func TestDeriveNameTag(t *testing.T) {
	tests := []struct {
		name     string
		product  metadata.Product
		expected string
	}{
		{
			name:     "full series episode",
			product:  metadata.Product{SeasonNumber: intPtr(1), EpisodeNumber: intPtr(6)},
			expected: "s01e06",
		},
		{
			name:     "season only",
			product:  metadata.Product{SeasonNumber: intPtr(3)},
			expected: "s03",
		},
		{
			name:     "episode only",
			product:  metadata.Product{EpisodeNumber: intPtr(10)},
			expected: "e10",
		},
		{
			name:     "no season or episode (movie)",
			product:  metadata.Product{Kind: metadata.KindMovie},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveNameTag(tt.product)
			if got != tt.expected {
				t.Errorf("deriveNameTag() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func intPtr(i int) *int { return &i }
