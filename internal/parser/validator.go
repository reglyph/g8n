package parser

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/reglyph/g8n/internal/spec"
)

func validateDefault(f *Field) error {
	if !f.HasDefault {
		return nil
	}

	ctx := fmt.Sprintf("line %d: variable %s", f.Line, f.Key)

	if strings.Contains(f.Default, "${") {
		if f.Kind.IsStringLike() {
			return nil
		}

		return fmt.Errorf("%s: default %q contains a ${...} reference, which is only supported for string-like types", ctx, f.Default)
	}

	if f.Kind == spec.KindEnum {
		if f.Default != "" && !slices.Contains(f.Enum, f.Default) {
			return fmt.Errorf("%s: default %q does not contain an enum reference", ctx, f.Default)
		}

		return nil
	}

	if err := f.Kind.ValidateLiteral(f.Default); err != nil {
		return fmt.Errorf("%s: %w", ctx, err)
	}

	return nil
}

func validateConstraints(f *Field) error {
	var rx *regexp.Regexp

	if f.HasRegex() {
		var err error

		rx, err = regexp.Compile(f.Regex)
		if err != nil {
			return fmt.Errorf("line %d: variable %s: invalid @regex=%q: %w", f.Line, f.Key, f.Regex, err)
		}
	}

	if !f.HasDefault {
		return nil
	}

	if f.StartsWith != "" && !strings.HasPrefix(f.Default, f.StartsWith) {
		return fmt.Errorf("line %d: variable %s: default %q does not start with %q", f.Line, f.Key, f.Default, f.StartsWith)
	}

	if f.HasRegex() && !rx.MatchString(f.Default) {
		return fmt.Errorf("line %d: variable %s: default %q does not match @regex=%q", f.Line, f.Key, f.Default, f.Regex)
	}

	return nil
}
