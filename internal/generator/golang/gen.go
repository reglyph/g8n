// Package golang generates Go source from an environment schema.
package golang

import (
	"fmt"

	"go/format"

	"github.com/reglyph/g8n/internal/generator/core"
	"github.com/reglyph/g8n/internal/naming"
	"github.com/reglyph/g8n/internal/parser"
	"github.com/reglyph/g8n/internal/printer"
	"github.com/reglyph/g8n/internal/spec"
)

var reservedFieldNames = map[string]string{
	"Env":           "EnvValue",
	"SensitiveKeys": "SensitiveKeysValue",
	"Load":          "LoadValue",
	"LoadFrom":      "LoadFromValue",
	"ExpandVars":    "ExpandVarsValue",
}

type goGen struct {
	schema    *parser.Schema
	uses      map[string]bool
	hasEnum   bool
	hasExpand bool
}

func (g *goGen) fieldName(key string) string {
	n := naming.GoFieldName(key)

	if repl, ok := reservedFieldNames[n]; ok {
		return repl
	}

	return n
}

// Generate renders the schema as a Go source file.
func Generate(s *parser.Schema) ([]byte, error) {
	return (&goGen{schema: s}).build()
}

func (g *goGen) build() ([]byte, error) {
	if err := core.CheckFieldNameCollisions(g.schema, g.fieldName); err != nil {
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

		if core.Expandable(f) {
			g.hasExpand = true
			g.uses["regexp"] = true
		}
	}

	return nil
}
