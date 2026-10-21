package translit

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	wrdSep        = "||||||||"
	excludeMarker = "&&&&&&&&"
)

type translitter struct {
	input                         string
	changeLiteralsMap             map[string]string
	ignoreList                    []string
	changeWhiteList               []string
	changeBlackList               []string
	segmentator                   string
	wordSeparators                []string
	keepRegister                  bool
	keepSegmentatorRepetitions    bool
	short_SE_form                 bool
	shortenEpisode                bool
	title                         bool
	caps                          bool
	DEBUG_panicIfLiteralIsUNKNOWN bool
}

func newTranslitter(opts ...TranslitterOption) *translitter {
	tr := translitter{}
	options := defaultTranslitterOptions()
	for _, change := range opts {
		change(&options)
	}
	tr.changeLiteralsMap = options.changeLiteralsMap
	tr.ignoreList = options.ignoreList
	tr.changeWhiteList = options.changeWhiteList
	tr.changeBlackList = options.changeBlackList
	tr.segmentator = options.segmentator
	tr.wordSeparators = options.wordSeparators
	tr.keepRegister = options.keepRegister
	tr.keepSegmentatorRepetitions = options.keepSegmentatorRepetitions
	tr.short_SE_form = options.short_SE_form
	tr.title = options.title
	tr.caps = options.caps
	tr.DEBUG_panicIfLiteralIsUNKNOWN = true
	return &tr
}

func (tr *translitter) assertValid() {
	if len(tr.changeWhiteList) != 0 && len(tr.changeBlackList) != 0 {
		panic("translitter options include both WhiteList and BlackList")
	}
	for _, ignore := range tr.ignoreList {
		for _, wl := range tr.changeWhiteList {
			if ignore == wl {
				panic(fmt.Sprintf("transliteration imposible: literal '%v' is both in WhiteList and IgnoreList", ignore))
			}
		}
	}
}

func (tr *translitter) convert(input string) string {
	tr.input = input
	buffer := tr.input
	buffer = strings.TrimSpace(buffer)
	outputBuffer := ""
	words := seaparateWords(buffer, tr.wordSeparators...)
	for i, word := range words {
		words[i] = excludeBlackListLiterals(word, tr.changeBlackList...)
	}
	//for each word:
	for _, word := range words {
		lits := getLiterals(word)
		for _, lit := range lits {
			switch len(tr.changeWhiteList) {
			case 0:
				//convert all literals, but keep listed to ignore
				outputBuffer += tr.convertNonListed(tr.ignoreList, lit)
			default:
				//convert whiteList literals
				outputBuffer += tr.convertListed(tr.changeWhiteList, lit)
			}
		}
		outputBuffer += wrdSep
	}
	outputBuffer = strings.TrimSuffix(outputBuffer, wrdSep)
	outputBuffer = strings.ReplaceAll(outputBuffer, wrdSep, tr.segmentator)
	if !tr.keepSegmentatorRepetitions {
		outputBuffer = removeSegmentstorRepetitions(outputBuffer, tr.segmentator)
	}
	outputBuffer = strings.TrimSuffix(outputBuffer, tr.segmentator)
	if tr.short_SE_form {
		outputBuffer = applyShort_SE_form(outputBuffer)
	}
	if tr.title && len(outputBuffer) > 0 {
		outputBuffer = toTitle(outputBuffer)
	}
	if tr.caps {
		outputBuffer = strings.ToUpper(outputBuffer)
	}

	return outputBuffer
}

func (tr *translitter) convertListed(list []string, lit literal) string {
	if listed(list, lit.glyph) {
		return tr.convertLiteralToText(tr.changeLiteralsMap, lit)
	}
	return lit.glyph
}

func (tr *translitter) convertNonListed(list []string, lit literal) string {
	if !listed(list, lit.glyph) {
		return tr.convertLiteralToText(tr.changeLiteralsMap, lit)
	}
	return lit.glyph
}

func seaparateWords(text string, wsList ...string) []string {
	if len(wsList) == 0 {
		return []string{text}
	}
	for _, ws := range wsList {
		text = strings.ReplaceAll(text, ws, wrdSep)
	}
	return strings.Split(text, wrdSep)
}

func excludeBlackListLiterals(text string, blckList ...string) string {
	if len(blckList) == 0 {
		return text
	}
	for _, bl := range blckList {
		text = strings.ReplaceAll(text, bl, excludeMarker)
	}
	return strings.ReplaceAll(text, excludeMarker, "")
}

func getLiterals(text string) []literal {
	liters := []literal{}
	for _, s := range strings.Split(text, "") {
		liters = append(liters, newLiteral(s))
	}
	return liters
}

func listed(list []string, s string) bool {
	for _, l := range list {
		if l == s {
			return true
		}
	}
	return false
}

func applyShort_SE_form(text string) string {

	reS := regexp.MustCompile(`_(\d)+_sezon`)
	season := reS.FindString(text)
	reE := regexp.MustCompile(`_(\d)+_seriya`)
	episode := reE.FindString(text)
	seasonNum := ""
	episodeNum := ""
	if season != "" {
		seasonNum = strings.TrimPrefix(season, "_")
		seasonNum = strings.TrimSuffix(seasonNum, "_sezon")
		seasonNum = "_s" + seasonNum
		text = strings.ReplaceAll(text, season, seasonNum)
	}
	if episode != "" {
		episodeNum = strings.TrimPrefix(episode, "_")
		episodeNum = strings.TrimSuffix(episodeNum, "_seriya")
		episodeNum = "e" + episodeNum
		text = strings.ReplaceAll(text, episode, episodeNum)
	}

	return text
}

func (tr *translitter) convertLiteralToText(literalsMap map[string]string, lit literal) string {
	v, ok := tr.changeLiteralsMap[lit.glyph]
	if !ok {
		switch tr.DEBUG_panicIfLiteralIsUNKNOWN {
		case true:
			panic(fmt.Sprintf("transliteration imposible: \n%v\n'%v' is not in literalsMap", tr.input, lit.glyph))
		case false:
			v = lit.glyph
		}
	}
	if tr.keepRegister && lit.capped {
		v = strings.ToUpper(v)
	}
	return v
}

func toTitle(str string) string {
	converted := strings.Split(str, "")
	converted[0] = strings.ToUpper(converted[0])
	str = strings.Join(converted, "")
	return str
}

func removeSegmentstorRepetitions(text string, segmentator string) string {
	squashed := strings.ReplaceAll(text, segmentator+segmentator, segmentator)
	for squashed != text {
		text = squashed
		squashed = strings.ReplaceAll(squashed, segmentator+segmentator, segmentator)
	}
	return squashed
}

/*
file = translit.Normalize(file, pathValid)

*/

func String(input string, options ...TranslitterOption) string {
	tr := newTranslitter(options...)
	tr.assertValid()
	return tr.convert(input)
}

func RenamingVariants(input string, options ...TranslitterOption) []string {
	tr := newTranslitter(options...)
	tr.assertValid()
	full := tr.convert(input)
	segments := strings.Split(full, tr.segmentator)
	variants := []string{""}
	for i, segment := range segments {
		newVariant := variants[i] + tr.segmentator + segment
		newVariant = strings.TrimPrefix(newVariant, tr.segmentator)
		variants = append(variants, newVariant)
	}
	return variants[1:]
}

func RenamingBlocks(input string, options ...TranslitterOption) []string {
	tr := newTranslitter(options...)
	tr.assertValid()
	full := tr.convert(input)
	segments := strings.Split(full, tr.segmentator)
	return segments
}
