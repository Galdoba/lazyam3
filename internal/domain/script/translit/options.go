package translit

type TranslitterOption func(*translitOpt)

type translitOpt struct {
	changeLiteralsMap          map[string]string
	ignoreList                 []string
	changeWhiteList            []string
	changeBlackList            []string
	segmentator                string
	wordSeparators             []string
	keepSegmentatorRepetitions bool
	keepRegister               bool
	short_SE_form              bool
	title                      bool
	caps                       bool
	byWords                    bool
}

func defaultTranslitterOptions() translitOpt {
	return translitOpt{
		changeLiteralsMap:          defaultLiteralsMap(),
		ignoreList:                 defaultIgnoreList(),
		changeWhiteList:            []string{},
		changeBlackList:            []string{},
		segmentator:                "_",
		wordSeparators:             []string{" ", "_"},
		keepRegister:               false,
		keepSegmentatorRepetitions: false,
		short_SE_form:              false,
		title:                      true,
		caps:                       false,
	}
}

func KeepRegister() TranslitterOption {
	return func(to *translitOpt) {
		to.keepRegister = true
	}
}

func RegisterLow() TranslitterOption {
	return func(to *translitOpt) {
		to.keepRegister = false
		to.title = false
	}
}

func RegisterHigh() TranslitterOption {
	return func(to *translitOpt) {
		to.keepRegister = false
		to.title = false
		to.caps = true
	}
}

func KeepUnderscoreRepetitions() TranslitterOption {
	return func(to *translitOpt) {
		to.keepRegister = true
	}
}

func Short_SE_Form() TranslitterOption {
	return func(to *translitOpt) {
		to.short_SE_form = true
	}
}
