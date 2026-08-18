// Package core holds logic shared by the language generator packages.
package core

import (
	"fmt"
	"strconv"

	"github.com/reglyph/g8n/internal/parser"
	"github.com/reglyph/g8n/internal/spec"
)

// Expandable reports whether the field value supports ${...} expansion.
func Expandable(f *parser.Field) bool {
	if f.Sensitive || f.HasRegex() {
		return false
	}

	return f.Kind.IsStringLike()
}

// ExpandExpr wraps expr in an expandVars call when the field is expandable.
func ExpandExpr(f *parser.Field, expr string) string {
	if Expandable(f) {
		return "expandVars(" + expr + ", m)"
	}

	return expr
}

// Literal renders the field's default value as a literal in the given language.
func Literal(f *parser.Field, lang spec.Lang) (string, error) {
	if !f.HasDefault {
		return zeroLiteral(f.Kind, lang), nil
	}

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
		return castSigned(lit.Int, lang), nil

	case spec.KindFloat64:
		return castFloat(lit.Float, lang), nil

	case spec.KindBool:
		return strconv.FormatBool(lit.Bool), nil

	default:
		return "", fmt.Errorf("unknown kind %s for variable %s", f.Kind, f.Key)
	}
}

func zeroLiteral(kind spec.Kind, lang spec.Lang) string {
	switch kind {
	case spec.KindInt, spec.KindPort:
		return "0"
	case spec.KindInt64:
		return castSigned(0, lang)
	case spec.KindBool:
		return "false"
	case spec.KindFloat64:
		return "0"
	default:
		return `""`
	}
}

// castSigned prefixes the int64 literal with an int64() cast for Go, which requires one for 64-bit integer constants; TS needs none.
func castSigned(n int64, lang spec.Lang) string {
	s := strconv.FormatInt(n, 10)

	if lang == spec.LangGo {
		return "int64(" + s + ")"
	}

	return s
}

// castFloat prefixes the float literal with a float64() cast for Go; TS needs none.
func castFloat(f float64, lang spec.Lang) string {
	s := strconv.FormatFloat(f, 'g', -1, 64)

	if lang == spec.LangGo {
		return "float64(" + s + ")"
	}

	return s
}
