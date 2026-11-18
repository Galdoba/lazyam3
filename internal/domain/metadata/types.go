package metadata

// Product is a flat metadata record (one shape for series, seasons, episodes).
type Product struct {
	ID            int64    `json:"id"`
	Kind          Kind     `json:"kind"`
	Title         string   `json:"title"`
	ParentID      *int64   `json:"parent_id,omitempty"`
	SeasonNumber  *int     `json:"season_number,omitempty"`
	EpisodeNumber *int     `json:"episode_number,omitempty"`
	SeasonEp      string   `json:"season_ep,omitempty"`
	SeriesTitle   string   `json:"series_title,omitempty"`
	Description   string   `json:"description,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	Countries     []string `json:"countries,omitempty"`
	KinopoiskID   string   `json:"kinopoisk_id,omitempty"`
	IMDBID        string   `json:"imdb_id,omitempty"`
}

// Kind classifies the product type.
type Kind string

const (
	KindMovie         Kind = "movie"
	KindMovieEpisode  Kind = "movie.episode"
	KindSeries        Kind = "series"
	KindSeason        Kind = "series.season"
	KindSeriesEpisode Kind = "series.season.episode"
)
