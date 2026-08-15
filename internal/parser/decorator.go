package parser

import (
	"strings"

	"github.com/reglyph/g8n/internal/parser/constraints"
	"github.com/reglyph/g8n/internal/spec"
)

// decorator describes one field decorator: which syntaxes it supports and
// how it is applied. Constraint-style decorators additionally carry the
// validation, emission and JSON Schema logic.
type decorator struct {
	name string

	// supported forms
	standalone  bool // @name=value as the whole line
	token       bool // @name=value among other tokens on one line
	option      bool // name=value inside @type=(...)
	singleToken bool // standalone form requires exactly one token

	apply    func(f *Field, val string, warn warnf) // standalone and token forms
	applyOpt func(f *Field, val string, warn warnf) // @type option form; falls back to apply

	constraint Constraint
}

// decorators is the registry of field decorators. Standalone matching order
// matters and must be preserved when adding entries.
var decorators []*decorator

// BuildConstraints returns the constraints present on the field, in the
// order their checks must run: startsWith, regex, then min and max.
func BuildConstraints(f *Field) []Constraint {
	var out []Constraint

	for _, d := range decorators {
		if d.constraint == nil {
			continue
		}

		if c := d.constraint; c.Present(f) {
			out = append(out, c)
		}
	}

	return out
}

func init() {
	decorators = []*decorator{
		{name: "docs", standalone: true, token: true, apply: func(f *Field, val string, warn warnf) {
			if val != "" {
				f.Docs = append(f.Docs, val)
			}
		}},
		{name: "default", standalone: true, token: true, apply: func(f *Field, val string, warn warnf) {
			if val != "" {
				f.Default = val
				f.HasDefault = true
			}
		}},
		{name: "startsWith", standalone: true, option: true,
			apply:      func(f *Field, val string, warn warnf) { applyStringConstraint(f, "@startsWith", val, warn) },
			applyOpt:   func(f *Field, val string, warn warnf) { applyStringConstraint(f, "startsWith", val, warn) },
			constraint: constraints.StartsWith(),
		},
		{name: "regex", standalone: true, option: true,
			apply:      func(f *Field, val string, warn warnf) { applyStringConstraint(f, "@regex", val, warn) },
			applyOpt:   func(f *Field, val string, warn warnf) { applyStringConstraint(f, "regex", val, warn) },
			constraint: constraints.Regex(),
		},
		{name: "type", standalone: true, singleToken: true, token: true,
			apply: func(f *Field, val string, warn warnf) { applyType(f, val, warn) },
		},
		{name: "required", token: true, apply: func(f *Field, val string, warn warnf) {
			f.Required = true
		}},
		{name: "sensitive", token: true, apply: func(f *Field, val string, warn warnf) {
			f.Sensitive = true
		}},
	}
}

func lookupDecorator(name string) *decorator {
	for _, d := range decorators {
		if d.name == name {
			return d
		}
	}

	return nil
}

type warnf func(message string, args ...any)

func isComment(line string) bool {
	return strings.HasPrefix(line, "#")
}

func isSeparator(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed != "" && strings.Trim(trimmed, "-") == ""
}

func isDecorator(line string) bool {
	tokens := strings.Fields(line)

	if len(tokens) == 0 {
		return false
	}

	return strings.HasPrefix(tokens[0], "@")
}

func isRootDecorator(line string) bool {
	if !isDecorator(line) {
		return false
	}

	for _, token := range strings.Fields(line) {
		name := decoratorName(token)

		if name != "@package" && name != "@out" {
			return false
		}
	}

	return true
}

func decoratorName(token string) string {
	eq := strings.IndexByte(token, '=')
	paren := strings.IndexByte(token, '(')

	switch {
	case eq < 0:
		if paren < 0 {
			return token
		}

		return token[:paren]
	case paren < 0:
		return token[:eq]
	case paren < eq:
		return token[:paren]
	default:
		return token[:eq]
	}
}

func applyCommentBlock(s *Schema, pending []string, f *Field, warn warnf) {
	for _, body := range pending {
		switch {
		case isRootDecorator(body):
			parseRootDecorators(s, body, warn)
		case isDecorator(body):
			parseFieldDecorators(f, body, warn)
		default:
			f.Docs = append(f.Docs, body)
		}
	}
}

func parseRootDecorators(s *Schema, body string, warn warnf) {
	for _, tok := range splitDecoratorTokens(body) {
		switch {
		case tok == "@package":
			warn("@package required a value")
		case strings.HasPrefix(tok, "@package="):
			s.Package = strings.TrimSpace(strings.TrimPrefix(tok, "@package="))
		case tok == "@out" || strings.HasPrefix(tok, "@out("):
			parseOut(s, tok, warn)
		}
	}
}

