package spec

import "testing"

func TestLangString(t *testing.T) {
	cases := []struct {
		lang Lang
		want string
	}{
		{LangGo, "go"},
		{LangTS, "ts"},
		{Lang(-1), "go"},
	}

	for _, c := range cases {
		if got := c.lang.String(); got != c.want {
			t.Errorf("Lang(%d).String() = %q, want %q", c.lang, got, c.want)
		}
	}
}

func TestParseLang(t *testing.T) {
	for _, c := range []struct {
		name string
		lang Lang
		ok   bool
	}{
		{"go", LangGo, true},
		{"ts", LangTS, true},
		{"rust", 0, false},
		{"", 0, false},
	} {
		got, ok := ParseLang(c.name)
		if ok != c.ok || (c.ok && got != c.lang) {
			t.Errorf("ParseLang(%q) = %v, %v; want %v, %v", c.name, got, ok, c.lang, c.ok)
		}
	}
}

func TestLangSpecMapping(t *testing.T) {
	cases := []struct {
		kind Kind
		want string
	}{
		{KindString, "string"},
		{KindURL, "string"},
		{KindEmail, "string"},
		{KindEnum, "string"},
		{KindInt, "number"},
		{KindInt64, "number"},
		{KindFloat64, "number"},
		{KindPort, "number"},
		{KindBool, "boolean"},
	}

	for _, c := range cases {
		if got := c.kind.Spec().TSType; got != c.want {
			t.Errorf("Kind(%v).Spec().TSType = %q, want %q", c.kind, got, c.want)
		}
	}
}
