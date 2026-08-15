package naming

import "testing"

func TestGoFieldName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"APP_ENV", "AppEnv"},
		{"PORT", "Port"},
		{"FEATURE_V_2", "FeatureV2"},
		{"V3_ID", "V3Id"},
		{"MAX_UPLOAD_BYTES", "MaxUploadBytes"},
	}

	for _, c := range cases {
		if got := GoFieldName(c.in); got != c.want {
			t.Errorf("GoFieldName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTSFieldName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"APP_ENV", "appEnv"},
		{"PORT", "port"},
		{"FEATURE_V_2", "featureV2"},
		{"V3_ID", "v3Id"},
		{"MAX_UPLOAD_BYTES", "maxUploadBytes"},
		{"", ""},
	}

	for _, c := range cases {
		if got := TSFieldName(c.in); got != c.want {
			t.Errorf("TSFieldName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLowerFirst(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Hello", "hello"},
		{"DBHost", "dBHost"},
		{"", ""},
		{"x", "x"},
	}

	for _, c := range cases {
		if got := LowerFirst(c.in); got != c.want {
			t.Errorf("LowerFirst(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
