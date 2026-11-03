package metadata

import "testing"

func TestKind_Predicates(t *testing.T) {
	cases := []struct {
		name      string
		kind      Kind
		isMovie   bool
		isSeries  bool
		isSeason  bool
		isEpisode bool
		isMe      bool
		isSeEp    bool
	}{
		{"movie", KindMovie, true, false, false, false, false, false},
		{"movie episode", KindMovieEpisode, false, false, false, true, true, false},
		{"series", KindSeries, false, true, false, false, false, false},
		{"season", KindSeason, false, false, true, false, false, false},
		{"series episode", KindSeriesEpisode, false, false, false, true, false, true},
		{"unknown", Kind("unknown"), false, false, false, false, false, false},
	}
	for _, c := range cases {
		if c.kind.IsMovie() != c.isMovie {
			t.Errorf("%s: IsMovie()=%v, want %v", c.name, c.kind.IsMovie(), c.isMovie)
		}
		if c.kind.IsSeries() != c.isSeries {
			t.Errorf("%s: IsSeries()=%v, want %v", c.name, c.kind.IsSeries(), c.isSeries)
		}
		if c.kind.IsSeason() != c.isSeason {
			t.Errorf("%s: IsSeason()=%v, want %v", c.name, c.kind.IsSeason(), c.isSeason)
		}
		if c.kind.IsEpisode() != c.isEpisode {
			t.Errorf("%s: IsEpisode()=%v, want %v", c.name, c.kind.IsEpisode(), c.isEpisode)
		}
		if c.kind.IsMovieEpisode() != c.isMe {
			t.Errorf("%s: IsMovieEpisode()=%v, want %v", c.name, c.kind.IsMovieEpisode(), c.isMe)
		}
		if c.kind.IsSeriesEpisode() != c.isSeEp {
			t.Errorf("%s: IsSeriesEpisode()=%v, want %v", c.name, c.kind.IsSeriesEpisode(), c.isSeEp)
		}
	}
}
