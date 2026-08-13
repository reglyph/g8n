package generator

import (
	"strings"
	"testing"

	"github.com/reglyph/g8n/internal/parser"
	"github.com/reglyph/g8n/internal/spec"
)

func TestZeroLiteral(t *testing.T) {
	cases := []struct {
		kind spec.Kind
		want string
	}{
		{spec.KindString, `""`},
		{spec.KindURL, `""`},
		{spec.KindEmail, `""`},
		{spec.KindEnum, `""`},
		{spec.KindInt, "0"},
		{spec.KindPort, "0"},
		{spec.KindInt64, "int64(0)"},
		{spec.KindBool, "false"},
		{spec.KindFloat64, "0"},
	}

	for _, c := range cases {
		if got := zeroLiteral(c.kind); got != c.want {
			t.Errorf("zeroLiteral(%v) = %q, want %q", c.kind, got, c.want)
		}
	}
}

func TestLiteral(t *testing.T) {
	field := func(kind spec.Kind, def string) *parser.Field {
		return &parser.Field{Key: "K", Kind: kind, Default: def, HasDefault: def != ""}
	}

	cases := []struct {
		name string
		f    *parser.Field
		want string
		err  bool
	}{
		{"string", field(spec.KindString, "hello"), `"hello"`, false},
		{"url", field(spec.KindURL, "http://reglyph.dev"), `"http://reglyph.dev"`, false},
		{"email", field(spec.KindEmail, "a@b.co"), `"a@b.co"`, false},
		{"enum", field(spec.KindEnum, "dev"), `"dev"`, false},
		{"int canonical", field(spec.KindInt, "08"), "8", false},
		{"int bad", field(spec.KindInt, "abc"), "", true},
		{"port", field(spec.KindPort, "8080"), "8080", false},
		{"int64", field(spec.KindInt64, "67"), "int64(67)", false},
		{"int64 bad", field(spec.KindInt64, "x"), "", true},
		{"float", field(spec.KindFloat64, "0.5"), "float64(0.5)", false},
		{"float bad", field(spec.KindFloat64, "x"), "", true},
		{"bool canonical", field(spec.KindBool, "TRUE"), "true", false},
		{"bool bad", field(spec.KindBool, "maybe"), "", true},
		{"unknown kind", field(spec.Kind(99), "x"), "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := &gen{}
			got, err := g.literal(c.f)

			if c.err && err == nil {
				t.Fatalf("expected error, got %q", got)
			}

			if !c.err && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != c.want {
				t.Errorf("literal = %q, want %q", got, c.want)
			}
		})
	}
}

func TestLiteralNoDefault(t *testing.T) {
	g := &gen{}
	f := &parser.Field{Key: "K", Kind: spec.KindString}
	if got, err := g.literal(f); err != nil || got != `""` {
		t.Errorf("literal without default = %q, %v; want %q", got, err, `""`)
	}
}

func TestExpandable(t *testing.T) {
	str := &parser.Field{Key: "K", Kind: spec.KindString}

	cases := []struct {
		name string
		f    *parser.Field
		want bool
	}{
		{"string", str, true},
		{"url", &parser.Field{Key: "K", Kind: spec.KindURL}, true},
		{"enum", &parser.Field{Key: "K", Kind: spec.KindEnum}, true},
		{"int", &parser.Field{Key: "K", Kind: spec.KindInt}, false},
		{"sensitive", &parser.Field{Key: "K", Kind: spec.KindString, Sensitive: true}, false},
		{"regex", &parser.Field{Key: "K", Kind: spec.KindString, Regex: "^x"}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := &gen{}

			if got := g.expandable(c.f); got != c.want {
				t.Errorf("expandable = %v, want %v", got, c.want)
			}
		})
	}
}

func TestFieldName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"DB_HOST", "DbHost"},
		{"ENV", "EnvValue"},
		{"SENSITIVE_KEYS", "SensitiveKeysValue"},
		{"LOAD", "LoadValue"},
		{"LOAD_FROM", "LoadFromValue"},
		{"EXPAND_VARS", "ExpandVarsValue"},
	}

	for _, c := range cases {
		g := &gen{}

		if got := g.fieldName(c.in); got != c.want {
			t.Errorf("fieldName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCheckFieldNameCollisions(t *testing.T) {
	g := &gen{schema: &parser.Schema{Fields: []*parser.Field{
		{Key: "A", Line: 1},
		{Key: "B", Line: 2},
	}}}

	if err := g.checkFieldNameCollisions(); err != nil {
		t.Fatalf("no collision expected, got %v", err)
	}

	g = &gen{schema: &parser.Schema{Fields: []*parser.Field{
		{Key: "DB_HOST", Line: 1},
		{Key: "db_host", Line: 2},
	}}}

	err := g.checkFieldNameCollisions()
	if err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("want collision error, got %v", err)
	}
}

func TestComputeUses(t *testing.T) {
	g := &gen{schema: &parser.Schema{Fields: []*parser.Field{
		{Key: "A", Kind: spec.KindString},
		{Key: "B", Kind: spec.KindEnum, Enum: []string{"x"}},
		{Key: "C", Kind: spec.KindInt},
		{Key: "D", Kind: spec.KindURL},
		{Key: "E", Kind: spec.KindString, Regex: "^x"},
		{Key: "F", Kind: spec.KindString, Sensitive: true},
		{Key: "G", Kind: spec.KindString, Required: true, HasDefault: true},
	}}}

	if err := g.computeUses(); err != nil {
		t.Fatal(err)
	}

	for _, imp := range []string{"os", "strings", "strconv", "net/url", "regexp", "fmt"} {
		if !g.uses[imp] {
			t.Errorf("uses[%q] = false, want true", imp)
		}
	}

	if !g.hasEnum {
		t.Error("hasEnum = false, want true")
	}

	if !g.hasExpand {
		t.Error("hasExpand = false, want true")
	}
}

func TestRegexVarAndLowerFirst(t *testing.T) {
	g := &gen{}

	if got := g.regexVar(&parser.Field{Key: "API_TOKEN"}); got != "apiTokenRe" {
		t.Errorf("regexVar(API_TOKEN) = %q, want apiTokenRe", got)
	}

	if got := lowerFirst("Hello"); got != "hello" {
		t.Errorf("lowerFirst(Hello) = %q, want hello", got)
	}

	if got := lowerFirst(""); got != "" {
		t.Errorf("lowerFirst(\"\") = %q, want empty", got)
	}
}
