package typescript

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/reglyph/g8n/internal/parser"
	"github.com/reglyph/g8n/internal/printer"
	"github.com/reglyph/g8n/internal/spec"
)

func (g *tsGen) writeFieldLoad(p *printer.Printer, f *parser.Field) error {
	target := "e." + g.fieldName(f.Key)
	key := strconv.Quote(f.Key)

	hasDefault := f.HasDefault && !f.Sensitive

	p.Line("{")
	p.Indent()

	if f.Required {
		if !hasDefault {
			p.Linef("const v = m[%s];", key)
			p.Line("if (v === undefined || v === \"\") {")
			p.Indent()
			p.Linef("throw new Error('env: required variable %s is missing or empty');", key)
			p.Dedent()
			p.Line("}")

			if err := g.convertFromV(p, f, target, "v"); err != nil {
				return err
			}

			p.Dedent()
			p.Line("}")

			return nil
		}

		p.Linef("const v = m[%s];", key)
		p.Line("if (v === undefined || v === \"\") {")
		p.Indent()

		if err := g.writeDefaultValue(p, f, target); err != nil {
			return err
		}

		p.Dedent()
		p.Line("} else {")
		p.Indent()

		if err := g.convertFromV(p, f, target, "v"); err != nil {
			return err
		}

		p.Dedent()
		p.Line("}")

		p.Dedent()
		p.Line("}")

		return nil
	}

	p.Linef("const v = m[%s];", key)
	p.Line("if (v !== undefined && v !== \"\") {")
	p.Indent()

	if err := g.convertFromV(p, f, target, "v"); err != nil {
		return err
	}

	p.Dedent()

	if hasDefault {
		p.Line("} else {")
		p.Indent()

		if err := g.writeDefaultValue(p, f, target); err != nil {
			return err
		}

		p.Dedent()
	}

	p.Line("}")
	p.Dedent()
	p.Line("}")

	return nil
}

// convertFromV expands the value (mirroring goGen's expandStmt) and converts
// it into the target field.
func (g *tsGen) convertFromV(p *printer.Printer, f *parser.Field, target, src string) error {
	if g.expandable(f) {
		p.Linef("const t = expandVars(%s, m);", src)
		src = "t"
	}

	return g.writeConversion(p, f, target, src)
}

func (g *tsGen) writeDefaultValue(p *printer.Printer, f *parser.Field, target string) error {
	lit, err := g.literal(f)
	if err != nil {
		return err
	}

	if f.Kind.IsStringLike() {
		p.Linef("const t = %s;", g.expandExpr(f, lit))

		return g.writeConversion(p, f, target, "t")
	}

	p.Linef("%s = %s;", target, lit)

	return nil
}

func (g *tsGen) expandExpr(f *parser.Field, expr string) string {
	if g.expandable(f) {
		return "expandVars(" + expr + ", m)"
	}

	return expr
}

func (g *tsGen) literal(f *parser.Field) (string, error) {
	if !f.HasDefault {
		return g.tsZeroLiteral(f.Kind), nil
	}

	return g.tsTypedLiteral(f)
}

func (g *tsGen) tsZeroLiteral(kind spec.Kind) string {
	switch kind {
	case spec.KindString, spec.KindURL, spec.KindEmail, spec.KindEnum:
		return `""`
	case spec.KindInt, spec.KindInt64, spec.KindPort:
		return "0"
	case spec.KindBool:
		return "false"
	case spec.KindFloat64:
		return "0"
	default:
		return `""`
	}
}

func (g *tsGen) tsTypedLiteral(f *parser.Field) (string, error) {
	lit, err := f.Kind.ParseLiteral(f.Default)
	if err != nil {
		return "", fmt.Errorf("variable %s: %w", f.Key, err)
	}

	switch f.Kind {
	case spec.KindString, spec.KindURL, spec.KindEmail, spec.KindEnum:
		return strconv.Quote(lit.Str), nil
	case spec.KindInt, spec.KindInt64, spec.KindPort:
		return strconv.FormatInt(lit.Int, 10), nil
	case spec.KindFloat64:
		return strconv.FormatFloat(lit.Float, 'g', -1, 64), nil
	case spec.KindBool:
		return strconv.FormatBool(lit.Bool), nil
	default:
		return "", fmt.Errorf("unknown kind %s for variable %s", f.Kind, f.Key)
	}
}

func (g *tsGen) writeConversion(p *printer.Printer, f *parser.Field, target, src string) error {
	if err := g.writeValueCheck(p, f, src); err != nil {
		return err
	}

	g.writeConstraints(p, f, src)

	return g.writeAssignment(p, f, target, src)
}

