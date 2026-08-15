package spec

// Lang identifies a supported output language.
type Lang int

const (
	// LangGo is the Go output language.
	LangGo Lang = iota
	// LangTS is the TypeScript output language.
	LangTS
)

// String returns the CLI name of the language.
func (l Lang) String() string {
	switch l {
	case LangTS:
		return "ts"
	default:
		return "go"
	}
}

// ParseLang resolves a language name to a Lang, reporting whether it is known.
func ParseLang(name string) (Lang, bool) {
	switch name {
	case "go":
		return LangGo, true
	case "ts":
		return LangTS, true
	default:
		return 0, false
	}
}
