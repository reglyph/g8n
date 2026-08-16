// Package generator dispatches schema generation to a language-specific
// generator package.
package generator

import (
	"fmt"

	"github.com/reglyph/g8n/internal/generator/golang"
	"github.com/reglyph/g8n/internal/generator/typescript"
	"github.com/reglyph/g8n/internal/parser"
	"github.com/reglyph/g8n/internal/spec"
)

var generators = map[spec.Lang]func(*parser.Schema) ([]byte, error){
	spec.LangGo: golang.Generate,
	spec.LangTS: typescript.Generate,
}

// Generate renders the schema as a formatted source file in the given language.
func Generate(s *parser.Schema, lang spec.Lang) ([]byte, error) {
	f, ok := generators[lang]
	if !ok {
		return nil, fmt.Errorf("unsupported language %s", lang)
	}

	return f(s)
}

// Supported reports whether the language has a registered generator.
func Supported(lang spec.Lang) bool {
	_, ok := generators[lang]

	return ok
}
