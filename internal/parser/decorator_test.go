package parser

import (
	"strings"
	"testing"

	"github.com/reglyph/g8n/internal/spec"
)

func TestIsDecoratorEmptyLine(t *testing.T) {
	if isDecorator("") || isDecorator("   ") {
		t.Error("isDecorator must return false for lines without tokens")
	}
}

func TestSplitParamsParenAware(t *testing.T) {
	if got := splitParams("a,(b,c),d"); len(got) != 3 || got[0] != "a" || got[1] != "(b,c)" || got[2] != "d" {
		t.Errorf("splitParams(a,(b,c),d) = %#v", got)
	}

	if got := splitParams(""); got != nil {
		t.Errorf("splitParams() = %#v, want nil", got)
	}
}

func TestRegexWithCommaInGroup(t *testing.T) {
	s, err := ParseString("", "# @type=string(regex=^(a,b)+$)\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	f := s.FieldByKey("K")
	if f == nil || f.Regex != "^(a,b)+$" {
		t.Fatalf("regex = %q, want ^(a,b)+$", f.Regex)
	}

	if len(s.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", s.Warnings)
	}
}

func TestEnumValueWithCommaInParens(t *testing.T) {
	s, err := ParseString("", "# @type=enum(one,(two,three))\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	f := s.FieldByKey("K")
	if f == nil {
		t.Fatal("field K missing")
	}

	if len(f.Enum) != 2 || f.Enum[0] != "one" || f.Enum[1] != "(two,three)" {
		t.Errorf("enum = %#v, want [one (two,three)]", f.Enum)
	}
}

func TestDecoratorNameParenWithoutEquals(t *testing.T) {
	if got := decoratorName("@out(path=x)"); got != "@out" {
		t.Errorf("decoratorName(@out(path=x)) = %q, want @out", got)
	}
}

func TestApplyCommentBlockRootDecorator(t *testing.T) {
	s := &Schema{}
	f := &Field{Key: "K"}

	applyCommentBlock(s, []string{"@package=pkg"}, f, func(string, ...any) {})

	if s.Package != "pkg" {
		t.Errorf("Package = %q, want pkg", s.Package)
	}

	if len(f.Docs) != 0 {
		t.Errorf("root decorator must not become docs, got %v", f.Docs)
	}
}

func TestParseBarePackageWarning(t *testing.T) {
	s, err := ParseString("", "# @package\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "@package required a value") {
		t.Errorf("want @package warning, got %v", s.Warnings)
	}
}

func TestParseBareOutWarning(t *testing.T) {
	s, err := ParseString("", "# @out\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "requires a path parameter") {
		t.Errorf("want @out warning, got %v", s.Warnings)
	}
}

func TestParseOutParamWithoutEquals(t *testing.T) {
	s, err := ParseString("", "# @out(path)\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	if s.OutPath != "" {
		t.Errorf("OutPath = %q, want empty", s.OutPath)
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "requires a path parameter") {
		t.Errorf("want @out warning, got %v", s.Warnings)
	}
}

func TestParseOutPackageParam(t *testing.T) {
	s, err := ParseString("", "# @out(package=foo)\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	if s.Package != "foo" {
		t.Errorf("Package = %q, want foo", s.Package)
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "requires a path parameter") {
		t.Errorf("want @out path warning, got %v", s.Warnings)
	}
}

func TestParseNonDecoratorTokenInLine(t *testing.T) {
	s, err := ParseString("", "# @required foo\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	if !s.FieldByKey("K").Required {
		t.Error("@required lost in multi-token line")
	}

	if len(s.Warnings) != 0 {
		t.Errorf("want no warnings, got %v", s.Warnings)
	}
}

func TestParseDefaultInMultiTokenLine(t *testing.T) {
	s, err := ParseString("", "# @required @default=1\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	f := s.FieldByKey("K")
	if !f.Required || !f.HasDefault || f.Default != "1" {
		t.Errorf("required/default lost: %+v", f)
	}
}

func TestParseUnknownDecoratorInLoop(t *testing.T) {
	s, err := ParseString("", "# @unknown\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "unknown decorator @unknown") {
		t.Errorf("want unknown decorator warning, got %v", s.Warnings)
	}
}

func TestParseStandaloneDefault(t *testing.T) {
	s, err := ParseString("", "# @default=5\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	f := s.FieldByKey("K")
	if !f.HasDefault || f.Default != "5" {
		t.Errorf("default lost: %+v", f)
	}
}

func TestParseTypeUnclosedParen(t *testing.T) {
	s, err := ParseString("", "# @type=url(abc\nK=http://x.dev\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "expected closing ')'") {
		t.Errorf("want closing paren warning, got %v", s.Warnings)
	}

	if s.FieldByKey("K").Kind != spec.KindString {
		t.Errorf("kind must fall back to string")
	}
}

func TestParseUnknownTypeFallsBack(t *testing.T) {
	s, err := ParseString("", "# @type=bigint\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	if s.FieldByKey("K").Kind != spec.KindString {
		t.Errorf("kind must fall back to string")
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "unknown @type") {
		t.Errorf("want unknown @type warning, got %v", s.Warnings)
	}
}

func TestParseEnumNoValues(t *testing.T) {
	s, err := ParseString("", "# @type=enum()\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "comma-separated") {
		t.Errorf("want enum warning, got %v", s.Warnings)
	}
}

func TestParseTypeOptionWithoutEquals(t *testing.T) {
	s, err := ParseString("", "# @type=string(foo)\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "ignoring unsupported @type option") {
		t.Errorf("want unsupported option warning, got %v", s.Warnings)
	}
}

func TestParseStartsWithEmptyValue(t *testing.T) {
	s, err := ParseString("", "# @type=string(startsWith=)\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "startsWith requires a non-empty value") {
		t.Errorf("want startsWith warning, got %v", s.Warnings)
	}
}

func TestParseRegexEmptyValue(t *testing.T) {
	s, err := ParseString("", "# @type=string(regex=)\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "regex requires a non-empty value") {
		t.Errorf("want regex warning, got %v", s.Warnings)
	}
}

func TestParseRegexOnNonStringKind(t *testing.T) {
	s, err := ParseString("", "# @type=port(regex=^1)\nP=\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "string-like") {
		t.Errorf("want string-like warning, got %v", s.Warnings)
	}
}

func TestParseUnknownTypeOption(t *testing.T) {
	s, err := ParseString("", "# @type=string(foo=bar)\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "unknown @type option") {
		t.Errorf("want unknown option warning, got %v", s.Warnings)
	}
}

func TestParseStartsWithOnNonStringKindStandalone(t *testing.T) {
	s, err := ParseString("", "# @type=int\n# @startsWith=sk-\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "string-like") {
		t.Errorf("want string-like warning, got %v", s.Warnings)
	}
}

func TestParseStandaloneStartsWithEmpty(t *testing.T) {
	s, err := ParseString("", "# @startsWith=\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "startsWith requires a non-empty value") {
		t.Errorf("want startsWith warning, got %v", s.Warnings)
	}
}
