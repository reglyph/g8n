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

func TestParseDuplicateTypeWarns(t *testing.T) {
	s, err := ParseString("", "# @type=port\n# @type=int\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	f := s.FieldByKey("K")
	if f == nil {
		t.Fatal("field K missing")
	}

	if f.Kind != spec.KindInt {
		t.Errorf("kind = %v, want last @type to win (int)", f.Kind)
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "@type already declared") {
		t.Errorf("want duplicate @type warning, got %v", s.Warnings)
	}
}

func TestParseSourceSingleProvider(t *testing.T) {
	s, err := ParseString("", "# @source=1password(vault=prod, item=db-creds, field=password)\nDB_PASSWORD=\n")
	if err != nil {
		t.Fatal(err)
	}

	f := s.FieldByKey("DB_PASSWORD")
	if f == nil {
		t.Fatal("field DB_PASSWORD missing")
	}

	if f.Source != "1password(vault=prod, item=db-creds, field=password)" {
		t.Errorf("Source = %q, want raw value preserved", f.Source)
	}

	if len(f.SourceSpecs) != 1 {
		t.Fatalf("SourceSpecs = %d specs, want 1", len(f.SourceSpecs))
	}

	got := f.SourceSpecs[0]
	if got.Provider != spec.Provider1Password {
		t.Errorf("Provider = %q, want 1password", got.Provider)
	}

	if got.Next != nil {
		t.Error("single spec must not have a Next")
	}

	wantParams := map[string]string{"vault": "prod", "item": "db-creds", "field": "password"}
	if len(got.Params) != 3 {
		t.Errorf("Params = %v, want %v", got.Params, wantParams)
	}

	for k, v := range wantParams {
		if got.Params[k] != v {
			t.Errorf("Params[%q] = %q, want %q", k, got.Params[k], v)
		}
	}

	if len(s.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", s.Warnings)
	}
}

func TestParseSourceChainFallback(t *testing.T) {
	s, err := ParseString("", "# @source=1password(vault=prod, item=db) | aws(secret=prod/db/password, region=us-east-1) | env(DB_PASSWORD)\nDB_PASSWORD=\n")
	if err != nil {
		t.Fatal(err)
	}

	specs := s.FieldByKey("DB_PASSWORD").SourceSpecs
	if len(specs) != 3 {
		t.Fatalf("SourceSpecs = %d specs, want 3", len(specs))
	}

	wantProviders := []spec.SecretProvider{spec.Provider1Password, spec.ProviderAWS, spec.ProviderEnv}
	for i, want := range wantProviders {
		if specs[i].Provider != want {
			t.Errorf("specs[%d].Provider = %q, want %q", i, specs[i].Provider, want)
		}
	}

	if specs[0].Next != &specs[1] || specs[1].Next != &specs[2] || specs[2].Next != nil {
		t.Error("fallback chain not linked in order")
	}

	if specs[1].Params["region"] != "us-east-1" {
		t.Errorf("aws region = %q, want us-east-1", specs[1].Params["region"])
	}

	if len(s.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", s.Warnings)
	}
}

func TestParseSourceBareParamSkipped(t *testing.T) {
	s, err := ParseString("", "# @source=env(DB_PASSWORD)\nDB_PASSWORD=\n")
	if err != nil {
		t.Fatal(err)
	}

	specs := s.FieldByKey("DB_PASSWORD").SourceSpecs
	if len(specs) != 1 || specs[0].Provider != spec.ProviderEnv {
		t.Fatalf("SourceSpecs = %#v, want single env spec", specs)
	}

	if len(specs[0].Params) != 0 {
		t.Errorf("Params = %v, want none (bare token)", specs[0].Params)
	}

	if len(s.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", s.Warnings)
	}
}

func TestParseSourceUnknownProviderBestEffort(t *testing.T) {
	s, err := ParseString("", "# @source=custom-vault(addr=http://localhost:8200)\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	specs := s.FieldByKey("K").SourceSpecs
	if len(specs) != 1 || specs[0].Provider != spec.SecretProvider("custom-vault") {
		t.Fatalf("SourceSpecs = %#v, want single best-effort spec", specs)
	}

	if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], "unknown provider") {
		t.Errorf("want unknown provider warning, got %v", s.Warnings)
	}
}

func TestParseSourceWarnings(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "missing closing paren", src: "# @source=aws(secret=x\nK=\n", want: "missing closing ')'"},
		{name: "empty value", src: "# @source=\nK=\n", want: "@source requires a value"},
		{name: "duplicate", src: "# @source=env(ONE)\n# @source=env(TWO)\nK=\n", want: "@source already declared"},
		{name: "empty parameter", src: "# @source=aws(secret=, region=us-east-1)\nK=\n", want: "empty parameter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := ParseString("", tt.src)
			if err != nil {
				t.Fatal(err)
			}

			if len(s.Warnings) == 0 || !strings.Contains(s.Warnings[0], tt.want) {
				t.Errorf("ParseString(%q) warnings = %v, want containing %q", tt.src, s.Warnings, tt.want)
			}
		})
	}
}

func TestParseSourceDuplicateLastWins(t *testing.T) {
	s, err := ParseString("", "# @source=env(ONE)\n# @source=env(TWO)\nK=\n")
	if err != nil {
		t.Fatal(err)
	}

	if got := s.FieldByKey("K").Source; got != "env(TWO)" {
		t.Errorf("Source = %q, want last @source to win", got)
	}
}
