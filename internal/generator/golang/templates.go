package golang

import (
	"strconv"

	"github.com/reglyph/g8n/internal/spec"
)

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

//nolint:gosec // template body contains schema param names, not real credentials
const onePasswordFetcherSource = `const secretFetchTimeout = 10 * time.Second

func fetch1Password(vault, item, field string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), secretFetchTimeout)
	defer cancel()

	ref := "op://" + vault + "/" + item + "/" + field

	out, err := exec.CommandContext(ctx, "op", "read", ref).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}

		return "", fmt.Errorf("1password: %s", msg)
	}

	return strings.TrimSpace(string(out)), nil
}
`

// providerOrder fixes the emission order of secret fetchers.
var providerOrder = []spec.SecretProvider{
	spec.Provider1Password,
	spec.ProviderAWS,
	spec.ProviderVault,
	spec.ProviderInfisical,
	spec.ProviderDoppler,
	spec.ProviderAzure,
	spec.ProviderGCP,
	spec.ProviderEnv,
}

// fetcherFuncName returns the generated fetcher function name for a provider.
func fetcherFuncName(pr spec.SecretProvider) string {
	switch pr {
	case spec.Provider1Password:
		return "fetch1Password"
	case spec.ProviderAWS:
		return "fetchAWS"
	case spec.ProviderVault:
		return "fetchVault"
	case spec.ProviderInfisical:
		return "fetchInfisical"
	case spec.ProviderDoppler:
		return "fetchDoppler"
	case spec.ProviderAzure:
		return "fetchAzure"
	case spec.ProviderGCP:
		return "fetchGCP"
	case spec.ProviderEnv:
		return "fetchEnv"
	default:
		return ""
	}
}

// fetcherArgs returns the positional argument expressions for a provider call.
func fetcherArgs(sp *spec.SourceSpec) []string {
	var names []string

	switch sp.Provider {
	case spec.Provider1Password:
		names = []string{"vault", "item", "field"}
	case spec.ProviderAWS:
		names = []string{"secret", "region"}
	case spec.ProviderVault:
		names = []string{"addr", "token", "mount", "path", "field"}
	case spec.ProviderInfisical:
		names = []string{"domain", "token", "project", "environment", "secret"}
	case spec.ProviderDoppler:
		names = []string{"token", "project", "config", "secret"}
	case spec.ProviderAzure:
		names = []string{"vault", "secret", "version"}
	case spec.ProviderGCP:
		names = []string{"project", "secret", "version"}
	case spec.ProviderEnv:
		names = []string{"name"}
	}

	args := make([]string, 0, len(names))

	for _, name := range names {
		args = append(args, strconv.Quote(sp.Params[name]))
	}

	return args
}
