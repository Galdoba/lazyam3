package script

import (
	"fmt"
	"strings"

	"github.com/Galdoba/lazyam3/internal/domain/metadata"
	"github.com/Galdoba/lazyam3/internal/domain/script/translit"
)

// deriveOutBase derives the output filename base from the product.
// Returns something like "Belaya_koroleva_s01e01".
func deriveOutBase(p metadata.Product) string {
	if p.SeriesTitle == "" {
		return ""
	}

	base := translit.String(p.SeriesTitle, translit.RegisterLow(), translit.Short_SE_Form())
	if len(base) > 0 {
		base = strings.ToUpper(string(base[0])) + base[1:]
	}

	var parts []string
	parts = append(parts, base)

	var se []string
	if p.SeasonNumber != nil {
		se = append(se, fmt.Sprintf("s%02d", *p.SeasonNumber))
	}
	if p.EpisodeNumber != nil {
		se = append(se, fmt.Sprintf("e%02d", *p.EpisodeNumber))
	}
	if len(se) > 0 {
		parts = append(parts, strings.Join(se, ""))
	}

	return strings.Join(parts, "_")
}

// deriveBaseName returns the transliterated series title with first character uppercase.
func deriveBaseName(p metadata.Product) string {
	s := translit.String(p.SeriesTitle, translit.RegisterLow(), translit.Short_SE_Form())
	if len(s) > 0 {
		s = strings.ToUpper(string(s[0])) + s[1:]
	}
	return s
}

// deriveSeasonTag returns the season number as a zero-padded string.
func deriveSeasonTag(p metadata.Product) string {
	if p.SeasonNumber != nil {
		return fmt.Sprintf("%02d", *p.SeasonNumber)
	}
	return ""
}

// deriveEpisodeTag returns the episode number as a zero-padded string.
func deriveEpisodeTag(p metadata.Product) string {
	if p.EpisodeNumber != nil {
		return fmt.Sprintf("%02d", *p.EpisodeNumber)
	}
	return ""
}

// deriveNameTag returns the ready-to-use season/episode suffix appended to
// BASENAME: "_s01e06" (season+episode), "_s01" (season only), "_e06"
// (episode only), or "" when the product has neither. Products without
// season/episode numbers (movies) must not get "_s"/"e" fragments.
func deriveNameTag(seasonTag, episodeTag string) string {
	switch {
	case seasonTag != "" && episodeTag != "":
		return "_s" + seasonTag + "e" + episodeTag
	case seasonTag != "":
		return "_s" + seasonTag
	case episodeTag != "":
		return "_e" + episodeTag
	}
	return ""
}

// deriveSeasonDir returns the target-directory suffix "_s01", or "" when the
// product has no season (movies and products without a season number).
func deriveSeasonDir(seasonTag string) string {
	if seasonTag == "" {
		return ""
	}
	return "_s" + seasonTag
}
