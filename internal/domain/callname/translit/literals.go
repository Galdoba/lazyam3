package translit

import "strings"

func defaultLiteralsMap() map[string]string {
	defaultLiteralsMap := make(map[string]string)
	//Symbols to remove
	//Symbols to underscore
	defaultLiteralsMap[" "] = "_" //space
	defaultLiteralsMap[","] = "_"
	defaultLiteralsMap[";"] = "_"
	defaultLiteralsMap[":"] = "_"
	defaultLiteralsMap["("] = "_"
	defaultLiteralsMap[")"] = "_"
	defaultLiteralsMap["_"] = "_"
	defaultLiteralsMap["."] = "_"
	defaultLiteralsMap[`"`] = "_"
	defaultLiteralsMap["'"] = "_"
	defaultLiteralsMap["`"] = "_"
	defaultLiteralsMap["…"] = "_"
	defaultLiteralsMap["«"] = "_"
	defaultLiteralsMap["»"] = "_"
	defaultLiteralsMap["*"] = "_"
	defaultLiteralsMap["/"] = "_"
	defaultLiteralsMap[`\`] = "_"
	defaultLiteralsMap["?"] = "_"
	defaultLiteralsMap["¡"] = "_"
	defaultLiteralsMap["-"] = "_"
	defaultLiteralsMap["–"] = "_"
	defaultLiteralsMap["—"] = "_"
	defaultLiteralsMap["#"] = "_"
	defaultLiteralsMap["!"] = "_"
	defaultLiteralsMap["№"] = "_"
	defaultLiteralsMap["+"] = "_"
	defaultLiteralsMap["="] = "_"
	defaultLiteralsMap["["] = "_"
	defaultLiteralsMap["]"] = "_"
	defaultLiteralsMap["&"] = "_"
	defaultLiteralsMap["’"] = "_"
	defaultLiteralsMap["„"] = "_"
	defaultLiteralsMap["“"] = "_"
	defaultLiteralsMap["|"] = "_"
	defaultLiteralsMap["®"] = "_"

	//Cyrillic
	defaultLiteralsMap["а"] = "a"
	defaultLiteralsMap["б"] = "b"
	defaultLiteralsMap["в"] = "v"
	defaultLiteralsMap["г"] = "g"
	defaultLiteralsMap["д"] = "d"
	defaultLiteralsMap["е"] = "e"
	defaultLiteralsMap["ё"] = "e"
	defaultLiteralsMap["ж"] = "zh"
	defaultLiteralsMap["з"] = "z"
	defaultLiteralsMap["и"] = "i"
	defaultLiteralsMap["й"] = "y"
	defaultLiteralsMap["к"] = "k"
	defaultLiteralsMap["л"] = "l"
	defaultLiteralsMap["м"] = "m"
	defaultLiteralsMap["н"] = "n"
	defaultLiteralsMap["о"] = "o"
	defaultLiteralsMap["п"] = "p"
	defaultLiteralsMap["р"] = "r"
	defaultLiteralsMap["с"] = "s"
	defaultLiteralsMap["т"] = "t"
	defaultLiteralsMap["у"] = "u"
	defaultLiteralsMap["ф"] = "f"
	defaultLiteralsMap["х"] = "h"
	defaultLiteralsMap["ц"] = "c"
	defaultLiteralsMap["ч"] = "ch"
	defaultLiteralsMap["ш"] = "sh"
	defaultLiteralsMap["щ"] = "sh"
	defaultLiteralsMap["ь"] = ""
	defaultLiteralsMap["ы"] = "y"
	defaultLiteralsMap["ъ"] = ""
	defaultLiteralsMap["э"] = "e"
	defaultLiteralsMap["ю"] = "yu"
	defaultLiteralsMap["я"] = "ya"
	//Non-Latin
	defaultLiteralsMap["à"] = "a"
	defaultLiteralsMap["ê"] = "e"
	defaultLiteralsMap["è"] = "e"
	defaultLiteralsMap["é"] = "e"
	defaultLiteralsMap["ü"] = "u"

	//Neizvestnyy_istoriya_odnogo_ubiycy_unknown

	return defaultLiteralsMap
}

func defaultIgnoreList() []string {
	return []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
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
