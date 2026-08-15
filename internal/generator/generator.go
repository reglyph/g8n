package generator

import (
	"fmt"

	"go/format"

	"github.com/reglyph/g8n/internal/naming"
	"github.com/reglyph/g8n/internal/parser"
	"github.com/reglyph/g8n/internal/printer"
	"github.com/reglyph/g8n/internal/spec"
)

var goReservedFieldNames = map[string]string{
	"Env":           "EnvValue",
	"SensitiveKeys": "SensitiveKeysValue",
	"Load":          "LoadValue",
	"LoadFrom":      "LoadFromValue",
	"ExpandVars":    "ExpandVarsValue",
}

// Generate renders the schema as a formatted source file in the given language.
func Generate(s *parser.Schema, lang spec.Lang) ([]byte, error) {
	switch lang {
	case spec.LangGo:
		return (&goGen{schema: s}).build()
	default:
		return nil, fmt.Errorf("unsupported language %s", lang)
	}
}

type goGen struct {
	schema    *parser.Schema
	uses      map[string]bool
	hasEnum   bool
	hasExpand bool
}

func (g *goGen) fieldName(key string) string {
	n := naming.GoFieldName(key)

	if repl, ok := goReservedFieldNames[n]; ok {
		return repl
	}

	return n
}

func checkFieldNameCollisions(s *parser.Schema, fieldName func(string) string) error {
	seen := make(map[string]*parser.Field, len(s.Fields))

	for _, f := range s.Fields {
		name := fieldName(f.Key)

		if first, ok := seen[name]; ok {
			return fmt.Errorf("line %d: variable %q: field name %q collides with variable %q declared on line %d",
				f.Line, f.Key, name, first.Key, first.Line)
		}

		seen[name] = f
	}

	return nil
}

func (g *goGen) build() ([]byte, error) {
	if err := checkFieldNameCollisions(g.schema, g.fieldName); err != nil {
		return nil, err
	}

	if err := g.computeUses(); err != nil {
		return nil, err
	}

	var p printer.Printer
	if err := g.writeHeader(&p); err != nil {
		return nil, err
	}

	g.writeImports(&p)
	g.writeStruct(&p)
	g.writeSensitiveKeys(&p)

	if err := g.writeLoaders(&p); err != nil {
		return nil, err
	}

	g.writeEnumHelper(&p)
	g.writeExpandHelper(&p)
	g.writeRegexVars(&p)

	formatted, err := format.Source(p.Bytes())
	if err != nil {
		return nil, fmt.Errorf("generated source for schema %q is not valid Go: %w", g.schema.SourcePath, err)
	}

	return formatted, nil
}

func (g *goGen) computeUses() error {
	g.uses = map[string]bool{"os": true, "strings": true}

	for _, f := range g.schema.Fields {
		switch f.Kind {
		case spec.KindInt, spec.KindInt64, spec.KindBool, spec.KindFloat64, spec.KindPort:
			g.uses["strconv"] = true
		case spec.KindURL:
			g.uses["net/url"] = true
		case spec.KindEnum:
			g.hasEnum = true
		case spec.KindString, spec.KindEmail:
		default:
			return fmt.Errorf("unknown kind %s for variable %s", f.Kind, f.Key)
		}

		if f.Required && (!f.HasDefault || f.Sensitive) {
			g.uses["fmt"] = true
		}

		if f.Kind != spec.KindString {
			g.uses["fmt"] = true
		}

		if f.StartsWith != "" || f.HasRegex() {
			g.uses["fmt"] = true

			if f.HasRegex() {
				g.uses["regexp"] = true
			}
		}

		if g.expandable(f) {
			g.hasExpand = true
			g.uses["regexp"] = true
		}
	}

	return nil
}

func (g *goGen) expandable(f *parser.Field) bool {
	if f.Sensitive || f.HasRegex() {
		return false
	}

	return f.Kind.IsStringLike()
}
