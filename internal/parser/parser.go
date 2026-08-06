package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var varKeyRx = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)

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
	}

	return s, nil
}

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

func ParseFile(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ParseString(path, string(data))
}

func ParseString(source, src string) (*Schema, error) {
	s := &Schema{SourcePath: source}

	currentLine := 0
	appendWarning := func(msg string, args ...any) {
		location := ""

		if source != "" {
			location = " " + source
		}

		s.Warnings = append(s.Warnings, fmt.Sprintf("s%s:%d: %s", location, currentLine, fmt.Sprintf(msg, args...)))
	}

	appendError := func(lineNo int, msg string, args ...any) error {
		location := ""
		if source != "" {
			location = " " + source
		}

		return fmt.Errorf("s%s:%d: %s", location, lineNo, fmt.Sprintf(msg, args...))
	}

	scanner := bufio.NewScanner(strings.NewReader(src))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var pending []string
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		currentLine = lineNo

		raw := strings.TrimRight(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(raw)

		switch {
		case trimmed == "":
			pending = nil

		case isComment(trimmed):
			body := strings.TrimLeft(trimmed, "#")
			body = strings.TrimSpace(body)

			if isSeparator(body) {
				pending = nil
				continue
			}

			if isRootDecorator(body) {
				parseRootDecorators(s, body, appendWarning)
				continue
			}

			pending = append(pending, body)

		default:
			m := varKeyRx.FindStringSubmatch(trimmed)
			if m == nil {
				return nil, appendError(lineNo, "expected KEY=VALUE, got %q", trimmed)
			}

			k, v := m[1], m[2]
			f := &Field{Key: k, Default: v, Line: lineNo}

			if v != "" {
				f.HasDefault = true
			}

			applyCommentBlock(s, pending, f, appendWarning)
			pending = nil

			if prev := s.FieldByKey(k); prev != nil {
				appendWarning("var %q is already present on :%d", k, prev.Line)
				continue
			}

			if f.Sensitive && f.HasDefault {
				appendWarning("variable %q sensitive and declares a default value", k)
				f.HasDefault = false
				f.Default = ""
			}

			if err := validateDefault(f); err != nil {
				return nil, err
			}

			if err := validateConstraints(f); err != nil {
				return nil, err
			}

			s.Fields = append(s.Fields, f)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return s, nil
}
