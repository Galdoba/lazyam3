package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	mdomain "github.com/Galdoba/lazyam3/internal/domain/metadata"
)

// ---------------------------------------------------------------------------
// JSON types (storage format, infrastructure concern)
// ---------------------------------------------------------------------------

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
	StartDate       time.Time     `json:"start_date"`
	EndDate         string        `json:"end_date,omitempty"`
	MainGenre       string        `json:"mainGenre"`
	AgeRestrictions *int64        `json:"ageRestrictions,omitempty"`
	Description     string        `json:"description"`
	ImdbId          string `json:"imdbId"`
	KinopoiskId     string `json:"kinopoiskId"`
	Countries       []string `json:"countries"`
	Quote           string `json:"quote"`
	Studios         []string `json:"studios"`
	Genres          []string `json:"genres"`
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

type Descendants bool

const (
	WithDescendants    Descendants = true
	WithoutDescendants Descendants = false
)

// Found is a pruned tree result: the target sits inside at its proper place.
type Found struct {
	Kind mdomain.Kind
	Data Root
}

// ---------------------------------------------------------------------------
// Load + lookup
// ---------------------------------------------------------------------------

// LoadRoot reads and parses a metadata JSON file.
func LoadRoot(path string) (*Root, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read metadata file: %w", err)
	}
	var root Root
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse metadata JSON: %w", err)
	}
	if len(root.Series) == 0 && len(root.Movies) == 0 {
		return nil, fmt.Errorf("empty metadata: no series or movies found")
	}
	return &root, nil
}

// FindByID searches the tree for the element with the given ID.
func (r *Root) FindByID(id int64, d Descendants) (*Found, bool) {
	if r == nil {
		return nil, false
	}

	// --- Movies ---
	for i := range r.Movies {
		m := &r.Movies[i]
		if m.Id == id {
			out := cloneMovie(m, d)
			return &Found{Kind: mdomain.KindMovie, Data: Root{Movies: []MoviesItem{out}}}, true
		}
		for j := range m.Episodes {
			if m.Episodes[j].Id == id {
				parent := cloneMovieShallow(m)
				parent.Episodes = []EpisodesItem{m.Episodes[j]}
				return &Found{
					Kind: mdomain.KindMovieEpisode,
					Data: Root{Movies: []MoviesItem{parent}},
				}, true
			}
		}
	}

	// --- Series ---
	for i := range r.Series {
		s := &r.Series[i]
		if s.Id == id {
			out := cloneSeries(s, d)
			return &Found{Kind: mdomain.KindSeries, Data: Root{Series: []SeriesItem{out}}}, true
		}
		for j := range s.Seasons {
			season := &s.Seasons[j]
			if season.Id == id {
				parent := cloneSeriesShallow(s)
				parent.Seasons = []SeasonsItem{cloneSeason(season, d)}
				return &Found{
					Kind: mdomain.KindSeason,
					Data: Root{Series: []SeriesItem{parent}},
				}, true
			}
			for k := range season.Episodes {
				if season.Episodes[k].Id == id {
					parent := cloneSeriesShallow(s)
					sc := cloneSeasonShallow(season)
					sc.Episodes = []EpisodesItem{season.Episodes[k]}
					parent.Seasons = []SeasonsItem{sc}
					return &Found{
						Kind: mdomain.KindSeriesEpisode,
						Data: Root{Series: []SeriesItem{parent}},
					}, true
				}
			}
		}
	}

	return nil, false
}

// ---------------------------------------------------------------------------
// Clone helpers
// ---------------------------------------------------------------------------

func cloneStrings(s []string) []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}

func clonePersons(s []PersonsItem) []PersonsItem {
	if s == nil {
		return nil
	}
	out := make([]PersonsItem, len(s))
	copy(out, s)
	return out
}

func cloneEpisodes(s []EpisodesItem) []EpisodesItem {
	if s == nil {
		return nil
	}
	out := make([]EpisodesItem, len(s))
	copy(out, s)
	return out
}

func cloneMovieShallow(m *MoviesItem) MoviesItem {
	c := *m
	c.Countries = cloneStrings(m.Countries)
	c.Studios = cloneStrings(m.Studios)
	c.Genres = cloneStrings(m.Genres)
	c.Persons = clonePersons(m.Persons)
	c.Episodes = nil
	return c
}

func cloneMovie(m *MoviesItem, d Descendants) MoviesItem {
	c := cloneMovieShallow(m)
	if d == WithDescendants {
		c.Episodes = cloneEpisodes(m.Episodes)
	}
	return c
}

func cloneSeasonShallow(s *SeasonsItem) SeasonsItem {
	c := *s
	c.Persons = clonePersons(s.Persons)
	c.Episodes = nil
	return c
}

