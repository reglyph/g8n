// Package generator dispatches schema generation to a language-specific generator package.
package generator

import (
	"fmt"

	"github.com/reglyph/g8n/internal/generator/golang"
	"github.com/reglyph/g8n/internal/generator/typescript"
	"github.com/reglyph/g8n/internal/parser"
	"github.com/reglyph/g8n/internal/spec"
)

// Generate renders the schema as a formatted source file in the given language.
func Generate(s *parser.Schema, lang spec.Lang) ([]byte, error) {
	switch lang {
	case spec.LangGo:
		return golang.Generate(s)
	case spec.LangTS:
		return typescript.Generate(s)
	default:
		return nil, fmt.Errorf("unsupported language %s", lang)
	}
}
