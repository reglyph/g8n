package golang

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/reglyph/g8n/internal/naming"
	"github.com/reglyph/g8n/internal/parser"
	"github.com/reglyph/g8n/internal/parser/constraints"
	"github.com/reglyph/g8n/internal/printer"
	"github.com/reglyph/g8n/internal/spec"
)

func (g *goGen) writeFieldLoad(p *printer.Printer, f *parser.Field) error {
	target := "e." + g.fieldName(f.Key)
	key := strconv.Quote(f.Key)

	hasDefault := f.HasDefault && !f.Sensitive
	expand := g.expandable(f)
	expandStmt := func(p *printer.Printer, src string) {
		if expand {
			p.Linef("%s = expandVars(%s, m)", src, src)
		}
	}

	if f.Required {
		if !hasDefault {
			p.Line("{")
			p.Indent()
			p.Linef("v, ok := m[%s]", key)
			p.Line("if !ok || v == \"\" {")
			p.Indent()
			p.Linef("return e, fmt.Errorf(\"env: required variable %%q is missing or empty\", %s)", key)
			p.Dedent()
			p.Line("}")
			expandStmt(p, "v")

			if err := g.writeConversion(p, f, target, "v", false); err != nil {
				return err
			}

			p.Dedent()
			p.Line("}")

			return nil
		}

		p.Line("{")
		p.Indent()
		p.Linef("v, ok := m[%s]", key)
		p.Line("if !ok || v == \"\" {")
		p.Indent()

		if err := g.writeDefaultValue(p, f, target, false); err != nil {
			return err
		}

		p.Dedent()
		p.Line("} else {")
		p.Indent()
		expandStmt(p, "v")

		if err := g.writeConversion(p, f, target, "v", false); err != nil {
			return err
		}

		p.Dedent()
		p.Line("}")
		p.Dedent()
		p.Line("}")

		return nil
	}

	p.Line("{")
	p.Indent()
	p.Linef("if v, ok := m[%s]; ok && v != \"\" {", key)
	p.Indent()
	expandStmt(p, "v")

	if err := g.writeConversion(p, f, target, "v", true); err != nil {
		return err
	}

	if hasDefault {
		p.Dedent()
		p.Line("} else {")
		p.Indent()

		if err := g.writeDefaultValue(p, f, target, true); err != nil {
			return err
		}
	}

	p.Dedent()
	p.Line("}")
	p.Dedent()
	p.Line("}")

	return nil
}

func (g *goGen) writeDefaultValue(p *printer.Printer, f *parser.Field, target string, ptr bool) error {
	lit, err := g.literal(f)
	if err != nil {
		return err
	}

	if f.Kind.IsStringLike() {
		p.Linef("t := %s", g.expandExpr(f, lit))

		return g.writeConversion(p, f, target, "t", ptr)
	}

	if ptr {
		p.Linef("t := %s", lit)
		p.Linef("%s = &t", target)

		return nil
	}

	p.Linef("%s = %s", target, lit)

	return nil
}

func (g *goGen) expandExpr(f *parser.Field, expr string) string {
	if g.expandable(f) {
		return "expandVars(" + expr + ", m)"
	}

	return expr
}

func (g *goGen) literal(f *parser.Field) (string, error) {
	if !f.HasDefault {
		return g.zeroLiteral(f.Kind), nil
	}

	return g.typedLiteral(f)
}

func (g *goGen) zeroLiteral(kind spec.Kind) string {
	switch kind {
	case spec.KindInt, spec.KindPort:
		return "0"
	case spec.KindInt64:
		return "int64(0)"
	case spec.KindBool:
		return "false"
	case spec.KindFloat64:
		return "0"
	default:
		return `""`
	}
}

func (g *goGen) typedLiteral(f *parser.Field) (string, error) {
	lit, err := f.Kind.ParseLiteral(f.Default)
	if err != nil {
		return "", fmt.Errorf("variable %s: %w", f.Key, err)
	}

	switch f.Kind {
	case spec.KindString, spec.KindURL, spec.KindEmail, spec.KindEnum:
		return strconv.Quote(lit.Str), nil
	case spec.KindInt, spec.KindPort:
		return strconv.FormatInt(lit.Int, 10), nil
	case spec.KindInt64:
		return "int64(" + strconv.FormatInt(lit.Int, 10) + ")", nil
	case spec.KindFloat64:
		return "float64(" + strconv.FormatFloat(lit.Float, 'g', -1, 64) + ")", nil
	case spec.KindBool:
		return strconv.FormatBool(lit.Bool), nil
	default:
		return "", fmt.Errorf("unknown kind %s for variable %s", f.Kind, f.Key)
	}
}

