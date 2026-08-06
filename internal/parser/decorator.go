package parser

import "strings"

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
		case tok == "@out":
			// todo: parse out path
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
