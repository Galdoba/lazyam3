package script

// Deprecated: replaced by callname package.
// func deriveOutBase(p metadata.Product) string {
// 	if p.SeriesTitle == "" {
// 		return ""
// 	}
//
// 	base := translit.String(p.SeriesTitle, translit.RegisterLow(), translit.Short_SE_Form())
// 	if len(base) > 0 {
// 		base = strings.ToUpper(string(base[0])) + base[1:]
// 	}
//
// 	var parts []string
// 	parts = append(parts, base)
//
// 	var se []string
// 	if p.SeasonNumber != nil {
// 		se = append(se, fmt.Sprintf("s%02d", *p.SeasonNumber))
// 	}
// 	if p.EpisodeNumber != nil {
// 		se = append(se, fmt.Sprintf("e%02d", *p.EpisodeNumber))
// 	}
// 	if len(se) > 0 {
// 		parts = append(parts, strings.Join(se, ""))
// 	}
//
// 	return strings.Join(parts, "_")
// }

// Deprecated: replaced by callname package.
// func deriveBaseName(p metadata.Product) string {
// 	s := translit.String(p.SeriesTitle, translit.RegisterLow(), translit.Short_SE_Form())
// 	if len(s) > 0 {
// 		s = strings.ToUpper(string(s[0])) + s[1:]
// 	}
// 	return s
// }

// Deprecated: replaced by callname package.
// func deriveSeasonTag(p metadata.Product) string {
// 	if p.SeasonNumber != nil {
// 		return fmt.Sprintf("%02d", *p.SeasonNumber)
// 	}
// 	return ""
// }

// Deprecated: replaced by callname package.
// func deriveEpisodeTag(p metadata.Product) string {
// 	if p.EpisodeNumber != nil {
// 		return fmt.Sprintf("%02d", *p.EpisodeNumber)
// 	}
// 	return ""
// }

// Deprecated: replaced by callname package.
// func deriveNameTag(seasonTag, episodeTag string) string {
// 	switch {
// 	case seasonTag != "" && episodeTag != "":
// 		return "_s" + seasonTag + "e" + episodeTag
// 	case seasonTag != "":
// 		return "_s" + seasonTag
// 	case episodeTag != "":
// 		return "_e" + episodeTag
// 	}
// 	return ""
// }

// Deprecated: replaced by callname package.
// func deriveSeasonDir(seasonTag string) string {
// 	if seasonTag == "" {
// 		return ""
// 	}
// 	return "_s" + seasonTag
// }