func cloneSeason(s *SeasonsItem, d Descendants) SeasonsItem {
	c := cloneSeasonShallow(s)
	if d == WithDescendants {
		c.Episodes = cloneEpisodes(s.Episodes)
	}
	return c
}

func cloneSeriesShallow(s *SeriesItem) SeriesItem {
	c := *s
	c.Countries = cloneStrings(s.Countries)
	c.Studios = cloneStrings(s.Studios)
	c.Genres = cloneStrings(s.Genres)
	c.Persons = clonePersons(s.Persons)
	c.Seasons = nil
	return c
}

func cloneSeries(s *SeriesItem, d Descendants) SeriesItem {
	c := cloneSeriesShallow(s)
	if d == WithDescendants {
		c.Seasons = make([]SeasonsItem, len(s.Seasons))
		for i := range s.Seasons {
			c.Seasons[i] = cloneSeason(&s.Seasons[i], WithDescendants)
		}
	}
	return c
}

// ---------------------------------------------------------------------------
// Found → Product conversion
// ---------------------------------------------------------------------------

// FoundToProduct converts a Found result into a domain Product.
func FoundToProduct(f *Found) *mdomain.Product {
	if f == nil {
		return nil
	}
	switch f.Kind {
	case mdomain.KindSeries:
		if len(f.Data.Series) == 0 {
			return nil
		}
		s := f.Data.Series[0]
		return &mdomain.Product{
			ID:            s.Id,
			Kind:          f.Kind,
			Title:         s.Title,
			SeriesTitle:   s.Title,
			Description:   s.Description,
			Genres:        s.Genres,
			Countries:     s.Countries,
			KinopoiskID:   s.KinopoiskId,
			IMDBID:        s.ImdbId,
		}
	case mdomain.KindSeason:
		if len(f.Data.Series) == 0 || len(f.Data.Series[0].Seasons) == 0 {
			return nil
		}
		season := f.Data.Series[0].Seasons[0]
		parent := f.Data.Series[0]
		return &mdomain.Product{
			ID:            season.Id,
			Kind:          f.Kind,
			Title:         season.Title,
			ParentID:      &parent.Id,
			SeasonNumber:  intPtr(season.SeasonNumber),
			SeriesTitle:   parent.Title,
			Genres:        parent.Genres,
			Countries:     parent.Countries,
			KinopoiskID:   parent.KinopoiskId,
			IMDBID:        parent.ImdbId,
		}
	case mdomain.KindSeriesEpisode:
		if len(f.Data.Series) == 0 || len(f.Data.Series[0].Seasons) == 0 {
			return nil
		}
		series := f.Data.Series[0]
		season := series.Seasons[0]
		ep := season.Episodes[0]
		parentID := season.Id
		return &mdomain.Product{
			ID:            ep.Id,
			Kind:          f.Kind,
			Title:         ep.Title,
			Description:   ep.Description,
			ParentID:      &parentID,
			SeasonNumber:  intPtr(season.SeasonNumber),
			EpisodeNumber: intPtr(ep.Number),
			SeasonEp:      seasonEp(season.SeasonNumber, ep.Number),
			SeriesTitle:   series.Title,
			Genres:        series.Genres,
			Countries:     series.Countries,
			KinopoiskID:   series.KinopoiskId,
			IMDBID:        series.ImdbId,
		}
	case mdomain.KindMovie:
		if len(f.Data.Movies) == 0 {
			return nil
		}
		m := f.Data.Movies[0]
		return &mdomain.Product{
			ID:          m.Id,
			Kind:        f.Kind,
			Title:       m.Title,
			SeriesTitle: m.Title,
			Description: m.Description,
			Genres:      m.Genres,
			Countries:   m.Countries,
			KinopoiskID: m.KinopoiskId,
			IMDBID:      m.ImdbId,
		}
	case mdomain.KindMovieEpisode:
		if len(f.Data.Movies) == 0 || len(f.Data.Movies[0].Episodes) == 0 {
			return nil
		}
		m := f.Data.Movies[0]
		ep := m.Episodes[0]
		return &mdomain.Product{
			ID:            ep.Id,
			Kind:          f.Kind,
			Title:         ep.Title,
			Description:   ep.Description,
			ParentID:      &m.Id,
			EpisodeNumber: intPtr(ep.Number),
			SeriesTitle:   m.Title,
			Genres:        m.Genres,
			Countries:     m.Countries,
			KinopoiskID:   m.KinopoiskId,
			IMDBID:        m.ImdbId,
		}
	}
	return nil
}

func intPtr(i int64) *int {
	n := int(i)
	return &n
}

func seasonEp(season, episode int64) string {
	return fmt.Sprintf("s%02de%02d", season, episode)
}
