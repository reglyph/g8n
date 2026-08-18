package generator_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/reglyph/g8n/internal/generator/fixtures"
	"github.com/reglyph/g8n/internal/generator/golang"
	"github.com/reglyph/g8n/internal/generator/typescript"
	"github.com/reglyph/g8n/internal/parser"
)

type scene struct {
	name string
	env  map[string]string
	want string
}

// TestParityGoTS runs the same scenarios through the generated Go and TypeScript.
func TestParityGoTS(t *testing.T) {
	cases := []struct {
		name   string
		schema string
		scenes []scene
	}{
		{"basic", fixtures.BasicSchema, []scene{
			{"missing required", map[string]string{}, `env: required variable "API_KEY" is missing or empty`},
			{"all defaults", map[string]string{"API_KEY": "x"}, "ok"},
			{"bad int", map[string]string{"API_KEY": "x", "DB_PORT": "abc"}, `env: DB_PORT: value "abc" is not an integer: strconv.Atoi: parsing "abc": invalid syntax`},
			{"port zero", map[string]string{"API_KEY": "x", "DB_PORT": "0"}, "env: DB_PORT: port 0 is out of range 1..65535"},
			{"port too big", map[string]string{"API_KEY": "x", "DB_PORT": "70000"}, "env: DB_PORT: port 70000 is out of range 1..65535"},
			{"bad enum", map[string]string{"API_KEY": "x", "APP_ENV": "invalid"}, `env: APP_ENV: value "invalid" is not in the allowed list [dev, staging, prod]`},
			{"bad url", map[string]string{"API_KEY": "x", "API_URL": "not-a-url"}, `env: API_URL: value "not-a-url" is not a valid URL`},
			{"bad int64", map[string]string{"API_KEY": "x", "MAX_UPLOAD_BYTES": "abc"}, `env: MAX_UPLOAD_BYTES: value "abc" is not an int64: strconv.ParseInt: parsing "abc": invalid syntax`},
			{"bad float", map[string]string{"API_KEY": "x", "JITTER": "abc"}, `env: JITTER: value "abc" is not a float64: strconv.ParseFloat: parsing "abc": invalid syntax`},
			{"inf float", map[string]string{"API_KEY": "x", "JITTER": "Inf"}, "ok"},
			{"bad bool", map[string]string{"API_KEY": "x", "DEBUG": "maybe"}, `env: DEBUG: value "maybe" is not a boolean: strconv.ParseBool: parsing "maybe": invalid syntax`},
			{"one bool", map[string]string{"API_KEY": "x", "DEBUG": "1"}, "ok"},
			{"bad email", map[string]string{"API_KEY": "x", "ADMIN_EMAIL": "no-at"}, `env: ADMIN_EMAIL: value "no-at" is not a valid email`},
			{"email without dot", map[string]string{"API_KEY": "x", "ADMIN_EMAIL": "a@b"}, `env: ADMIN_EMAIL: value "a@b" is not a valid email`},
			{"enum and port ok", map[string]string{"API_KEY": "x", "APP_ENV": "prod", "DB_PORT": "5432"}, "ok"},
			{"first field error wins", map[string]string{"API_KEY": "x", "DB_PORT": "abc", "ADMIN_EMAIL": "bad"}, `env: DB_PORT: value "abc" is not an integer: strconv.Atoi: parsing "abc": invalid syntax`},
		}},
		{"enum quotes", "# @package=env\n# @type=enum(it's,a`b,a${x}b,50% off)\nAPP_ENV=\n", []scene{
			{"bad enum", map[string]string{"APP_ENV": "invalid"}, "env: APP_ENV: value \"invalid\" is not in the allowed list [it's, a`b, a${x}b, 50% off]"},
			{"good enum", map[string]string{"APP_ENV": "a`b"}, "ok"},
		}},
		{"constraints", fixtures.ConstraintsSchema, []scene{
			{"all defaults", map[string]string{}, "ok"},
			{"bad startsWith", map[string]string{"SERVICE": "bad"}, `env: SERVICE: value "bad" does not start with "sk_"`},
			{"good startsWith", map[string]string{"SERVICE": "sk_good"}, "ok"},
			{"bad regex", map[string]string{"TOKEN": "zz"}, `env: TOKEN: value "zz" does not match the required pattern`},
			{"good regex", map[string]string{"TOKEN": "0123456789abcdef0123456789abcdef"}, "ok"},
			{"port zero", map[string]string{"WEB_PORT": "0"}, "env: WEB_PORT: port 0 is out of range 1..65535"},
			{"port too big", map[string]string{"WEB_PORT": "99999"}, "env: WEB_PORT: port 99999 is out of range 1..65535"},
			{"bad float", map[string]string{"RATIO": "x"}, `env: RATIO: value "x" is not a float64: strconv.ParseFloat: parsing "x": invalid syntax`},
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := parser.ParseString(c.name+".env.schema", c.schema)
			if err != nil {
				t.Fatal(err)
			}

			for _, sc := range c.scenes {
				t.Run(sc.name, func(t *testing.T) {
					goSrc, err := golang.Generate(s)
					if err != nil {
						t.Fatal(err)
					}

					tsSrc, err := typescript.Generate(s)
					if err != nil {
						t.Fatal(err)
					}

					gotGo := runGo(t, goSrc, sc.env)
					gotTS := runTS(t, tsSrc, sc.env)

					if gotGo != sc.want {
						t.Errorf("go = %q, want %q", gotGo, sc.want)
					}

					if gotTS != sc.want {
						t.Errorf("ts = %q, want %q", gotTS, sc.want)
					}
				})
			}
		})
	}
}

// runGo compiles the generated Go source with a main() wrapper and runs the loader against env.
func runGo(t *testing.T, src []byte, env map[string]string) string {
	t.Helper()

	file := filepath.Join(t.TempDir(), "env.go")

	code := strings.Replace(string(src), "package env\n", "package main\n", 1)
	code += `
func main() {
	_, err := LoadFrom(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("ok")
}
`

	// #nosec G306 -- throwaway file in the test's own temp dir
	if err := os.WriteFile(file, []byte(code), 0o600); err != nil {
		t.Fatal(err)
	}

	args := make([]string, 0, 2+len(env))
	args = append(args, "run", file)

	for k, v := range env {
		args = append(args, k+"="+v)
	}

	// #nosec G204 -- args are the test's own env pairs
	out, err := exec.Command("go", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("go run: %v\n%s", err, out)
	}

	return strings.TrimSpace(string(out))
}

// runTS type-checks the generated TypeScript, compiles it to CommonJS and runs the loader in node against env.
func runTS(t *testing.T, src []byte, env map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	tsFile := filepath.Join(dir, "env.ts")

	// #nosec G306 -- throwaway file in the test's own temp dir
	if err := os.WriteFile(tsFile, src, 0o600); err != nil {
		t.Fatal(err)
	}

	// #nosec G204 -- fixed args, only the temp file path varies
	typescript.TSCompile(t, "--module", "commonjs", "--target", "es2020", tsFile)

	envJSON, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}

	script := `const m = require(process.argv[1]);
const env = JSON.parse(process.argv[2]);
try {
	m.LoadFrom(env);
	console.log("ok");
} catch (e) {
	console.log(e.message);
}`

	// #nosec G204 -- fixed args, only the temp file path and env JSON vary
	cmd := exec.Command("node", "-e", script, filepath.Join(dir, "env.js"), string(envJSON))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node: %v\n%s", err, out)
	}

	return strings.TrimSpace(string(out))
}
