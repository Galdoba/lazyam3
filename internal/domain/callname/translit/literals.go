package translit

import "strings"

func defaultLiteralsMap() map[string]string {
	literalsMap := make(map[string]string)
	// Symbols to remove
	// Symbols to underscore
	literalsMap[" "] = "_" // space
	literalsMap[","] = "_"
	literalsMap[";"] = "_"
	literalsMap[":"] = "_"
	literalsMap["("] = "_"
	literalsMap[")"] = "_"
	literalsMap["_"] = "_"
	literalsMap["."] = "_"
	literalsMap[`"`] = "_"
	literalsMap["'"] = "_"
	literalsMap["`"] = "_"
	literalsMap["…"] = "_"
	literalsMap["«"] = "_"
	literalsMap["»"] = "_"
	literalsMap["*"] = "_"
	literalsMap["/"] = "_"
	literalsMap[`\`] = "_"
	literalsMap["?"] = "_"
	literalsMap["¡"] = "_"
	literalsMap["-"] = "_"
	literalsMap["–"] = "_"
	literalsMap["—"] = "_"
	literalsMap["#"] = "_"
	literalsMap["!"] = "_"
	literalsMap["№"] = "_"
	literalsMap["+"] = "_"
	literalsMap["="] = "_"
	literalsMap["["] = "_"
	literalsMap["]"] = "_"
	literalsMap["&"] = "_"
	literalsMap["’"] = "_"
	literalsMap["„"] = "_"
	literalsMap["“"] = "_"
	literalsMap["|"] = "_"
	literalsMap["®"] = "_"

	// Cyrillic
	literalsMap["а"] = "a"
	literalsMap["б"] = "b"
	literalsMap["в"] = "v"
	literalsMap["г"] = "g"
	literalsMap["д"] = "d"
	literalsMap["е"] = "e"
	literalsMap["ё"] = "e"
	literalsMap["ж"] = "zh"
	literalsMap["з"] = "z"
	literalsMap["и"] = "i"
	literalsMap["й"] = "y"
	literalsMap["к"] = "k"
	literalsMap["л"] = "l"
	literalsMap["м"] = "m"
	literalsMap["н"] = "n"
	literalsMap["о"] = "o"
	literalsMap["п"] = "p"
	literalsMap["р"] = "r"
	literalsMap["с"] = "s"
	literalsMap["т"] = "t"
	literalsMap["у"] = "u"
	literalsMap["ф"] = "f"
	literalsMap["х"] = "h"
	literalsMap["ц"] = "c"
	literalsMap["ч"] = "ch"
	literalsMap["ш"] = "sh"
	literalsMap["щ"] = "sh"
	literalsMap["ь"] = ""
	literalsMap["ы"] = "y"
	literalsMap["ъ"] = ""
	literalsMap["э"] = "e"
	literalsMap["ю"] = "yu"
	literalsMap["я"] = "ya"
	// Non-Latin
	literalsMap["à"] = "a"
	literalsMap["ê"] = "e"
	literalsMap["è"] = "e"
	literalsMap["é"] = "e"
	literalsMap["ü"] = "u"

	// Neizvestnyy_istoriya_odnogo_ubiycy_unknown

	return literalsMap
}

func defaultIgnoreList() []string {
	return []string{
		"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
		"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
	}
}

type literal struct {
	glyph  string
	capped bool
}

func newLiteral(s string) literal {
	l := literal{}
	if s == strings.ToUpper(s) {
		l.capped = true
	}
	l.glyph = strings.ToLower(s)
	return l
}