func (g *goGen) writeConversion(p *printer.Printer, f *parser.Field, target, src string, ptr bool) error {
	if err := g.writeParseAndAssign(p, f, target, src, ptr); err != nil {
		return err
	}

	if err := g.writeValueCheck(p, f, src); err != nil {
		return err
	}

	g.writeConstraints(p, f, src)

	return g.writeAssignment(p, f, target, src, ptr)
}

func (g *goGen) writeParseAndAssign(p *printer.Printer, f *parser.Field, target, src string, ptr bool) error {
	switch f.Kind {
	case spec.KindString, spec.KindURL, spec.KindEmail, spec.KindEnum:

	case spec.KindBool:
		p.Linef("b, convErr := strconv.ParseBool(%s)", src)
		constraints.WriteConvCheck(p, f, "a boolean", src)
		g.assignPointerOrValue(p, target, "b", ptr)

	case spec.KindInt, spec.KindPort:
		p.Linef("n, convErr := strconv.Atoi(%s)", src)
		constraints.WriteConvCheck(p, f, "an integer", src)

		if f.Kind.IsPort() {
			p.Linef("if n < %d || n > %d {", spec.PortMin, spec.PortMax)
			p.Indent()
			constraints.WritePortError(p, f)
			p.Dedent()
			p.Line("}")
		}

		g.assignPointerOrValue(p, target, "n", ptr)

	case spec.KindInt64:
		p.Linef("n, convErr := strconv.ParseInt(%s, 10, 64)", src)
		constraints.WriteConvCheck(p, f, "an int64", src)
		g.assignPointerOrValue(p, target, "n", ptr)

	case spec.KindFloat64:
		p.Linef("fv, convErr := strconv.ParseFloat(%s, 64)", src)
		constraints.WriteConvCheck(p, f, "a float64", src)
		g.assignPointerOrValue(p, target, "fv", ptr)

	default:
		return fmt.Errorf("unknown kind %s for variable %s", f.Kind, f.Key)
	}

	return nil
}

func (g *goGen) assignPointerOrValue(p *printer.Printer, target, name string, ptr bool) {
	if ptr {
		p.Linef("%s = &%s", target, name)
		return
	}

	p.Linef("%s = %s", target, name)
}

func (g *goGen) writeValueCheck(p *printer.Printer, f *parser.Field, src string) error {
	switch f.Kind {
	case spec.KindURL:
		p.Linef("if _, convErr := url.ParseRequestURI(%s); convErr != nil {", src)
		p.Indent()
		constraints.WriteValueError(p, f, "value is not a valid URL", "value %q is not a valid URL", src)
		p.Dedent()
		p.Line("}")

	case spec.KindEmail:
		p.Linef("at := strings.IndexByte(%s, '@')", src)
		p.Linef("dot := strings.LastIndexByte(%s, '.')", src)
		p.Line("if at <= 0 || dot <= at+1 {")
		p.Indent()
		constraints.WriteValueError(p, f, "value is not a valid email", "value %q is not a valid email", src)
		p.Dedent()
		p.Line("}")

	case spec.KindEnum:
		quoted := make([]string, 0, len(f.Enum))

		for _, v := range f.Enum {
			quoted = append(quoted, strconv.Quote(v))
		}

		p.Linef("if !envContains([]string{%s}, %s) {", strings.Join(quoted, ", "), src)
		p.Indent()
		constraints.WriteValueError(p, f, "value is not in the allowed list", "value %q is not in the allowed list ["+strings.Join(f.Enum, ", ")+"]", src)
		p.Dedent()
		p.Line("}")

	case spec.KindString, spec.KindInt, spec.KindInt64, spec.KindBool, spec.KindFloat64, spec.KindPort:
		// nothing to validate

	default:
		return fmt.Errorf("unknown kind %s for variable %s", f.Kind, f.Key)
	}

	return nil
}

func (g *goGen) writeConstraints(p *printer.Printer, f *parser.Field, src string) {
	cs := parser.BuildConstraints(f)
	if len(cs) == 0 {
		return
	}

	for _, c := range cs {
		c.Emit(p, f, src, g.regexVar(f), spec.LangGo)
	}
}

func (g *goGen) writeAssignment(p *printer.Printer, f *parser.Field, target, src string, ptr bool) error {
	switch f.Kind {
	case spec.KindString, spec.KindURL, spec.KindEmail, spec.KindEnum:
		g.assignPointerOrValue(p, target, src, ptr)

	case spec.KindInt, spec.KindInt64, spec.KindBool, spec.KindFloat64, spec.KindPort:
		// assigned in writeParseAndAssign

	default:
		return fmt.Errorf("unknown kind %s for variable %s", f.Kind, f.Key)
	}

	return nil
}

func (g *goGen) regexVar(f *parser.Field) string {
	return naming.LowerFirst(naming.GoFieldName(f.Key)) + "Re"
}
