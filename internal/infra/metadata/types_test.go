package metadata

import (
	"testing"

	mdomain "github.com/Galdoba/lazyam3/internal/domain/metadata"
)

// buildTestRoot creates a minimal tree for testing.
func buildTestRoot() *Root {
	seriesID := int64(100)
	seasonID := int64(101)
	epID := int64(12061)
	movieID := int64(200)

	return &Root{
		Series: []SeriesItem{
			{
				Id:          seriesID,
				Title:       "White Queen",
				Genres:      []string{"drama", "history"},
				Countries:   []string{"UK"},
				KinopoiskId: "723644",
				ImdbId:      "tt2372220",
				Seasons: []SeasonsItem{
					{
						Id:           seasonID,
						SeasonNumber: 1,
						Title:        "Season 1",
						Episodes: []EpisodesItem{
							{Id: epID, Number: 1, Title: "Pilot", Description: "First episode"},
						},
					},
				},
			},
		},
		Movies: []MoviesItem{
			{
				Id:          movieID,
				Title:       "Movie Title",
				Genres:      []string{"action"},
				Countries:   []string{"US"},
				KinopoiskId: "123456",
				ImdbId:      "tt123456",
				Episodes: []EpisodesItem{
					{Id: 201, Number: 1, Title: "Part 1"},
				},
			},
		},
	}
}

func TestFindByID_Series(t *testing.T) {
	root := buildTestRoot()
	found, ok := root.FindByID(100, WithDescendants)
	if !ok {
		t.Fatal("FindByID(100) returned ok=false")
	}
	if found.Kind != mdomain.KindSeries {
		t.Errorf("Kind = %v, want %v", found.Kind, mdomain.KindSeries)
	}
	if len(found.Data.Series) != 1 {
		t.Fatalf("Series count = %d, want 1", len(found.Data.Series))
	}
	if found.Data.Series[0].Title != "White Queen" {
		t.Errorf("Title = %q, want %q", found.Data.Series[0].Title, "White Queen")
	}
}

func TestFindByID_Season(t *testing.T) {
	root := buildTestRoot()
	found, ok := root.FindByID(101, WithDescendants)
	if !ok {
		t.Fatal("FindByID(101) returned ok=false")
	}
	if found.Kind != mdomain.KindSeason {
		t.Errorf("Kind = %v, want %v", found.Kind, mdomain.KindSeason)
	}
	if len(found.Data.Series) != 1 || len(found.Data.Series[0].Seasons) != 1 {
		t.Fatalf("Season not found in pruned tree")
	}
}

func TestFindByID_Episode(t *testing.T) {
	root := buildTestRoot()
	found, ok := root.FindByID(12061, WithDescendants)
	if !ok {
		t.Fatal("FindByID(12061) returned ok=false")
	}
	if found.Kind != mdomain.KindSeriesEpisode {
		t.Errorf("Kind = %v, want %v", found.Kind, mdomain.KindSeriesEpisode)
	}
	if len(found.Data.Series) != 1 || len(found.Data.Series[0].Seasons) != 1 {
		t.Fatalf("Episode not found in pruned tree")
	}
	ep := found.Data.Series[0].Seasons[0].Episodes[0]
	if ep.Title != "Pilot" {
		t.Errorf("Episode title = %q, want %q", ep.Title, "Pilot")
	}
}

func TestFindByID_Movie(t *testing.T) {
	root := buildTestRoot()
	found, ok := root.FindByID(200, WithDescendants)
	if !ok {
		t.Fatal("FindByID(200) returned ok=false")
	}
	if found.Kind != mdomain.KindMovie {
		t.Errorf("Kind = %v, want %v", found.Kind, mdomain.KindMovie)
	}
	if len(found.Data.Movies) != 1 {
		t.Fatalf("Movies count = %d, want 1", len(found.Data.Movies))
	}
	if found.Data.Movies[0].Title != "Movie Title" {
		t.Errorf("Title = %q, want %q", found.Data.Movies[0].Title, "Movie Title")
	}
}