func (g *tsGen) writeValueCheck(p *printer.Printer, f *parser.Field, src string) error {
	switch f.Kind {
	case spec.KindURL:
		p.Linef("if (!isValidUrl(%s)) {", src)
		p.Indent()
		g.writeURLError(p, f, src)
		p.Dedent()
		p.Line("}")

	case spec.KindEmail:
		p.Linef("const at = %s.indexOf(\"@\");", src)
		p.Linef("const dot = %s.lastIndexOf(\".\");", src)
		p.Line("if (at <= 0 || dot <= at + 1) {")
		p.Indent()
		g.writeEmailError(p, f, src)
		p.Dedent()
		p.Line("}")

	case spec.KindEnum:
		p.Linef("if (!(%s as readonly string[]).includes(%s)) {", g.enumValuesName(f), src)
		p.Indent()
		g.writeEnumError(p, f, src)
		p.Dedent()
		p.Line("}")

	case spec.KindString, spec.KindInt, spec.KindInt64, spec.KindBool, spec.KindFloat64, spec.KindPort:
		// nothing to validate

	default:
		return fmt.Errorf("unknown kind %s for variable %s", f.Kind, f.Key)
	}

	return nil
}

func (g *tsGen) writeAssignment(p *printer.Printer, f *parser.Field, target, src string) error {
	switch f.Kind {
	case spec.KindInt:
		p.Linef("const n = parseIntStrict(%s);", src)
		p.Line("if (n === null) {")
		p.Indent()
		g.writeConvError(p, f, "an integer", src)
		p.Dedent()
		p.Line("}")
		p.Linef("%s = n;", target)

	case spec.KindInt64:
		p.Linef("const n = parseIntStrict(%s);", src)
		p.Line("if (n === null) {")
		p.Indent()
		g.writeConvError(p, f, "an int64", src)
		p.Dedent()
		p.Line("}")
		p.Linef("%s = n;", target)

	case spec.KindPort:
		p.Linef("const n = parseIntStrict(%s);", src)
		p.Line("if (n === null) {")
		p.Indent()
		g.writeConvError(p, f, "an integer", src)
		p.Dedent()
		p.Line("}")
		p.Linef("if (n < %d || n > %d) {", spec.PortMin, spec.PortMax)
		p.Indent()
		g.writePortError(p, f)
		p.Dedent()
		p.Line("}")
		p.Linef("%s = n;", target)

	case spec.KindBool:
		p.Linef("const b = parseBoolStrict(%s);", src)
		p.Line("if (b === null) {")
		p.Indent()
		g.writeConvError(p, f, "a boolean", src)
		p.Dedent()
		p.Line("}")
		p.Linef("%s = b;", target)

	case spec.KindFloat64:
		p.Linef("const f = parseFloatStrict(%s);", src)
		p.Line("if (f === null) {")
		p.Indent()
		g.writeConvError(p, f, "a float64", src)
		p.Dedent()
		p.Line("}")
		p.Linef("%s = f;", target)

	case spec.KindString, spec.KindURL, spec.KindEmail:
		p.Linef("%s = %s;", target, src)

	case spec.KindEnum:
		p.Linef("%s = %s as %s;", target, src, g.enumTypeName(f))

	default:
		return fmt.Errorf("unknown kind %s for variable %s", f.Kind, f.Key)
	}

	return nil
}

func (g *tsGen) writeConstraints(p *printer.Printer, f *parser.Field, src string) {
	cs := parser.BuildConstraints(f)
	if len(cs) == 0 {
		return
	}

	for _, c := range cs {
		c.Emit(p, f, src, g.regexVar(f), spec.LangTS)
	}
}

func (g *tsGen) writeConvError(p *printer.Printer, f *parser.Field, what, src string) {
	if f.Sensitive {
		p.Linef("throw new Error('env: %s: value is not %s');", f.Key, what)
	} else {
		p.Linef("throw new Error(`env: %s: value \"${%s}\" is not %s`);", f.Key, src, what)
	}
}

func (g *tsGen) writePortError(p *printer.Printer, f *parser.Field) {
	if f.Sensitive {
		p.Linef("throw new Error('env: %s: port is out of range %d..%d');", f.Key, spec.PortMin, spec.PortMax)
	} else {
		p.Linef("throw new Error(`env: %s: port \"${n}\" is out of range %d..%d`);", f.Key, spec.PortMin, spec.PortMax)
	}
}

func (g *tsGen) writeURLError(p *printer.Printer, f *parser.Field, src string) {
	if f.Sensitive {
		p.Linef("throw new Error('env: %s: value is not a valid URL');", f.Key)
	} else {
		p.Linef("throw new Error(`env: %s: value \"${%s}\" is not a valid URL`);", f.Key, src)
	}
}

func (g *tsGen) writeEmailError(p *printer.Printer, f *parser.Field, src string) {
	if f.Sensitive {
		p.Linef("throw new Error('env: %s: value is not a valid email');", f.Key)
	} else {
		p.Linef("throw new Error(`env: %s: value \"${%s}\" is not a valid email`);", f.Key, src)
	}
}

func (g *tsGen) writeEnumError(p *printer.Printer, f *parser.Field, src string) {
	allowed := "[" + strings.Join(f.Enum, ", ") + "]"

	if f.Sensitive {
		p.Linef("throw new Error('env: %s: value is not in the allowed list %s');", f.Key, allowed)
	} else {
		p.Linef("throw new Error(`env: %s: value \"${%s}\" is not in the allowed list %s`);", f.Key, src, allowed)
	}
}
