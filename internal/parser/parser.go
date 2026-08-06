package parser

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

var varKeyRx = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)

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

	lineNo := 0

	for scanner.Scan() {
		lineNo++
		currentLine = lineNo

		raw := strings.TrimRight(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(raw)

		switch {
		// todo: comment & trimmed is nil
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

			if prev := s.FieldByKey(k); prev != nil {
				// todo: duplicate
				appendWarning("var %q is already present on :%d", k, prev.Line)
				continue
			}

			// todo: sensitive

			s.Fields = append(s.Fields, f)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return s, nil
}
