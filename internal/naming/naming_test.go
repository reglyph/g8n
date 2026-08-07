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
