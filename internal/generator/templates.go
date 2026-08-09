package generator

const enumHelperSource = `func envContains(values []string, v string) bool {
	for _, item := range values {
		if item == v {
			return true
		}
	}

	return false
}
`

const expandHelperSource = `var envVarRefRx = regexp.MustCompile("\\$\\{(?P<name>[A-Za-z_][A-Za-z0-9_]*)\\}")

func expandVars(s string, m map[string]string) string {
	return envVarRefRx.ReplaceAllStringFunc(s, func(ref string) string {
		idx := envVarRefRx.SubexpIndex("name")
		sub := envVarRefRx.FindStringSubmatch(ref)[idx]

		if v, ok := m[sub]; ok && v != "" {
			return v
		}

		return ref
	})
}
`
