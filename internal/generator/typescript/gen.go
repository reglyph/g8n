// Package typescript generates TypeScript source from an environment schema.
package typescript

import (
	"fmt"

	"github.com/reglyph/g8n/internal/generator/core"
	"github.com/reglyph/g8n/internal/naming"
	"github.com/reglyph/g8n/internal/parser"
	"github.com/reglyph/g8n/internal/printer"
	"github.com/reglyph/g8n/internal/spec"
)

var reservedFieldNames = map[string]string{
	"env": "envValue",
}

type tsGen struct {
	schema    *parser.Schema
	uses      map[string]bool
	hasEnum   bool
	hasExpand bool
}

func (g *tsGen) fieldName(key string) string {
	n := naming.TSFieldName(key)

	if repl, ok := reservedFieldNames[n]; ok {
		return repl
	}

	return n
}

// Generate renders the schema as a TypeScript source file.
func Generate(s *parser.Schema) ([]byte, error) {
	return (&tsGen{schema: s}).build()
}

func (g *tsGen) build() ([]byte, error) {
	if err := core.CheckFieldNameCollisions(g.schema, g.fieldName); err != nil {
		return nil, err
	}

	if err := g.computeUses(); err != nil {
		return nil, err
	}

	var p printer.Printer
	g.writeHeader(&p)
	g.writeInterface(&p)
	g.writeSensitiveKeys(&p)

	if err := g.writeLoaders(&p); err != nil {
		return nil, err
	}

	g.writeEnumHelper(&p)
	g.writeExpandHelper(&p)
	g.writeRegexVars(&p)
	g.writeConversionHelpers(&p)

	return p.Bytes(), nil
}

func (g *tsGen) computeUses() error {
	g.uses = map[string]bool{}

	for _, f := range g.schema.Fields {
		switch f.Kind {
		case spec.KindInt, spec.KindInt64, spec.KindPort:
			g.uses["int"] = true
		case spec.KindBool:
			g.uses["bool"] = true
		case spec.KindFloat64:
			g.uses["float"] = true
		case spec.KindURL:
			g.uses["url"] = true
		case spec.KindEnum:
			g.hasEnum = true
		case spec.KindString, spec.KindEmail:
		default:
			return fmt.Errorf("unknown kind %s for variable %s", f.Kind, f.Key)
		}
	}

	for _, f := range g.schema.Fields {
		if core.Expandable(f) {
			g.hasExpand = true
		}
	}

	return nil
}

// regexVar returns the module-level regex variable name for the field.
func (g *tsGen) regexVar(f *parser.Field) string {
	return naming.TSFieldName(f.Key) + "Re"
}

// tsType returns the interface property type for the field.
func (g *tsGen) tsType(f *parser.Field) string {
	if f.Kind == spec.KindEnum {
		return naming.UpperFirst(g.fieldName(f.Key))
	}

	return f.Kind.Spec().TSType
}
