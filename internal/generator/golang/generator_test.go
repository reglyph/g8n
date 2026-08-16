package golang

import (
	"flag"
	"go/ast"
	"go/importer"
	goparser "go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"
	"testing"

	"github.com/reglyph/g8n/internal/generator/fixtures"
	"github.com/reglyph/g8n/internal/parser"
)

var update = flag.Bool("update", false, "rewrite golden files")

const basicSchema = fixtures.BasicSchema

const constraintsSchema = fixtures.ConstraintsSchema

const minimalSchema = fixtures.MinimalSchema

func TestGenerateGolden(t *testing.T) {
	for _, c := range []struct {
		name, schema, golden string
	}{
		{"basic", basicSchema, "testdata/basic.golden"},
		{"constraints", constraintsSchema, "testdata/constraints.golden"},
		{"minimal", minimalSchema, "testdata/minimal.golden"},
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

func TestGenerateGoldenOutputTypeChecks(t *testing.T) {
	for _, c := range []struct {
		name, schema string
	}{
		{"basic", basicSchema},
		{"constraints", constraintsSchema},
		{"minimal", minimalSchema},
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

			typeCheck(t, out)
		})
	}
}

func TestGenerateRequiresPackage(t *testing.T) {
	s, err := parser.ParseString("test.env.schema", "A=1\n")
	if err != nil {
		t.Fatal(err)
	}

	_, err = Generate(s)
	if err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("want missing package error, got %v", err)
	}
}

func TestGenerateFieldNameCollision(t *testing.T) {
	s, err := parser.ParseString("test.env.schema", "# @package=env\nDB_HOST=a\ndb_host=b\n")
	if err != nil {
		t.Fatal(err)
	}

	_, err = Generate(s)
	if err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("want collision error, got %v", err)
	}
}

func typeCheck(t *testing.T, src []byte) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, "env.go", src, goparser.AllErrors)
	if err != nil {
		t.Fatalf("generated code does not parse: %v\n%s", err, src)
	}

	//goland:noinspection GoDeprecation && TODO
	conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil)}
	if _, err := conf.Check("env", fset, []*ast.File{f}, nil); err != nil {
		t.Fatalf("generated code does not type-check: %v\n%s", err, src)
	}
}
