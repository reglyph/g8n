package parser

import (
	"strings"

	"github.com/whoqmi/g8n/internal/spec"
)

type warnf func(message string, args ...any)

func (f *Field) HasRegex() bool {
	return f.Regex != ""
}

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
		case tok == "@out" || strings.HasPrefix(tok, "@out="):
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

	for _, param := range strings.Split(inner, ",") {
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
	if strings.HasPrefix(body, "@docs=") {
		if v := strings.TrimSpace(strings.TrimPrefix(body, "@docs=")); v != "" {
			f.Docs = append(f.Docs, v)
		}

		return
	}

	if strings.HasPrefix(body, "@default=") {
		if v := strings.TrimSpace(strings.TrimPrefix(body, "@default=")); v != "" {
			f.Default = v
			f.HasDefault = true
		}

		return
	}

	if strings.HasPrefix(body, "@startsWith=") {
		applyStringConstraint(f, "@startsWith", strings.TrimSpace(strings.TrimPrefix(body, "@startsWith=")), warn)
		return
	}

	if strings.HasPrefix(body, "@regex=") {
		applyStringConstraint(f, "@regex", strings.TrimSpace(strings.TrimPrefix(body, "@regex=")), warn)
		return
	}

	if strings.HasPrefix(body, "@type=") && len(splitDecoratorTokens(body)) == 1 {
		applyType(f, strings.TrimSpace(strings.TrimPrefix(body, "@type=")), warn)
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

		switch name {
		case "@required":
			f.Required = true
		case "@sensitive":
			f.Sensitive = true
		case "@default":
			if arg != "" {
				f.Default = arg
				f.HasDefault = true
			}
		case "@docs":
			if arg != "" {
				f.Docs = append(f.Docs, arg)
			}
		default:
			warn("unknown decorator %s", name)
		}
	}
}

func splitDecoratorTokens(body string) []string {
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
			cur.WriteRune(')')
		case (r == ' ' || r == '\t' || r == '\n' || r == '\r') && depth == 0:
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

		for _, part := range strings.Split(name, ",") {
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
		// todo: apply options
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

	switch decorator {
	case "@startsWith":
		f.StartsWith = val
	case "@regex":
		f.Regex = val
	}
}
