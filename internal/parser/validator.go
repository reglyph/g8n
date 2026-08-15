package parser

import (
	"fmt"
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
	for _, c := range BuildConstraints(f) {
		if err := c.Validate(f); err != nil {
			return err
		}
	}

	if !f.HasDefault || strings.Contains(f.Default, "${") {
		return nil
	}

	for _, c := range BuildConstraints(f) {
		if err := c.ValidateDefault(f); err != nil {
			return err
		}
	}

	return nil
}
