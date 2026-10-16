package metadata

import (
	"time"
)

type Root struct {
	Series []SeriesItem `json:"series"`
	Movies []MoviesItem `json:"movies"`
}

type MoviesItem struct {
	Id              int64          `json:"id"`
	OriginalTitle   string         `json:"originalTitle"`
	Title           string         `json:"title"`
	PremierYear     int64          `json:"premierYear"`
	Description     string         `json:"description"`
	MainGenre       string         `json:"mainGenre"`
	AgeRestrictions *int64         `json:"ageRestrictions"`
	KinopoiskId     string         `json:"kinopoiskId"`
	ImdbId          string         `json:"imdbId"`
	PagePublished   bool           `json:"pagePublished"`
	Quote           string         `json:"quote"`
	ReleaseYear     *int64         `json:"releaseYear"`
	Countries       []string       `json:"countries"`
	Studios         []string       `json:"studios"`
	Genres          []string       `json:"genres"`
	Persons         []PersonsItem  `json:"persons"`
	Episodes        []EpisodesItem `json:"episodes"`
}

type EpisodesItem struct {
	Id            int64  `json:"id"`
	Number        int64  `json:"number"`
	Title         string `json:"title"`
	OriginalTitle string `json:"originalTitle"`
	PremierYear   *int64 `json:"premierYear"`
	Description   string `json:"description"`
	PagePublished bool   `json:"pagePublished"`
	ReleaseYear   *int64 `json:"releaseYear"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
}

type PersonsItem struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type SeriesItem struct {
	Id              int64         `json:"id"`
	OriginalTitle   string        `json:"originalTitle"`
	Title           string        `json:"title"`
	StartDate       time.Time     `json:"start_date"` //time.Time? 2021-07-15T00:00:00+03:00
	EndDate         string        `json:"end_date,omitempty"`
	MainGenre       string        `json:"mainGenre"`
	AgeRestrictions *int64        `json:"ageRestrictions,omitempty"`
	Description     string        `json:"description"`
	ImdbId          string        `json:"imdb_id"`
	KinopoiskId     string        `json:"kinopoiskId"`
	Countries       []string      `json:"countries"`
	Quote           string        `json:"quote"`
	Studios         []string      `json:"studios"`
	Genres          []string      `json:"genres"`
	Persons         []PersonsItem `json:"persons"`
	Seasons         []SeasonsItem `json:"seasons"`
}

type SeasonsItem struct {
	Id            int64          `json:"id"`
	Type          string         `json:"type"`
	SeasonNumber  int64          `json:"seasonNumber"`
	Title         string         `json:"title"`
	OriginalTitle string         `json:"originalTitle"`
	Description   string         `json:"description"`
	Persons       []PersonsItem  `json:"persons"`
	Episodes      []EpisodesItem `json:"episodes"`
}

type Kind string

const (
	KindMovie         Kind = "movie"
	KindMovieEpisode  Kind = "movie.episode"
	KindSeries        Kind = "series"
	KindSeason        Kind = "series.season"
	KindSeriesEpisode Kind = "series.season.episode"
)

type Descendants bool

const (
	WithDescendants    Descendants = true  // цель + всё поддерево под ней
	WithoutDescendants Descendants = false // только сам объект, без детей
)

// Found — полностью самостоятельная копия: не ссылается на исходный Root,
// безопасна для передачи, кэширования и модификации.
type Found struct {
	Kind Kind
	Data Root // обрезанное дерево; цель лежит внутри на своём месте
}
