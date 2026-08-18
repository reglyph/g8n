package typescript

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/reglyph/g8n/internal/generator/core"
	"github.com/reglyph/g8n/internal/generator/fixtures"
	"github.com/reglyph/g8n/internal/parser"
	"github.com/reglyph/g8n/internal/spec"
)

var update = flag.Bool("update", false, "rewrite golden files")

func TestTSZeroLiteral(t *testing.T) {
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
		{spec.KindInt64, "0"},
		{spec.KindBool, "false"},
		{spec.KindFloat64, "0"},
	}

	for _, c := range cases {
		f := &parser.Field{Key: "K", Kind: c.kind}

		if got, _ := core.Literal(f, spec.LangTS); got != c.want {
			t.Errorf("zero literal for %v = %q, want %q", c.kind, got, c.want)
		}
	}
}

func TestTSLiteral(t *testing.T) {
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
		{"int64", field(spec.KindInt64, "67"), "67", false},
		{"int64 bad", field(spec.KindInt64, "x"), "", true},
		{"float", field(spec.KindFloat64, "0.5"), "0.5", false},
		{"float bad", field(spec.KindFloat64, "x"), "", true},
		{"bool canonical", field(spec.KindBool, "TRUE"), "true", false},
		{"bool bad", field(spec.KindBool, "maybe"), "", true},
		{"unknown kind", field(spec.Kind(99), "x"), "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := core.Literal(c.f, spec.LangTS)

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

func TestTSFieldName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"DB_HOST", "dbHost"},
		{"ENV", "envValue"},
	}

	for _, c := range cases {
		g := &tsGen{}

		if got := g.fieldName(c.in); got != c.want {
			t.Errorf("fieldName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTSEnumNames(t *testing.T) {
	g := &tsGen{}

	f := &parser.Field{Key: "APP_ENV"}
	if got := g.enumValuesName(f); got != "appEnvValues" {
		t.Errorf("enumValuesName = %q, want appEnvValues", got)
	}

	if got := g.enumTypeName(f); got != "AppEnv" {
		t.Errorf("enumTypeName = %q, want AppEnv", got)
	}
}

func TestTSCheckFieldNameCollisions(t *testing.T) {
	fieldName := (&tsGen{}).fieldName

	if err := core.CheckFieldNameCollisions(&parser.Schema{Fields: []*parser.Field{
		{Key: "A", Line: 1},
		{Key: "B", Line: 2},
	}}, fieldName); err != nil {
		t.Fatalf("no collision expected, got %v", err)
	}

	err := core.CheckFieldNameCollisions(&parser.Schema{Fields: []*parser.Field{
		{Key: "DB_HOST", Line: 1},
		{Key: "db_host", Line: 2},
	}}, fieldName)
	if err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("want collision error, got %v", err)
	}
}

func TestTSComputeUses(t *testing.T) {
	g := &tsGen{schema: &parser.Schema{Fields: []*parser.Field{
		{Key: "A", Kind: spec.KindString},
		{Key: "B", Kind: spec.KindEnum, Enum: []string{"x"}},
		{Key: "C", Kind: spec.KindInt},
		{Key: "D", Kind: spec.KindURL},
		{Key: "E", Kind: spec.KindString, Sensitive: true},
		{Key: "F", Kind: spec.KindString, Required: true, HasDefault: true},
	}}}

	if err := g.computeUses(); err != nil {
		t.Fatal(err)
	}

	for _, k := range []string{"int", "url"} {
		if !g.uses[k] {
			t.Errorf("uses[%q] = false, want true", k)
		}
	}

	if !g.hasEnum {
		t.Error("hasEnum = false, want true")
	}

	if !g.hasExpand {
		t.Error("hasExpand = false, want true")
	}
}

func TestGenerateTSGolden(t *testing.T) {
	for _, c := range []struct {
		name, schema, golden string
	}{
		{"basic", fixtures.BasicSchema, "testdata/basic.golden"},
		{"constraints", fixtures.ConstraintsSchema, "testdata/constraints.golden"},
		{"minimal", fixtures.MinimalSchema, "testdata/minimal.golden"},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, err := parser.ParseString(c.name+".env.schema", c.schema)
			if err != nil {
				t.Fatal(err)
			}

			out, err := Generate(s)
			if err != nil {
				t.Fatal(err)
			}

			if *update {
				// #nosec G306 -- generated source must stay readable by all users of the repo
				if err := os.WriteFile(c.golden, out, 0o644); err != nil {
					t.Fatal(err)
				}

				return
			}

			want, err := os.ReadFile(c.golden)
			if err != nil {
				t.Fatalf("missing golden file %s (run with -update): %v", c.golden, err)
			}

			if string(out) != string(want) {
				t.Errorf("generated output mismatch\n--- got ---\n%s\n--- want ---\n%s", out, want)
			}
		})
	}
}

func TestGenerateTSOutputTypeChecks(t *testing.T) {
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not available")
	}

	for _, c := range []struct{ name, schema string }{
		{"basic", fixtures.BasicSchema},
		{"constraints", fixtures.ConstraintsSchema},
		{"minimal", fixtures.MinimalSchema},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, err := parser.ParseString(c.name+".env.schema", c.schema)
			if err != nil {
				t.Fatal(err)
			}

			out, err := Generate(s)
			if err != nil {
				t.Fatal(err)
			}

			tsCheck(t, out)
		})
	}
}

func tsCheck(t *testing.T, src []byte) {
	t.Helper()

	dir := t.TempDir()
	file := filepath.Join(dir, "env.ts")

	if err := os.WriteFile(file, src, 0o600); err != nil {
		t.Fatal(err)
	}

	// #nosec G204 -- fixed args, only the temp file path varies
	cmd := exec.Command("npx", "-p", "typescript", "tsc", "--strict", "--noEmit", file)
	out, err := cmd.CombinedOutput()

	if err != nil && !strings.Contains(string(out), "could not be found") {
		t.Fatalf("generated code does not type-check: %v\n%s", err, out)
	}
}
