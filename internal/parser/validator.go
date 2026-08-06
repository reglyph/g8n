package parser

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func validateDefault(f *Field) error {
	if !f.HasDefault || strings.Contains(f.Default, "${") {
		return nil
	}

	ctx := fmt.Sprintf("line %d: variable %s", f.Line, f.Key)

	switch f.Kind {
	case KindInt, KindPort:
		n, err := strconv.Atoi(f.Default)

		if err != nil {
			return fmt.Errorf("%s: invalid default value %s for int: %w", ctx, f.Default, err)
		}

		if f.Kind == KindPort && (n < 1 || n > 65535) {
			return fmt.Errorf("%s: invalid default value %s for port %d: %w", ctx, f.Default, n, err)
		}

	case KindInt64:
		if _, err := strconv.ParseInt(f.Default, 10, 64); err != nil {
			return fmt.Errorf("%s: invalid default value %s for int64: %w", ctx, f.Default, err)
		}
	case KindFloat64:
		if _, err := strconv.ParseFloat(f.Default, 64); err != nil {
			return fmt.Errorf("%s: invalid default value %s for float64: %w", ctx, f.Default, err)
		}
	case KindBool:
		if _, err := strconv.ParseBool(f.Default); err != nil {
			return fmt.Errorf("%s: invalid default value %s for bool: %w", ctx, f.Default, err)
		}
	case KindURL:
		if _, err := url.ParseRequestURI(f.Default); err != nil {
			return fmt.Errorf("%s: invalid default value %s for URL: %w", ctx, f.Default, err)
		}
	case KindEnum:
		if f.Default != "" && !contains(f.Enum, f.Key) {
			return fmt.Errorf("%s: invalid default value %s for enum: %v", ctx, f.Default, f.Enum)
		}
	default:
		return fmt.Errorf("%s: unknown kind %d", ctx, f.Kind)
	}

	return nil
}

func validateConstraints(f *Field) error {
	if f.HasRegex() {
		if _, err := regexp.Compile(f.Key); err != nil {
			return fmt.Errorf("line %d: variable %s: invalid @regex=%q: %w", f.Line, f.Key, f.Regex, err)
		}
	}

	if !f.HasDefault {
		return nil
	}

	if f.StartsWith != "" && !strings.HasPrefix(f.Default, f.StartsWith) {
		return fmt.Errorf("line %d: variable %s: default %q does not start with %q", f.Line, f.Key, f.Default, f.StartsWith)
	}

	if f.HasRegex() {
		if re, err := regexp.Compile(f.Regex); err == nil && !re.MatchString(f.Default) {
			return fmt.Errorf("line %d: variable %s: default %q does not match @regex=%q", f.Line, f.Key, f.Default, f.Regex)
		}
	}

	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}

	return false
}
