package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/reglyph/g8n/internal/generator"
	"github.com/reglyph/g8n/internal/jsonschema"
	"github.com/reglyph/g8n/internal/parser"
)

var ver = "0.0.1"
var pkgNameRx = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

const (
	dirPerm  = 0o755
	filePerm = 0o644
)

func emit(w io.Writer, format string, args ...any) {
	//nolint:errcheck // writing to stdout/stderr is best-effort
	_, _ = fmt.Fprintf(w, format, args...)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("g8n", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		schemaPath = fs.String("schema", ".env.schema", "path to the .env.schema file")
		outPath    = fs.String("out", "", "path to the output .go file")
		pkgName    = fs.String("package", "", "output package name (default: directory of -out, or @package= decorator)")
		force      = fs.Bool("force", false, "write the file even if nothing changed")
		dryRun     = fs.Bool("dry-run", false, "print the generated source to stdout without writing")
		envName    = fs.String("env", "", "merge the .env.<env> overlay on top of .env.local")
		jsonOut    = fs.String("json", "", "write a JSON Schema (draft-07) to this path instead of Go code")
		showVer    = fs.Bool("version", false, "print version and exit")
	)

	fs.Usage = func() {
		emit(stderr, "@reglyph/g8n — type-safe struct generator for Go\n\n")
		emit(stderr, "Usage:\n  g8n -out <file.go> [flags]\n  g8n -schema .env.schema -out internal/env/env.go\n\nFlags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}

		return 2
	}

	if *showVer {
		emit(stdout, "@reglyph/g8n %s\n", ver)
		return 0
	}

	if fs.NArg() > 0 {
		emit(stderr, "g8n: unexpected positional arguments: %v\n", fs.Args())
		return 2
	}

	if err := generate(*schemaPath, *outPath, *pkgName, *envName, *jsonOut, *force, *dryRun, stdout, stderr); err != nil {
		emit(stderr, "g8n: %v\n", err)
		return 1
	}

	return 0
}

func overlayFiles(schemaPath, env string) []string {
	dir := filepath.Dir(schemaPath)

	var files []string

	files = append(files, filepath.Join(dir, ".env.local"))
	if env != "" {
		files = append(files, filepath.Join(dir, ".env."+env))
	}

	return files
}

func generate(schemaPath, outPath, pkgName, envName, jsonOut string, force, dryRun bool, stdout, stderr io.Writer) error {
	schema, err := parser.ParseFiles(schemaPath, overlayFiles(schemaPath, envName)...)
	if err != nil {
		return err
	}

	for _, w := range schema.Warnings {
		emit(stderr, "%s\n", w)
	}

	if jsonOut != "" {
		return generateJSONSchema(schema, jsonOut, dryRun, stdout, stderr)
	}

	pkgName, outPath, err = resolveOutputs(schemaPath, schema, pkgName, outPath)
	if err != nil {
		return err
	}

	schema.Package = pkgName

	src, err := generator.Generate(schema)
	if err != nil {
		return fmt.Errorf("generate %s: %w", outPath, err)
	}

	if dryRun {
		_, err = stdout.Write(src)

		return err
	}

	return writeIfChanged(outPath, src, force, stderr)
}

func resolveOutputs(schemaPath string, schema *parser.Schema, pkg, outPath string) (pkgName, out string, err error) {
	if pkg == "" {
		pkg = schema.Package
	}

	if outPath == "" {
		outPath = schema.OutPath

		if outPath != "" {
			outPath = resolveRelative(filepath.Dir(schemaPath), outPath)
		}
	}

	if outPath == "" {
		return "", "", fmt.Errorf("no out path: pass -out or declare @out(path=...) in %s", schemaPath)
	}

	if pkg == "" {
		pkg = basePackageName(outPath)
	}

	if !pkgNameRx.MatchString(pkg) {
		return "", "", fmt.Errorf("invalid Go package name %q", pkg)
	}

	return pkg, outPath, nil
}

func writeIfChanged(outPath string, src []byte, force bool, stderr io.Writer) error {
	// #nosec G304 -- outPath is the user-provided -out flag
	existing, rerr := os.ReadFile(outPath)
	identical := rerr == nil && string(existing) == string(src)

	if identical && !force {
		emit(stderr, "g8n: %s is up to date\n", outPath)

		return nil
	}

	if dir := filepath.Dir(outPath); dir != "." {
		// #nosec G301 -- directories are created in the user's repo with standard permissions
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	// #nosec G306 -- generated source must stay readable by all users of the repo
	if err := os.WriteFile(outPath, src, filePerm); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	if identical {
		emit(stderr, "g8n: refreshed %s\n", outPath)
	} else {
		emit(stderr, "g8n: wrote %s\n", outPath)
	}

	return nil
}

func generateJSONSchema(schema *parser.Schema, path string, dryRun bool, stdout, stderr io.Writer) error {
	src, err := jsonschema.Generate(schema)
	if err != nil {
		return fmt.Errorf("generate JSON schema: %w", err)
	}

	src = append(src, '\n')

	if dryRun {
		_, err = stdout.Write(src)

		return err
	}

	if dir := filepath.Dir(path); dir != "." {
		// #nosec G301 -- directories are created in the user's repo with standard permissions
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	// #nosec G306 -- generated source must stay readable by all users of the repo
	if err := os.WriteFile(path, src, filePerm); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	emit(stderr, "g8n: wrote %s\n", path)

	return nil
}

func basePackageName(out string) string {
	dir := filepath.Dir(out)
	base := filepath.Base(dir)

	if base == "" || base == "." || base == string(filepath.Separator) {
		return "env"
	}

	var keep []rune

	for _, r := range base {
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			keep = append(keep, r)
		}
	}

	name := string(keep)

	if name == "" {
		return "env"
	}

	if name[0] >= '0' && name[0] <= '9' {
		name = "env_" + name
	}

	return name
}

func resolveRelative(dir, opt string) string {
	if filepath.IsAbs(opt) {
		return opt
	}

	return filepath.Join(dir, opt)
}