func parseOut(s *Schema, tok string, warn warnf) {
	inner := strings.TrimPrefix(tok, "@out")
	inner = strings.TrimSpace(strings.TrimPrefix(inner, "("))
	inner = strings.TrimSuffix(inner, ")")

	if inner == "" {
		warn("@out requires a path parameter, e.g. @out(path=internal/config/env.go)")
		return
	}

	for _, param := range splitParams(inner) {
		kv := strings.SplitN(strings.TrimSpace(param), "=", 2)

		if len(kv) != 2 {
			continue
		}

		switch strings.TrimSpace(kv[0]) {
		case "path":
			s.OutPath = strings.TrimSpace(kv[1])
		case "package":
			s.Package = strings.TrimSpace(kv[1])
		}
	}

	if s.OutPath == "" {
		warn("@out requires a path parameter, e.g. @out(path=...)")
	}
}

func parseFieldDecorators(f *Field, body string, warn warnf) {
	if parseSingleDecorator(f, body, warn) {
		return
	}

	for _, tok := range splitDecoratorTokens(body) {
		if !strings.HasPrefix(tok, "@") {
			continue
		}

		name, arg := tok, ""
		if i := strings.IndexByte(tok, '='); i >= 0 {
			name, arg = tok[:i], tok[i+1:]
		}

		d := lookupDecorator(strings.TrimPrefix(name, "@"))

		if d == nil || !d.token {
			warn("unknown decorator %s", name)
			continue
		}

		d.apply(f, arg, warn)
	}
}

// parseSingleDecorator handles decorators that take the whole line as their
// argument. It reports whether the body was fully consumed.
func parseSingleDecorator(f *Field, body string, warn warnf) bool {
	for _, d := range decorators {
		if !d.standalone {
			continue
		}

		prefix := "@" + d.name + "="

		if !strings.HasPrefix(body, prefix) {
			continue
		}

		if d.singleToken && len(splitDecoratorTokens(body)) != 1 {
			continue
		}

		val := strings.TrimSpace(strings.TrimPrefix(body, prefix))
		d.apply(f, val, warn)

		return true
	}

	return false
}

// splitParenAware splits body on runes for which split reports true, ignoring
// separators nested inside parentheses.
func splitParenAware(body string, split func(rune) bool) []string {
	var tokens []string

	var cur strings.Builder

	depth := 0

	for _, r := range body {
		switch {
		case r == '(':
			depth++

			cur.WriteRune(r)
		case r == ')':
			if depth > 0 {
				depth--
			}

			cur.WriteRune(r)

		case split(r) && depth == 0:
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}

	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}

	return tokens
}

func splitDecoratorTokens(body string) []string {
	return splitParenAware(body, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\r' || r == '\n'
	})
}

func splitParams(s string) []string {
	return splitParenAware(s, func(r rune) bool { return r == ',' })
}

func applyType(f *Field, arg string, warn warnf) {
	name, opts := arg, ""

	if i := strings.IndexByte(arg, '('); i >= 0 {
		if !strings.HasSuffix(arg, ")") {
			warn("@type=%q, expected closing ')'", arg)
			return
		}

		name, opts = arg[:i], arg[i+1:len(arg)-1]
	}

	kind, known := spec.ParseKind(name)

	if !known {
		warn("unknown @type=%q, falling back to string", name)
	}

	f.Kind = kind

	if kind == spec.KindEnum {
		f.Enum = nil

		for _, part := range splitParams(opts) {
			if part = strings.TrimSpace(part); part != "" {
				f.Enum = append(f.Enum, part)
			}
		}

		if len(f.Enum) == 0 {
			warn("@type=enum(...) requires a comma-separated list of values")
		}

		return
	}

	if opts != "" {
		applyTypeOptions(f, opts, warn)
	}
}

func applyTypeOptions(f *Field, options string, warn warnf) {
	for _, opt := range splitParams(options) {
		kv := strings.SplitN(strings.TrimSpace(opt), "=", 2)

		if len(kv) != 2 {
			warn("ignoring unsupported @type option %q", strings.TrimSpace(opt))
			continue
		}

		k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])

		d := lookupDecorator(k)

		if d == nil || !d.option {
			warn("unknown @type option %q", k)
			continue
		}

		apply := d.apply

		if d.applyOpt != nil {
			apply = d.applyOpt
		}

		apply(f, v, warn)
	}
}

func applyStringConstraint(f *Field, decorator, val string, warn warnf) {
	if !f.Kind.IsConstrainable() {
		warn("%s is only supported for string-like types, ignoring for %q", decorator, f.Kind)
		return
	}

	if val == "" {
		warn("%s requires a non-empty value", decorator)
		return
	}

	switch strings.TrimPrefix(decorator, "@") {
	case "startsWith":
		f.StartsWith = val
	case "regex":
		f.Regex = val
	}
}
