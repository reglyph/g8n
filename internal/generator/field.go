package generator

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/whoqmi/g8n/internal/naming"
	"github.com/whoqmi/g8n/internal/parser"
	"github.com/whoqmi/g8n/internal/spec"
)

func (g *gen) writeFieldLoad(p *printer, f *parser.Field) error {
	target := "e." + g.fieldName(f.Key)
	key := strconv.Quote(f.Key)

	hasDefault := f.HasDefault && !f.Sensitive
	expand := g.expandable(f)
	expandStmt := func(p *printer, src string) {
		if expand {
			p.linef("%s = expandVars(%s, m)", src, src)
		}
	}

	if f.Required {
		if !hasDefault {
			p.line("{")
			p.indent()
			p.linef("v, ok := m[%s]", key)
			p.line("if !ok || v == \"\" {")
			p.indent()
			p.linef("return e, fmt.Errorf(\"env: required variable %%q is missing or empty\", %s)", key)
			p.dedent()
			p.line("}")
			expandStmt(p, "v")
			g.writeConversion(p, f, target, "v", false)
			p.dedent()
			p.line("}")

			return nil
		}

		p.line("{")
		p.indent()
		p.linef("v, ok := m[%s]", key)
		p.line("if !ok || v == \"\" {")
		p.indent()

		if err := g.writeDefaultValue(p, f, target, false); err != nil {
			return err
		}

		p.dedent()
		p.line("} else {")
		p.indent()
		expandStmt(p, "v")
		g.writeConversion(p, f, target, "v", false)
		p.dedent()
		p.line("}")
		p.dedent()
		p.line("}")

		return nil
	}

	p.line("{")
	p.indent()
	p.linef("if v, ok := m[%s]; ok && v != \"\" {", key)
	p.indent()
	expandStmt(p, "v")
	g.writeConversion(p, f, target, "v", true)

	if hasDefault {
		p.dedent()
		p.line("} else {")
		p.indent()

		if err := g.writeDefaultValue(p, f, target, true); err != nil {
			return err
		}
	}

	p.dedent()
	p.line("}")
	p.dedent()
	p.line("}")

	return nil
}

func (g *gen) writeDefaultValue(p *printer, f *parser.Field, target string, ptr bool) error {
	lit, err := g.literal(f)
	if err != nil {
		return err
	}

	if f.Kind.IsStringLike() {
		p.linef("t := %s", g.expandExpr(f, lit))
		g.writeConversion(p, f, target, "t", ptr)

		return nil
	}

	if ptr {
		p.linef("t := %s", lit)
		p.linef("%s = &t", target)

		return nil
	}

	p.linef("%s = %s", target, lit)

	return nil
}

func (g *gen) expandExpr(f *parser.Field, expr string) string {
	if g.expandable(f) {
		return "expandVars(" + expr + ", m)"
	}

	return expr
}

func (g *gen) literal(f *parser.Field) (string, error) {
	if !f.HasDefault {
		return zeroLiteral(f.Kind), nil
	}

	return typedLiteral(f)
}

