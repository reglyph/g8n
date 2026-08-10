package generator

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

	"github.com/reglyph/g8n/internal/parser"
)

var update = flag.Bool("update", false, "rewrite golden files")

const basicSchema = `# @package=env

# Host of the database
# @type=string @required
DB_HOST=db

# @type=port
# @default=5432
DB_PORT=

# @type=enum(dev,staging,prod)
# @default=dev
APP_ENV=

# @required @sensitive
API_KEY=

# @type=url
# @default=http://localhost
API_URL=

# @type=int64
# @default=1048576
MAX_UPLOAD_BYTES=

# @type=float64
# @default=0.5
JITTER=

# @type=bool
# @default=false
DEBUG=

# @type=string
# @default=hello
GREETING=

# @type=email
ADMIN_EMAIL=

OPTIONAL_NO_DEFAULT=

# @type=string
# @default=${DB_HOST}-primary
APP_URL=
`

const constraintsSchema = `# @package=env

# @type=string(startsWith=sk_)
# @default=sk_main
SERVICE=

# @type=string(regex=^[a-f0-9]{32}$)
TOKEN=
`

const minimalSchema = `# @package=env
A=1
`

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

	conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil)}
	if _, err := conf.Check("env", fset, []*ast.File{f}, nil); err != nil {
		t.Fatalf("generated code does not type-check: %v\n%s", err, src)
	}
}
