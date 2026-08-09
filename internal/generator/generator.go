package generator

import (
	"fmt"
	"strings"

	"go/format"

	"github.com/whoqmi/g8n/internal/naming"
	"github.com/whoqmi/g8n/internal/parser"
	"github.com/whoqmi/g8n/internal/spec"
)

var reservedFieldNames = map[string]string{
	"Env":           "EnvValue",
	"SensitiveKeys": "SensitiveKeysValue",
	"Load":          "LoadValue",
	"LoadFrom":      "LoadFromValue",
	"ExpandVars":    "ExpandVarsValue",
}

// Generate renders the schema as a formatted Go source file.
func Generate(s *parser.Schema) ([]byte, error) {
	g := &gen{schema: s}

	src, err := g.build()
	if err != nil {
		return nil, err
	}

	formatted, err := format.Source(src)
	if err != nil {
		return nil, fmt.Errorf("generated source for schema %q is not valid Go: %w", s.SourcePath, err)
	}

	return formatted, nil
}

type printer struct {
	buf   strings.Builder
	depth int
}

func (p *printer) indent() { p.depth++ }
func (p *printer) dedent() { p.depth-- }

func (p *printer) line(s string) {
	p.buf.WriteString(strings.Repeat("\t", p.depth))
	p.buf.WriteString(s)
	p.buf.WriteByte('\n')
}

func (p *printer) linef(f string, args ...any) {
	p.line(fmt.Sprintf(f, args...))
}

func (p *printer) blank() {
	p.buf.WriteByte('\n')
}

func (p *printer) writeRaw(s string) {
	p.buf.WriteString(s)
}

func (p *printer) bytes() []byte {
	return []byte(p.buf.String())
}

type gen struct {
	schema    *parser.Schema
	uses      map[string]bool
	hasEnum   bool
	hasExpand bool
}

func (g *gen) fieldName(key string) string {
	n := naming.GoFieldName(key)

	if repl, ok := reservedFieldNames[n]; ok {
		return repl
	}

	return n
}

func (g *gen) checkFieldNameCollisions() error {
	seen := make(map[string]*parser.Field, len(g.schema.Fields))

	for _, f := range g.schema.Fields {
		name := g.fieldName(f.Key)

		if first, ok := seen[name]; ok {
			return fmt.Errorf("line %d: variable %q: Go field name %q collides with variable %q declared on line %d",
				f.Line, f.Key, name, first.Key, first.Line)
		}

		seen[name] = f
	}

	return nil
}

func (g *gen) build() ([]byte, error) {
	if err := g.checkFieldNameCollisions(); err != nil {
		return nil, err
	}

	g.computeUses()

	var p printer
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

	return p.bytes(), nil
}

func (g *gen) computeUses() {
	g.uses = map[string]bool{"os": true, "strings": true}

	for _, f := range g.schema.Fields {
		//goland:noinspection GoSwitchMissingCasesForIotaConsts
		switch f.Kind {
		case spec.KindInt, spec.KindInt64, spec.KindBool, spec.KindFloat64, spec.KindPort:
			g.uses["strconv"] = true
		case spec.KindURL:
			g.uses["net/url"] = true
		case spec.KindEnum:
			g.hasEnum = true
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
}

func (g *gen) expandable(f *parser.Field) bool {
	if f.Sensitive || f.HasRegex() {
		return false
	}

	return f.Kind.IsStringLike()
}
