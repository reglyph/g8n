package naming

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// GoFieldName converts an env key into a Go identifier (CamelCase).
func GoFieldName(key string) string {
	var b strings.Builder

	var part []rune

	flush := func() {
		if len(part) == 0 {
			return
		}

		for i, r := range part {
			part[i] = unicode.ToLower(r)
		}

		if unicode.IsLetter(part[0]) {
			part[0] = unicode.ToUpper(part[0])
		}

		b.WriteString(string(part))
		part = part[:0]
	}

	for _, r := range key {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			part = append(part, unicode.ToLower(r))
		} else {
			flush()
		}
	}

	flush()

	return b.String()
}

// TSFieldName converts an env key into a TS identifier (camelCase).
func TSFieldName(key string) string {
	return LowerFirst(GoFieldName(key))
}

// UpperFirst raises the first rune of an identifier and keeps the rest as is.
func UpperFirst(s string) string {
	if s == "" {
		return s
	}

	r, size := utf8.DecodeRuneInString(s)

	return strings.ToUpper(string(r)) + s[size:]
}

// LowerFirst lowers the first rune of an identifier and keeps the rest as is.
func LowerFirst(s string) string {
	if s == "" {
		return s
	}

	r, size := utf8.DecodeRuneInString(s)

	return strings.ToLower(string(r)) + s[size:]
}
