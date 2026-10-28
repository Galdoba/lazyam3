package metadata

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Galdoba/lazyam3/internal/domain/metadata"
)

func TestRepository_FindByID(t *testing.T) {
	// Create a temporary metadata.json
	tmpDir := t.TempDir()
	metaFile := filepath.Join(tmpDir, "metadata.json")

	data := `{
		"series": [
			{
				"id": 100,
				"title": "Test Series",
				"genres": ["drama"],
				"countries": ["US"],
				"kinopoiskId": "999",
				"imdb_id": "tt999",
				"seasons": [
					{
						"id": 101,
						"type": "season",
						"seasonNumber": 1,
						"title": "Season 1",
						"episodes": [
							{"id": 12001, "number": 1, "title": "Pilot", "description": "First"}
						]
					}
				]
			}
		],
		"movies": [
			{
				"id": 200,
				"title": "Test Movie",
				"genres": ["action"],
				"countries": ["US"],
				"kinopoiskId": "888",
				"imdbId": "tt888",
				"episodes": []
			}
		]
	}`
	if err := os.WriteFile(metaFile, []byte(data), 0o644); err != nil {
		t.Fatalf("failed to write test metadata: %v", err)
	}

	repo, err := New(nil, metaFile)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	tests := []struct {
		name      string
		id        int64
		wantKind  metadata.Kind
		wantTitle string
		wantErr   bool
	}{
		{
			name:      "series",
			id:        100,
			wantKind:  metadata.KindSeries,
			wantTitle: "Test Series",
		},
		{
			name:      "season",
			id:        101,
			wantKind:  metadata.KindSeason,
			wantTitle: "Season 1",
		},
		{
			name:      "episode",
			id:        12001,
			wantKind:  metadata.KindSeriesEpisode,
			wantTitle: "Pilot",
		},
		{
			name:      "movie",
			id:        200,
			wantKind:  metadata.KindMovie,
			wantTitle: "Test Movie",
		},
		{
			name:    "not found",
			id:      99999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prod, err := repo.FindByID(tt.id)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("FindByID(%d) error: %v", tt.id, err)
			}
			if prod == nil {
				t.Fatal("expected product, got nil")
			}
			if prod.Kind != tt.wantKind {
				t.Errorf("Kind = %v, want %v", prod.Kind, tt.wantKind)
			}
			if prod.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", prod.Title, tt.wantTitle)
			}
		})
	}
}

func TestRepository_New_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	metaFile := filepath.Join(tmpDir, "metadata.json")

	data := `{"series": [], "movies": []}`
	if err := os.WriteFile(metaFile, []byte(data), 0o644); err != nil {
		t.Fatalf("failed to write test metadata: %v", err)
	}

	_, err := New(nil, metaFile)
	if err == nil {
		t.Error("expected error for empty metadata, got nil")
	}
}

func TestRepository_New_NotFound(t *testing.T) {
	_, err := New(nil, "/nonexistent/path/metadata.json")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestRepository_FindByID_RealFile(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "metadata.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testdata/metadata.json not available")
	}

	repo, err := New(nil, path)
	if err != nil {
		t.Fatalf("New(%q) error: %v", path, err)
	}

	parent11120 := int64(11120)
	parent11393 := int64(11393)
	parent9261 := int64(9261)

	tests := []struct {
		name            string
		id              int64
		wantKind        metadata.Kind
		wantTitle       string
		wantSeriesTitle string
		wantParent      *int64
		wantSeasonEp    string
		wantErr         bool
	}{
		{name: "series", id: 11120, wantKind: metadata.KindSeries, wantTitle: "Белая королева", wantSeriesTitle: "Белая королева"},
		{name: "season", id: 11393, wantKind: metadata.KindSeason, wantTitle: "Сезон 1", wantSeriesTitle: "Белая королева", wantParent: &parent11120},
		{name: "series episode", id: 12061, wantKind: metadata.KindSeriesEpisode, wantTitle: "Любовь короля", wantSeriesTitle: "Белая королева", wantParent: &parent11393, wantSeasonEp: "s01e01"},
		{name: "movie", id: 9257, wantKind: metadata.KindMovie, wantTitle: "Поймать Мэри", wantSeriesTitle: "Поймать Мэри"},
		{name: "movie episode", id: 10100, wantKind: metadata.KindMovieEpisode, wantTitle: "Фил Спектор", wantSeriesTitle: "Фил Спектор", wantParent: &parent9261},
		{name: "not found", id: 999999999, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prod, err := repo.FindByID(tt.id)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("FindByID(%d) error: %v", tt.id, err)
			}
			if prod.Kind != tt.wantKind {
				t.Errorf("Kind = %v, want %v", prod.Kind, tt.wantKind)
			}
			if prod.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", prod.Title, tt.wantTitle)
			}
			if prod.SeriesTitle != tt.wantSeriesTitle {
				t.Errorf("SeriesTitle = %q, want %q", prod.SeriesTitle, tt.wantSeriesTitle)
			}
			if tt.wantParent != nil {
				if prod.ParentID == nil {
					t.Errorf("ParentID = nil, want %d", *tt.wantParent)
				} else if *prod.ParentID != *tt.wantParent {
					t.Errorf("ParentID = %d, want %d", *prod.ParentID, *tt.wantParent)
				}
			}
			if tt.wantSeasonEp != "" && prod.SeasonEp != tt.wantSeasonEp {
				t.Errorf("SeasonEp = %q, want %q", prod.SeasonEp, tt.wantSeasonEp)
			}
		})
	}
}
