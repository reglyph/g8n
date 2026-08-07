package parser

import (
	"github.com/whoqmi/g8n/internal/spec"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseBasic(t *testing.T) {
	src := `# @package=config
# @out(path=internal/config/config.go)

# Host of the database
# @type=string @required
DB_HOST=db

# @type=port
DB_PORT=8080

# @type=enum(dev,staging,prod)
APP_ENV=dev

# A bool toggle
# @type=bool
FEATURE=false

# @type=int64
LIMIT=67

# @type=url
# @docs=a documentation link
SITE=https://whoqmi.me/
`
	s, err := ParseString("", src)
	if err != nil {
		t.Fatalf("ParseString: %v", err)
	}

	if s.Package != "config" {
		t.Errorf("Package = %q, want config", s.Package)
	}

	if s.OutPath != "internal/config/config.go" {
		t.Errorf("OutPath = %q, want internal/config/config.go", s.OutPath)
	}

	if got := len(s.Fields); got != 6 {
		t.Fatalf("got %d fields, want 6", got)
	}

	db := s.FieldByKey("DB_HOST")
	if db == nil {
		t.Fatal("DB_HOST missing")
	}

	if db.Kind != spec.KindString {
		t.Errorf("DB_HOST kind = %v, want string", db.Kind)
	}

	if !db.Required {
		t.Error("DB_HOST should be required")
	}

	if db.Sensitive {
		t.Error("DB_HOST should not be sensitive")
	}

	if !db.HasDefault || db.Default != "db" {
		t.Errorf("DB_HOST default = (has=%v,val=%q), want db", db.HasDefault, db.Default)
	}

	if len(db.Docs) != 1 || db.Docs[0] != "Host of the database" {
		t.Errorf("DB_HOST docs = %v", db.Docs)
	}

	port := s.FieldByKey("DB_PORT")
	if port.Kind != spec.KindPort {
		t.Errorf("DB_PORT kind = %v, want port", port.Kind)
	}

	env := s.FieldByKey("APP_ENV")
	if env.Kind != spec.KindEnum {
		t.Errorf("APP_ENV kind = %v, want enum", env.Kind)
	}

	if len(env.Enum) != 3 || env.Enum[1] != "staging" {
		t.Errorf("APP_ENV enum values = %v", env.Enum)
	}

	feat := s.FieldByKey("FEATURE")
	if feat.Kind != spec.KindBool || feat.Default != "false" {
		t.Errorf("FEATURE = kind %v default %q", feat.Kind, feat.Default)
	}

	limit := s.FieldByKey("LIMIT")
	if limit.Kind != spec.KindInt64 {
		t.Errorf("LIMIT kind = %v, want int64", limit.Kind)
	}

	site := s.FieldByKey("SITE")
	if site.Kind != spec.KindURL {
		t.Errorf("SITE kind = %v, want url", site.Kind)
	}

	if len(site.Docs) != 1 || site.Docs[0] != "a documentation link" {
		t.Errorf("SITE docs = %v", site.Docs)
	}
}

func TestParseEmpty(t *testing.T) {
	s, err := ParseString("", "")

	if err != nil {
		t.Fatal(err)
	}

	if len(s.Fields) > 0 {
		t.Errorf("expected 0 fields, got %d", len(s.Fields))
	}
}

func TestParseInvalidLine(t *testing.T) {
	_, err := ParseString("", "just a line\n")

	if err == nil || !strings.Contains(err.Error(), "expected KEY=VALUE") {
		t.Fatalf("want mailformed line error, got %v", err)
	}
}

func TestParseDuplicateLine(t *testing.T) {
	s, err := ParseString("", "A=1\nA=2")

	if err != nil {
		t.Fatal(err)
	}

	if len(s.Fields) > 1 {
		t.Fatalf("got %d lines, want 1 (-duplicate)", len(s.Fields))
	}

	if s.FieldByKey("A").Default != "1" {
		t.Errorf("first declaration should win")
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "already present") {
		t.Errorf("want duplicate warning, got %v", s.Warnings)
	}
}

func TestParseDefaultValidation(t *testing.T) {
	for _, c := range []struct {
		Name, Source string
		Ok           bool
	}{
		{"BadPort", "# @type=port\nP=99999\n", false},
		{"GoodPort", "# @type=port\nP=8080\n", true},
		{"BadInt", "# @type=int\nN=abc\n", false},
		{"GoodInt", "# @type=int\nN=67\n", true},
		{"BadEnum", "# @type=enum(a,b)\nE=c\n", false},
		{"GoodEnum", "# @type=enum(a,b)\nE=a\n", true},
		{"BadURL", "# @type=url\nU=://whatswrongwithyou\n", false},
		{"GoodURL", "# @type=url\nU=https://whoqmi.me/\n", true},
		{"BadEmail", "# @type=bool\nB=yesplease\n", false},
		{"GoodEmail", "# @type=email\nE=a@b.co\n", true},
		{"BadEmail", "# @type=email\nE=me?\n", false},
		{"GoodFloat", "# @type=float64\nF=0.5\n", true},
		{"BadNanFloat", "# @type=float64\nF=NaN\n", false},
		{"BadInfFloat", "# @type=float64\nF=Inf\n", false},
		{"GoodOctalInt", "# @type=int\nN=08\n", true},
	} {
		t.Run(c.Name, func(t *testing.T) {
			_, err := ParseString("", c.Source)

			if c.Ok && err != nil {
				t.Fatalf("expected success, got %v", err)
			}

			if !c.Ok && err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}

func TestParseRejectsVariableReferences(t *testing.T) {
	for _, c := range []struct{ name, src string }{
		{"Int", "# @type=int\nN=${BASE}\n"},
		{"Bool", "# @type=bool\nB=${FLAG}\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseString("", c.src)

			if err == nil || !strings.Contains(err.Error(), "${...}") {
				t.Fatalf("want ${} rejection error, got %v", err)
			}
		})
	}
}

func TestParseDecoratorWithEqualsInValue(t *testing.T) {
	s, err := ParseString("", "W=jwt=algo\n")

	if err != nil {
		t.Fatal(err)
	}

	if s.FieldByKey("W").Default != "jwt=algo" {
		t.Errorf("value with '=' not preserved")
	}
}

func TestParseConstraintOptions(t *testing.T) {
	s, err := ParseString("", "# @type=string(startsWith=sk-, regex=^[a-z0-9]+$)\nK=\n")

	if err != nil {
		t.Fatal(err)
	}

	f := s.FieldByKey("K")
	if f == nil {
		t.Fatal("K missing")
	}

	if f.StartsWith != "sk-" {
		t.Errorf("StartsWith = %q, want sk-", f.StartsWith)
	}

	if f.Regex != "^[a-z0-9]+$" {
		t.Errorf("Regex = %q, want ^[a-z0-9]+$", f.Regex)
	}

	if len(s.Warnings) != 0 {
		t.Errorf("want no warnings, got %v", s.Warnings)
	}
}

func TestParseStandaloneConstraintDecorators(t *testing.T) {
	s, err := ParseString("", "# @startsWith=sk-\n# @regex=^[a-z]+\nK=\n")

	if err != nil {
		t.Fatal(err)
	}

	f := s.FieldByKey("K")

	if f.StartsWith != "sk-" || f.Regex != "^[a-z]+" {
		t.Errorf("constraints lost: %+v", f)
	}
}

func TestParseTypeWithParenthesizedDocsOnSameLine(t *testing.T) {
	s, err := ParseString("", "# @type=url @docs=(see docs)\nK=https://whoqmi.me/\n")

	if err != nil {
		t.Fatal(err)
	}

	f := s.FieldByKey("K")

	if f.Kind != spec.KindURL {
		t.Errorf("kind = %v, want url", f.Kind)
	}

	if len(f.Docs) != 1 || f.Docs[0] != "(see docs)" {
		t.Errorf("docs = %v, want [(see docs)]", f.Docs)
	}
}

func TestParseInvalidRegexError(t *testing.T) {
	_, err := ParseString("", "# @regex=(^[a-\nK=\n")

	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestParseConstraintDefaultValidation(t *testing.T) {
	_, err := ParseString("", "# @startsWith=sk-\nK=i-am-not-sk-\n")
	if err == nil {
		t.Fatal("expected error: default does not start with prefix")
	}

	_, err = ParseString("", "# @regex=^[a-z]+$\nK=123\n")
	if err == nil {
		t.Fatal("expected error: default does not match regex")
	}

	_, err = ParseString("", "# @startsWith=sk-\nK=sk-good\n")
	if err != nil {
		t.Fatalf("valid default rejected: %v", err)
	}
}

func TestParseConstraintOnNonStringType(t *testing.T) {
	s, err := ParseString("c", "# @type=port(startsWith=1)\nP=\n")

	if err != nil {
		t.Fatal(err)
	}

	if s.FieldByKey("P").StartsWith != "" {
		t.Error("constraint must be ignored for non-string kinds")
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "string-like") {
		t.Errorf("want warning about unsupported kind, got %v", s.Warnings)
	}
}

func TestParseFilesMerge(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		t.Helper()

		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		return path
	}
	base := write("base.env.schema", "# @package=env\nA=1\nB=old\n")
	overlay := write(".env.local", "B=new\nC=3\n")

	s, err := ParseFiles(base, overlay)
	if err != nil {
		t.Fatal(err)
	}

	if s.Package != "env" {
		t.Errorf("package from base lost: %q", s.Package)
	}

	if got := s.FieldByKey("B").Default; got != "new" {
		t.Errorf("overlay did not replace B: %q", got)
	}

	if got := s.FieldByKey("C").Default; got != "3" {
		t.Errorf("overlay did not add C: %q", got)
	}

	if got := s.FieldByKey("A").Default; got != "1" {
		t.Errorf("base field lost: %q", got)
	}

	if len(s.Fields) != 3 {
		t.Errorf("len(fields) = %d, want 3", len(s.Fields))
	}
}

func TestParseFilesMissingOverlaySkipped(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.env.schema")
	if err := os.WriteFile(base, []byte("A=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := ParseFiles(base, filepath.Join(dir, ".env.prod"))
	if err != nil {
		t.Fatalf("missing overlay must be skipped, got %v", err)
	}

	if s.FieldByKey("A") == nil {
		t.Fatal("base fields missing")
	}
}

func TestParseFilesBaseError(t *testing.T) {
	if _, err := ParseFiles("/nonexistent/base.env.schema"); err == nil {
		t.Fatal("expected error for missing base file")
	}
}

func TestParseSensitiveDefaultNotEmbedded(t *testing.T) {
	s, err := ParseString("", "# @sensitive @required\nK=sdjkfjksdfklksdff\n")
	if err != nil {
		t.Fatal(err)
	}

	f := s.FieldByKey("K")
	if f == nil {
		t.Fatal("K missing")
	}

	if f.HasDefault || f.Default != "" {
		t.Errorf("sensitive default must be dropped, got HasDefault=%v Default=%q", f.HasDefault, f.Default)
	}

	if !f.Required || !f.Sensitive {
		t.Errorf("required/sensitive flags lost: %+v", f)
	}

	found := false
	for _, w := range s.Warnings {
		if strings.Contains(w, "sensitive and declares") {
			found = true
		}
	}

	if !found {
		t.Errorf("want warning about sensitive default, got %v", s.Warnings)
	}
}

func TestSeparatorResetsDocumentation(t *testing.T) {
	src := "# hi!!!\n# ---\n# attached\nX=1\n"
	s, err := ParseString("", src)

	if err != nil {
		t.Fatal(err)
	}

	f := s.FieldByKey("X")
	if len(f.Docs) != 1 || f.Docs[0] != "attached" {
		t.Errorf("separator should reset docs, got %v", f.Docs)
	}
}