func zeroLiteral(kind spec.Kind) string {
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

func typedLiteral(f *parser.Field) (string, error) {
	switch f.Kind {
	case spec.KindString, spec.KindURL, spec.KindEmail, spec.KindEnum:
		return strconv.Quote(f.Default), nil
	case spec.KindInt, spec.KindPort:
		n, err := strconv.ParseInt(f.Default, 10, 64)
		if err != nil {
			return "", fmt.Errorf("invalid default %q for variable %s: not an integer", f.Default, f.Key)
		}

		return strconv.FormatInt(n, 10), nil
	case spec.KindInt64:
		n, err := strconv.ParseInt(f.Default, 10, 64)
		if err != nil {
			return "", fmt.Errorf("invalid default %q for variable %s: not an int64", f.Default, f.Key)
		}

		return "int64(" + strconv.FormatInt(n, 10) + ")", nil
	case spec.KindFloat64:
		v, err := strconv.ParseFloat(f.Default, 64)
		if err != nil {
			return "", fmt.Errorf("invalid default %q for variable %s: not a float64", f.Default, f.Key)
		}

		return "float64(" + strconv.FormatFloat(v, 'g', -1, 64) + ")", nil
	case spec.KindBool:
		v, err := strconv.ParseBool(f.Default)
		if err != nil {
			return "", fmt.Errorf("invalid default %q for variable %s: not a boolean", f.Default, f.Key)
		}

		return strconv.FormatBool(v), nil
	default:
		return "", fmt.Errorf("unknown kind %s for variable %s", f.Kind, f.Key)
	}
}

func (g *gen) writeConversion(p *printer, f *parser.Field, target, src string, ptr bool) {
	g.writeParseAndAssign(p, f, target, src, ptr)
	g.writeValueCheck(p, f, src)
	g.writeConstraints(p, f, src)
	g.writeAssignment(p, f, target, src, ptr)
}

func (g *gen) writeParseAndAssign(p *printer, f *parser.Field, target, src string, ptr bool) {
	switch f.Kind {
	case spec.KindString, spec.KindURL, spec.KindEmail, spec.KindEnum:

	case spec.KindBool:
		p.linef("b, convErr := strconv.ParseBool(%s)", src)
		g.writeConvCheck(p, f, "a boolean", src)
		assignPointerOrValue(p, target, "b", ptr)

	case spec.KindInt, spec.KindPort:
		p.linef("n, convErr := strconv.Atoi(%s)", src)
		g.writeConvCheck(p, f, "an integer", src)

		if f.Kind.IsPort() {
			p.linef("if n < %d || n > %d {", spec.PortMin, spec.PortMax)
			p.indent()
			writePortError(p, f)
			p.dedent()
			p.line("}")
		}

		assignPointerOrValue(p, target, "n", ptr)

	case spec.KindInt64:
		p.linef("n, convErr := strconv.ParseInt(%s, 10, 64)", src)
		g.writeConvCheck(p, f, "an int64", src)
		assignPointerOrValue(p, target, "n", ptr)

	case spec.KindFloat64:
		p.linef("fv, convErr := strconv.ParseFloat(%s, 64)", src)
		g.writeConvCheck(p, f, "a float64", src)
		assignPointerOrValue(p, target, "fv", ptr)
	}
}

func (g *gen) writeConvCheck(p *printer, f *parser.Field, what, src string) {
	p.line("if convErr != nil {")
	p.indent()
	writeParseError(p, f, what, src)
	p.dedent()
	p.line("}")
}

func writeParseError(p *printer, f *parser.Field, what, src string) {
	if f.Sensitive {
		p.linef("return e, fmt.Errorf(\"env: %s: value is not %s\")", f.Key, what)
		return
	}

	p.linef("return e, fmt.Errorf(\"env: %s: value %%q is not %s: %%w\", %s, convErr)", f.Key, what, src)
}

func writePortError(p *printer, f *parser.Field) {
	if f.Sensitive {
		p.linef("return e, fmt.Errorf(\"env: %s: port is out of range %d..%d\")", f.Key, spec.PortMin, spec.PortMax)
		return
	}

	p.linef("return e, fmt.Errorf(\"env: %s: port %%d is out of range %d..%d\", n)", f.Key, spec.PortMin, spec.PortMax)
}

func assignPointerOrValue(p *printer, target, name string, ptr bool) {
	if ptr {
		p.linef("%s = &%s", target, name)
		return
	}

	p.linef("%s = %s", target, name)
}

func (g *gen) writeValueCheck(p *printer, f *parser.Field, src string) {
	//goland:noinspection GoSwitchMissingCasesForIotaConsts
	switch f.Kind {
	case spec.KindURL:
		p.linef("if _, convErr := url.ParseRequestURI(%s); convErr != nil {", src)
		p.indent()
		writeValueError(p, f, "value is not a valid URL", "value %q is not a valid URL", src)
		p.dedent()
		p.line("}")

	case spec.KindEmail:
		p.linef("at := strings.IndexByte(%s, '@')", src)
		p.linef("dot := strings.LastIndexByte(%s, '.')", src)
		p.line("if at <= 0 || dot <= at+1 {")
		p.indent()
		writeValueError(p, f, "value is not a valid email", "value %q is not a valid email", src)
		p.dedent()
		p.line("}")

	case spec.KindEnum:
		quoted := make([]string, 0, len(f.Enum))

		for _, v := range f.Enum {
			quoted = append(quoted, strconv.Quote(v))
		}

		p.linef("if !envContains([]string{%s}, %s) {", strings.Join(quoted, ", "), src)
		p.indent()
		writeValueError(p, f, "value is not in the allowed list", "value %q is not in the allowed list ["+strings.Join(f.Enum, ", ")+"]", src)
		p.dedent()
		p.line("}")

	case spec.KindString:
		// nothing to validate
	}
}

func writeValueError(p *printer, f *parser.Field, sensitiveMsg, msg, src string) {
	if f.Sensitive {
		p.linef("return e, fmt.Errorf(\"env: %s: %s\")", f.Key, sensitiveMsg)
		return
	}

	p.linef("return e, fmt.Errorf(\"env: %s: %s\", %s)", f.Key, msg, src)
}

func (g *gen) writeConstraints(p *printer, f *parser.Field, src string) {
	if !f.Kind.IsConstrainable() || (f.StartsWith == "" && !f.HasRegex()) {
		return
	}

	if f.StartsWith != "" {
		p.linef("if !strings.HasPrefix(%s, %q) {", src, f.StartsWith)
		p.indent()
		writeConstraintError(p, f, "value does not start with the required prefix", "value %q does not start with %q", src, strconv.Quote(f.StartsWith))
		p.dedent()
		p.line("}")
	}

	if f.HasRegex() {
		p.linef("if !%s.MatchString(%s) {", g.regexVar(f), src)
		p.indent()
		writeConstraintError(p, f, "value does not match the required pattern", "value %q does not match the required pattern", src, "")
		p.dedent()
		p.line("}")
	}
}

func writeConstraintError(p *printer, f *parser.Field, sensitiveMsg, msg, src, extra string) {
	if f.Sensitive {
		p.linef("return e, fmt.Errorf(\"env: %s: %s\")", f.Key, sensitiveMsg)
		return
	}

	if extra != "" {
		p.linef("return e, fmt.Errorf(\"env: %s: %s\", %s, %s)", f.Key, msg, src, extra)
		return
	}

	p.linef("return e, fmt.Errorf(\"env: %s: %s\", %s)", f.Key, msg, src)
}

func (g *gen) writeAssignment(p *printer, f *parser.Field, target, src string, ptr bool) {
	//goland:noinspection GoSwitchMissingCasesForIotaConsts
	switch f.Kind {
	case spec.KindString, spec.KindURL, spec.KindEmail, spec.KindEnum:
		assignPointerOrValue(p, target, src, ptr)
	}
}

func (g *gen) regexVar(f *parser.Field) string {
	return lowerFirst(naming.GoFieldName(f.Key)) + "Re"
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}

	r, size := utf8.DecodeRuneInString(s)

	return strings.ToLower(string(r)) + s[size:]
}
