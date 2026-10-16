package metadata

func (r *Root) FindByID(id int64, d Descendants) (*Found, bool) {
	if r == nil {
		return nil, false
	}

	// --- Фильмы ---
	for i := range r.Movies {
		m := &r.Movies[i]
		if m.Id == id {
			out := cloneMovie(m, d)
			return &Found{Kind: KindMovie, Data: Root{Movies: []MoviesItem{out}}}, true
		}
		for j := range m.Episodes {
			if m.Episodes[j].Id == id {
				parent := cloneMovieShallow(m)
				parent.Episodes = []EpisodesItem{m.Episodes[j]}
				return &Found{
					Kind: KindMovieEpisode,
					Data: Root{Movies: []MoviesItem{parent}},
				}, true
			}
		}
	}

	// --- Сериалы ---
	for i := range r.Series {
		s := &r.Series[i]
		if s.Id == id {
			out := cloneSeries(s, d)
			return &Found{Kind: KindSeries, Data: Root{Series: []SeriesItem{out}}}, true
		}
		for j := range s.Seasons {
			season := &s.Seasons[j]
			if season.Id == id {
				parent := cloneSeriesShallow(s)
				parent.Seasons = []SeasonsItem{cloneSeason(season, d)}
				return &Found{Kind: KindSeason, Data: Root{Series: []SeriesItem{parent}}}, true
			}
			for k := range season.Episodes {
				if season.Episodes[k].Id == id {
					parent := cloneSeriesShallow(s)
					sc := cloneSeasonShallow(season)
					sc.Episodes = []EpisodesItem{season.Episodes[k]}
					parent.Seasons = []SeasonsItem{sc}
					return &Found{
						Kind: KindSeriesEpisode,
						Data: Root{Series: []SeriesItem{parent}},
					}, true
				}
			}
		}
	}

	return nil, false
}

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
	copy(out, s) // плоская структура — хватает копии слайса
	return out
}

// shallow = сам объект, дети обнулены
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
			// Сезоны — уже как часть «поддерева сериала»,
			// поэтому их эпизоды берём целиком.
			c.Seasons[i] = cloneSeason(&s.Seasons[i], WithDescendants)
		}
	}
	return c
}
