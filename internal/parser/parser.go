package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var varKeyRx = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)

const (
	scannerInitialBufferSize = 64 * 1024
	scannerMaxTokenSize      = 1024 * 1024
)

// ParseFiles parses the base schema and merges the given overlay files on top of it.
// Missing overlay files are silently skipped.
func ParseFiles(base string, overlays ...string) (*Schema, error) {
	s, err := ParseFile(base)
	if err != nil {
		return nil, err
	}

	for _, overlay := range overlays {
		if overlay == "" {
			continue
		}

		o, err := ParseFile(overlay)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, err
		}

		Merge(s, o)
		s.Warnings = append(s.Warnings, o.Warnings...)

		if o.Package != "" || o.OutPath != "" {
			s.Warnings = append(s.Warnings, fmt.Sprintf("envschema %s: @package/@out are ignored in overlay files; configure them in the base schema instead", overlay))
		}
	}

	return s, nil
}

// Merge applies the fields of src on top of dst, replacing matching keys.
func Merge(dst, src *Schema) {
	for _, f := range src.Fields {
		replaced := false

		for i, exist := range dst.Fields {
			if exist.Key == f.Key {
				dst.Fields[i] = f
				replaced = true

				break
			}
		}

		if !replaced {
			dst.Fields = append(dst.Fields, f)
		}
	}
}

// ParseFile parses the schema file at the given path.
func ParseFile(path string) (*Schema, error) {
	// #nosec G304 -- path is provided by the caller
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ParseString(path, string(data))
}

// ParseString parses the given schema source. source is only used for
// error and warning messages and may be empty.
func ParseString(source, src string) (*Schema, error) {
	s := &Schema{SourcePath: source}

	currentLine := 0
	appendWarning := func(msg string, args ...any) {
		location := ""

		if source != "" {
			location = " " + source
		}

		s.Warnings = append(s.Warnings, fmt.Sprintf("envschema%s:%d: %s", location, currentLine, fmt.Sprintf(msg, args...)))
	}

	appendError := func(lineNo int, msg string, args ...any) error {
		location := ""
		if source != "" {
			location = " " + source
		}

		return fmt.Errorf("envschema%s:%d: %s", location, lineNo, fmt.Sprintf(msg, args...))
	}

	scanner := bufio.NewScanner(strings.NewReader(src))
	scanner.Buffer(make([]byte, 0, scannerInitialBufferSize), scannerMaxTokenSize)

	var pending []string

	lineNo := 0

	for scanner.Scan() {
		lineNo++
		currentLine = lineNo

		raw := strings.TrimRight(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(raw)

		switch {
		case trimmed == "":
			pending = pending[:0]

		case isComment(trimmed):
			body := strings.TrimLeft(trimmed, "#")
			body = strings.TrimSpace(body)

			if isSeparator(body) {
				pending = pending[:0]
				continue
			}

			if isRootDecorator(body) {
				parseRootDecorators(s, body, appendWarning)
				continue
			}

			pending = append(pending, body)

		default:
			if err := parseFieldLine(s, pending, trimmed, lineNo, appendWarning, appendError); err != nil {
				return nil, err
			}

			pending = pending[:0]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return s, nil
}

func parseFieldLine(s *Schema, pending []string, line string, lineNo int, warn warnf, fail func(int, string, ...any) error) error {
	m := varKeyRx.FindStringSubmatch(line)
	if m == nil {
		return fail(lineNo, "expected KEY=VALUE")
	}

	k, v := m[1], m[2]
	f := &Field{Key: k, Default: v, Line: lineNo}

	if v != "" {
		f.HasDefault = true
	}

	applyCommentBlock(s, pending, f, warn)

	if prev := s.FieldByKey(k); prev != nil {
		warn("var %q is already present on :%d", k, prev.Line)
		return nil
	}

	if f.Sensitive && f.HasDefault {
		warn("variable %q sensitive and declares a default value", k)

		f.HasDefault = false
		f.Default = ""
	}

	if err := validateDefault(f); err != nil {
		return err
	}

	if err := validateConstraints(f); err != nil {
		return err
	}

	s.Fields = append(s.Fields, f)

	return nil
}
