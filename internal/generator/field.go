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
		switch f.Kind {
		case spec.KindInt, spec.KindPort:
			return "0", nil
		case spec.KindInt64:
			return "int64(0)", nil
		case spec.KindBool:
			return "false", nil
		case spec.KindFloat64:
			return "0", nil
		default:
			return `""`, nil
		}
	}

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
	key := f.Key

	notNumeric := func(what string) {
		if f.Sensitive {
			p.linef("return e, fmt.Errorf(\"env: %s: value is not %s\")", key, what)
			return
		}

		p.linef("return e, fmt.Errorf(\"env: %s: value %%q is not %s: %%w\", %s, convErr)", key, what, src)
	}

	switch f.Kind {
	case spec.KindString, spec.KindURL, spec.KindEmail, spec.KindEnum:

	case spec.KindBool:
		p.linef("b, convErr := strconv.ParseBool(%s)", src)
		p.line("if convErr != nil {")
		p.indent()
		notNumeric("a boolean")
		p.dedent()
		p.line("}")

		if ptr {
			p.linef("%s = &b", target)
		} else {
			p.linef("%s = b", target)
		}

	case spec.KindInt, spec.KindPort:
		p.linef("n, convErr := strconv.Atoi(%s)", src)
		p.line("if convErr != nil {")
		p.indent()
		notNumeric("an integer")
		p.dedent()
		p.line("}")

		if f.Kind.IsPort() {
			p.linef("if n < %d || n > %d {", spec.PortMin, spec.PortMax)
			p.indent()

			if f.Sensitive {
				p.linef("return e, fmt.Errorf(\"env: %s: port is out of range %d..%d\")", key, spec.PortMin, spec.PortMax)
			} else {
				p.linef("return e, fmt.Errorf(\"env: %s: port %%d is out of range %d..%d\", n)", key, spec.PortMin, spec.PortMax)
			}

			p.dedent()
			p.line("}")
		}

		if ptr {
			p.linef("%s = &n", target)
		} else {
			p.linef("%s = n", target)
		}

	case spec.KindInt64:
		p.linef("n, convErr := strconv.ParseInt(%s, 10, 64)", src)
		p.line("if convErr != nil {")
		p.indent()
		notNumeric("an int64")
		p.dedent()
		p.line("}")

		if ptr {
			p.linef("%s = &n", target)
		} else {
			p.linef("%s = n", target)
		}

	case spec.KindFloat64:
		p.linef("fv, convErr := strconv.ParseFloat(%s, 64)", src)
		p.line("if convErr != nil {")
		p.indent()
		notNumeric("a float64")
		p.dedent()
		p.line("}")

		if ptr {
			p.linef("%s = &fv", target)
		} else {
			p.linef("%s = fv", target)
		}
	}

	switch f.Kind {
	case spec.KindURL:
		p.linef("if _, convErr := url.ParseRequestURI(%s); convErr != nil {", src)
		p.indent()

		if f.Sensitive {
			p.linef("return e, fmt.Errorf(\"env: %s: value is not a valid URL\")", key)
		} else {
			p.linef("return e, fmt.Errorf(\"env: %s: value %%q is not a valid URL\", %s)", key, src)
		}

		p.dedent()
		p.line("}")

	case spec.KindEmail:
		p.linef("at := strings.IndexByte(%s, '@')", src)
		p.linef("dot := strings.LastIndexByte(%s, '.')", src)
		p.line("if at <= 0 || dot <= at+1 {")
		p.indent()

		if f.Sensitive {
			p.linef("return e, fmt.Errorf(\"env: %s: value is not a valid email\")", key)
		} else {
			p.linef("return e, fmt.Errorf(\"env: %s: value %%q is not a valid email\", %s)", key, src)
		}

		p.dedent()
		p.line("}")

	case spec.KindEnum:
		quoted := make([]string, 0, len(f.Enum))

		for _, v := range f.Enum {
			quoted = append(quoted, strconv.Quote(v))
		}

		p.linef("if !envContains([]string{%s}, %s) {", strings.Join(quoted, ", "), src)
		p.indent()

		if f.Sensitive {
			p.linef("return e, fmt.Errorf(\"env: %s: value is not in the allowed list\")", key)
		} else {
			p.linef("return e, fmt.Errorf(\"env: %s: value %%q is not in the allowed list [%s]\", %s)",
				key, strings.Join(f.Enum, ", "), src)
		}

		p.dedent()
		p.line("}")

	case spec.KindString:
		// nothing to validate
	}

	if (f.StartsWith != "" || f.HasRegex()) && f.Kind.IsConstrainable() {
		if f.StartsWith != "" {
			p.linef("if !strings.HasPrefix(%s, %q) {", src, f.StartsWith)
			p.indent()

			if f.Sensitive {
				p.linef("return e, fmt.Errorf(\"env: %s: value does not start with the required prefix\")", key)
			} else {
				p.linef("return e, fmt.Errorf(\"env: %s: value %%q does not start with %%q\", %s, %q)", key, src, f.StartsWith)
			}

			p.dedent()
			p.line("}")
		}

		if f.HasRegex() {
			p.linef("if !%s.MatchString(%s) {", g.regexVar(f), src)
			p.indent()

			if f.Sensitive {
				p.linef("return e, fmt.Errorf(\"env: %s: value does not match the required pattern\")", key)
			} else {
				p.linef("return e, fmt.Errorf(\"env: %s: value %%q does not match the required pattern\", %s)", key, src)
			}

			p.dedent()
			p.line("}")
		}
	}

	switch f.Kind {
	case spec.KindString, spec.KindURL, spec.KindEmail, spec.KindEnum:
		if ptr {
			p.linef("%s = &%s", target, src)
		} else {
			p.linef("%s = %s", target, src)
		}
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
