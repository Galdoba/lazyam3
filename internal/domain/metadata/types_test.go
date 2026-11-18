package metadata

import (
	"testing"
)

func TestKindConstants(t *testing.T) {
	tests := []struct {
		name string
		kind Kind
		want string
	}{
		{"movie", KindMovie, "movie"},
		{"movie episode", KindMovieEpisode, "movie.episode"},
		{"series", KindSeries, "series"},
		{"season", KindSeason, "series.season"},
		{"series episode", KindSeriesEpisode, "series.season.episode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.kind) != tt.want {
				t.Errorf("Kind %q = %q, want %q", tt.name, string(tt.kind), tt.want)
			}
		})
	}
}

func TestProductZeroValue(t *testing.T) {
	p := Product{}
	if p.ID != 0 {
		t.Errorf("Product.ID = %d, want 0", p.ID)
	}
	if p.Kind != "" {
		t.Errorf("Product.Kind = %q, want empty", p.Kind)
	}
	if p.ParentID != nil {
		t.Errorf("Product.ParentID = %p, want nil", p.ParentID)
	}
	if p.SeasonNumber != nil {
		t.Errorf("Product.SeasonNumber = %p, want nil", p.SeasonNumber)
	}
	if p.EpisodeNumber != nil {
		t.Errorf("Product.EpisodeNumber = %p, want nil", p.EpisodeNumber)
	}
	if p.Genres != nil {
		t.Errorf("Product.Genres = %p, want nil", p.Genres)
	}
	if p.Countries != nil {
		t.Errorf("Product.Countries = %p, want nil", p.Countries)
	}
}

func TestProductConstruction(t *testing.T) {
	season := 1
	episode := 6
	parent := int64(12000)
	p := Product{
		ID:            12094,
		Kind:          KindSeriesEpisode,
		Title:         "Test Title",
		ParentID:      &parent,
		SeasonNumber:  &season,
		EpisodeNumber: &episode,
		SeriesTitle:   "Test Show",
		Genres:        []string{"Action"},
		Countries:     []string{"US"},
	}
	if p.ID != 12094 {
		t.Errorf("Product.ID = %d, want 12094", p.ID)
	}
	if p.Kind != KindSeriesEpisode {
		t.Errorf("Product.Kind = %q, want series.season.episode", p.Kind)
	}
	if p.ParentID == nil || *p.ParentID != 12000 {
		t.Errorf("Product.ParentID = %p, want 12000", p.ParentID)
	}
	if p.SeasonNumber == nil || *p.SeasonNumber != 1 {
		t.Errorf("Product.SeasonNumber = %p, want 1", p.SeasonNumber)
	}
	if p.EpisodeNumber == nil || *p.EpisodeNumber != 6 {
		t.Errorf("Product.EpisodeNumber = %p, want 6", p.EpisodeNumber)
	}
	if len(p.Genres) != 1 || p.Genres[0] != "Action" {
		t.Errorf("Product.Genres = %q, want [Action]", p.Genres)
	}
	if len(p.Countries) != 1 || p.Countries[0] != "US" {
		t.Errorf("Product.Countries = %q, want [US]", p.Countries)
	}
}