func TestFindByID_NotFound(t *testing.T) {
	root := buildTestRoot()
	_, ok := root.FindByID(99999, WithDescendants)
	if ok {
		t.Error("FindByID(99999) returned ok=true, want false")
	}
}

func TestFindByID_NilRoot(t *testing.T) {
	var root *Root
	found, ok := root.FindByID(100, WithDescendants)
	if ok {
		t.Error("FindByID on nil root returned ok=true")
	}
	if found != nil {
		t.Error("FindByID on nil root returned non-nil")
	}
}

func TestFoundToProduct_Series(t *testing.T) {
	root := buildTestRoot()
	found, _ := root.FindByID(100, WithDescendants)
	prod := FoundToProduct(found)

	if prod == nil {
		t.Fatal("FoundToProduct returned nil")
	}
	if prod.ID != 100 {
		t.Errorf("ID = %d, want 100", prod.ID)
	}
	if prod.Kind != "series" {
		t.Errorf("Kind = %v, want %v", prod.Kind, "series")
	}
	if prod.Title != "White Queen" {
		t.Errorf("Title = %q, want %q", prod.Title, "White Queen")
	}
	if prod.KinopoiskID != "723644" {
		t.Errorf("KinopoiskID = %q, want %q", prod.KinopoiskID, "723644")
	}
	if prod.IMDBID != "tt2372220" {
		t.Errorf("IMDBID = %q, want %q", prod.IMDBID, "tt2372220")
	}
}

func TestFoundToProduct_Season(t *testing.T) {
	root := buildTestRoot()
	found, _ := root.FindByID(101, WithDescendants)
	prod := FoundToProduct(found)

	if prod == nil {
		t.Fatal("FoundToProduct returned nil")
	}
	if prod.ID != 101 {
		t.Errorf("ID = %d, want 101", prod.ID)
	}
	if prod.Kind != "series.season" {
		t.Errorf("Kind = %v, want %v", prod.Kind, "series.season")
	}
	if prod.ParentID == nil || *prod.ParentID != 100 {
		t.Errorf("ParentID = %v, want 100", prod.ParentID)
	}
	if prod.SeasonNumber == nil || *prod.SeasonNumber != 1 {
		t.Errorf("SeasonNumber = %v, want 1", prod.SeasonNumber)
	}
}

func TestFoundToProduct_Episode(t *testing.T) {
	root := buildTestRoot()
	found, _ := root.FindByID(12061, WithDescendants)
	prod := FoundToProduct(found)

	if prod == nil {
		t.Fatal("FoundToProduct returned nil")
	}
	if prod.ID != 12061 {
		t.Errorf("ID = %d, want 12061", prod.ID)
	}
	if prod.Kind != "series.season.episode" {
		t.Errorf("Kind = %v, want %v", prod.Kind, "series.season.episode")
	}
	if prod.SeasonEp != "s01e01" {
		t.Errorf("SeasonEp = %q, want %q", prod.SeasonEp, "s01e01")
	}
	if prod.EpisodeNumber == nil || *prod.EpisodeNumber != 1 {
		t.Errorf("EpisodeNumber = %v, want 1", prod.EpisodeNumber)
	}
}

func TestFoundToProduct_Movie(t *testing.T) {
	root := buildTestRoot()
	found, _ := root.FindByID(200, WithDescendants)
	prod := FoundToProduct(found)

	if prod == nil {
		t.Fatal("FoundToProduct returned nil")
	}
	if prod.ID != 200 {
		t.Errorf("ID = %d, want 200", prod.ID)
	}
	if prod.Kind != "movie" {
		t.Errorf("Kind = %v, want %v", prod.Kind, "movie")
	}
	if prod.Genres[0] != "action" {
		t.Errorf("Genres[0] = %q, want %q", prod.Genres[0], "action")
	}
}

func TestFoundToProduct_Nil(t *testing.T) {
	prod := FoundToProduct(nil)
	if prod != nil {
		t.Error("FoundToProduct(nil) returned non-nil")
	}
}

