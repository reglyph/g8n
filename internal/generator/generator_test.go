package generator

import (
	"strings"
	"testing"

	"github.com/reglyph/g8n/internal/parser"
	"github.com/reglyph/g8n/internal/spec"
)

func TestGenerateDispatch(t *testing.T) {
	s, err := parser.ParseString("test.env.schema", "# @package=env\nA=1\n")
	if err != nil {
		t.Fatal(err)
	}

	for _, lang := range []spec.Lang{spec.LangGo, spec.LangTS} {
		out, err := Generate(s, lang)
		if err != nil {
			t.Fatalf("Generate(%s): %v", lang, err)
		}

		if !strings.Contains(string(out), "DO NOT EDIT") {
			t.Errorf("Generate(%s) output lacks header", lang)
		}
	}
}

func TestGenerateUnsupportedLang(t *testing.T) {
	s, err := parser.ParseString("test.env.schema", "# @package=env\nA=1\n")
	if err != nil {
		t.Fatal(err)
	}

	_, err = Generate(s, spec.Lang(99))
	if err == nil || !strings.Contains(err.Error(), "unsupported language") {
		t.Fatalf("want unsupported language error, got %v", err)
	}
}
