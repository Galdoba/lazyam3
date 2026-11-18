// Package callname is the pure source-file naming domain. It constructs
// self-documenting call names for MediaUnit files before the rendering
// script operates on them. Translit logic is duplicated from
// internal to callname — transliteration engine for Cyrillic→Latin conversion
// in doc/callname_plan.md).
//
// Examples
// ScriptInput
// original_names: [12345.mp4, 12345-rus_Russkii_Forsirovannye.srt]
// Igra_lzhecov_s01e26_PRT260928200000.mp4
// Igra_lzhecov_s01e26_PRT260928200000-rus_Russkii_Forsirovannye.srt
// Igra_lzhecov_s01e26_PRT260928200000//
package callname

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Galdoba/lazyam3/internal/domain/callname/translit"
	"github.com/Galdoba/lazyam3/internal/domain/metadata"
)

type ScriptInput struct {
	Base         string   `json:"base"`
	SeasonTag    string   `json:"season_tag"`
	EpisodeTag   string   `json:"episode_tag"`
	PRT          string   `json:"prt"`
	OriginalName string   `json:"original_name"`
	FileTags     []string `json:"file_tags"`
}

func (si ScriptInput) String() string {
	s := ""
	s += si.Base
	if si.SeasonTag+si.EpisodeTag != "" {
		s += "_" + si.SeasonTag + si.EpisodeTag
	}
	s += "_" + si.PRT
	if len(si.FileTags) > 0 {
		s += "-" + strings.Join(si.FileTags, "_")
	}
	s += filepath.Ext(si.OriginalName)
	return s
}

func extractFileTags(stem string) []string {
	// Tags are separated from ID by '-' and from each other by '_'.
	// 0 tags is valid.
	idEnd := strings.IndexFunc(stem, func(r rune) bool { return r < '0' || r > '9' })
	if idEnd < 0 || idEnd == len(stem) {
		return nil
	}
	rest := stem[idEnd:] // e.g. "-rus_Russkii"
	if rest == "" {
		return nil
	}
	parts := strings.Split(rest, "_")
	if len(parts) > 0 && strings.HasPrefix(parts[0], "-") {
		parts[0] = parts[0][1:]
	}
	for i, p := range parts {
		if p == "" {
			parts = parts[:i]
			break
		}
	}
	return parts
}

func (si ScriptInput) validate() error {
	if si.Base == "" {
		return fmt.Errorf("base must be set")
	}
	if si.PRT == "" {
		return fmt.Errorf("PRT must be set")
	}
	if !prtRe.MatchString(si.PRT) {
		return fmt.Errorf("PRT invalid: %q", si.PRT)
	}
	return nil
}

// BuildInputName constructs a call name for a single file.
func BuildInputName(product metadata.Product, originalFilename, timestamp string) (ScriptInput, error) {
	stem, _ := splitExtension(originalFilename)

	si := ScriptInput{
		Base:         deriveOutBase(product),
		SeasonTag:    deriveNameTag(product),
		PRT:          "PRT" + timestamp,
		OriginalName: originalFilename,
		FileTags:     extractFileTags(stem),
	}
	return si, si.validate()
}

// BuildCallNames builds call names for all files in a MediaUnit.
// Keys are original filenames; fails fast if any file fails.
func BuildInputNames(product metadata.Product, filenames []string, timestamp string) (map[string]string, error) {
	names := make(map[string]string, len(filenames))
	for _, filename := range filenames {
		call, err := BuildInputName(product, filename, timestamp)
		if err != nil {
			return nil, fmt.Errorf("call name for %q: %w", filename, err)
		}
		names[filename] = call.String()
	}
	return names, nil
}

// deriveOutBase returns the transliterated SeriesTitle with first character
// uppercase, or "" when the product has no SeriesTitle.
func deriveOutBase(p metadata.Product) string {
	if p.SeriesTitle == "" {
		return ""
	}
	s := translit.String(p.SeriesTitle, translit.RegisterLow(), translit.Short_SE_Form())
	if len(s) > 0 {
		s = strings.ToUpper(string(s[0])) + s[1:]
	}
	return s
}

// deriveNameTag returns the ready-to-use season/episode tag: "s01e06"
// (season+episode), "s01" (season only), "e06" (episode only), or ""
// when the product has neither. Products without season/episode numbers
// (movies) must not get "s"/"e" fragments.
func deriveNameTag(p metadata.Product) string {
	var season, episode string
	if p.SeasonNumber != nil {
		season = fmt.Sprintf("s%02d", *p.SeasonNumber)
	}
	if p.EpisodeNumber != nil {
		episode = fmt.Sprintf("e%02d", *p.EpisodeNumber)
	}
	switch {
	case season != "" && episode != "":
		return season + episode
	case season != "":
		return season
	case episode != "":
		return episode
	}
	return ""
}

// splitExtension splits a filename into stem and extension at the last dot.
// Filenames without a dot return the whole name as stem and empty extension.
func splitExtension(filename string) (stem, ext string) {
	if idx := strings.LastIndex(filename, "."); idx > 0 {
		return filename[:idx], filename[idx+1:]
	}
	return filename, ""
}