func TestFoundToProduct_SeriesTitle(t *testing.T) {
	root := buildTestRoot()
	tests := []struct {
		name string
		id   int64
		want string
	}{
		{name: "series", id: 100, want: "White Queen"},
		{name: "season", id: 101, want: "White Queen"},
		{name: "series episode", id: 12061, want: "White Queen"},
		{name: "movie", id: 200, want: "Movie Title"},
		{name: "movie episode", id: 201, want: "Movie Title"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, ok := root.FindByID(tt.id, WithDescendants)
			if !ok {
				t.Fatalf("FindByID(%d) returned ok=false", tt.id)
			}
			prod := FoundToProduct(found)
			if prod == nil {
				t.Fatal("FoundToProduct returned nil")
			}
			if prod.SeriesTitle != tt.want {
				t.Errorf("SeriesTitle = %q, want %q", prod.SeriesTitle, tt.want)
			}
		})
	}
}

func TestFoundToProduct_EmptySeries(t *testing.T) {
	found := &Found{Kind: mdomain.KindSeries, Data: Root{Series: []SeriesItem{}}}
	prod := FoundToProduct(found)
	if prod != nil {
		t.Error("FoundToProduct(empty series) returned non-nil")
	}
}

func TestFindByID_Series_WithoutDescendants(t *testing.T) {
	root := buildTestRoot()
	found, ok := root.FindByID(100, WithoutDescendants)
	if !ok {
		t.Fatal("FindByID(100, WithoutDescendants) returned ok=false")
	}
	if len(found.Data.Series) != 1 {
		t.Fatalf("Series count = %d, want 1", len(found.Data.Series))
	}
	if len(found.Data.Series[0].Seasons) != 0 {
		t.Errorf("Seasons = %d, want 0 (pruned)", len(found.Data.Series[0].Seasons))
	}
}

func TestFindByID_Season_WithoutDescendants(t *testing.T) {
	root := buildTestRoot()
	found, ok := root.FindByID(101, WithoutDescendants)
	if !ok {
		t.Fatal("FindByID(101, WithoutDescendants) returned ok=false")
	}
	if len(found.Data.Series) != 1 || len(found.Data.Series[0].Seasons) != 1 {
		t.Fatalf("Season not found in pruned tree")
	}
	if len(found.Data.Series[0].Seasons[0].Episodes) != 0 {
		t.Errorf("Episodes = %d, want 0 (pruned)", len(found.Data.Series[0].Seasons[0].Episodes))
	}
}

func TestFindByID_Movie_WithAndWithoutDescendants(t *testing.T) {
	root := &Root{Movies: []MoviesItem{{
		Id:    200,
		Title: "Movie Title",
		Episodes: []EpisodesItem{
			{Id: 1, Number: 1, Title: "Part 1"},
			{Id: 2, Number: 2, Title: "Part 2"},
		},
	}}}

	found, ok := root.FindByID(200, WithDescendants)
	if !ok {
		t.Fatal("FindByID(200, WithDescendants) returned ok=false")
	}
	if got := len(found.Data.Movies[0].Episodes); got != 2 {
		t.Errorf("WithDescendants Episodes = %d, want 2", got)
	}

	found, ok = root.FindByID(200, WithoutDescendants)
	if !ok {
		t.Fatal("FindByID(200, WithoutDescendants) returned ok=false")
	}
	if got := len(found.Data.Movies[0].Episodes); got != 0 {
		t.Errorf("WithoutDescendants Episodes = %d, want 0", got)
	}
}

func TestFindByID_CloneIsIndependent(t *testing.T) {
	root := buildTestRoot()
	found, ok := root.FindByID(100, WithDescendants)
	if !ok {
		t.Fatal("FindByID(100, WithDescendants) returned ok=false")
	}

	// Mutating the pruned copy must not touch the original tree.
	found.Data.Series[0].Genres[0] = "mutated"
	found.Data.Series[0].Seasons[0].Episodes[0].Title = "mutated"

	if root.Series[0].Genres[0] != "drama" {
		t.Error("original Series.Genres was mutated by the pruned copy")
	}
	if root.Series[0].Seasons[0].Episodes[0].Title != "Pilot" {
		t.Error("original Series episode was mutated by the pruned copy")
	}
}
