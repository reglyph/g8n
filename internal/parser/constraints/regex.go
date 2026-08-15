package constraints

import (
	"fmt"
	"regexp"

	"github.com/reglyph/g8n/internal/model"
	"github.com/reglyph/g8n/internal/printer"
	"github.com/reglyph/g8n/internal/spec"
)

// Regex returns the @regex constraint.
func Regex() Constraint { return regexConstraint{} }

type regexConstraint struct{}

func (regexConstraint) Name() string                { return "regex" }
func (regexConstraint) Present(f *model.Field) bool { return f.HasRegex() }

func (regexConstraint) Validate(f *model.Field) error {
	if _, err := regexp.Compile(f.Regex); err != nil {
		return fmt.Errorf("%s: invalid @regex=%q: %w", fieldCtx(f), f.Regex, err)
	}

	return nil
}

func (regexConstraint) ValidateDefault(f *model.Field) error {
	rx, err := regexp.Compile(f.Regex)
	if err != nil {
		return fmt.Errorf("%s: invalid @regex=%q: %w", fieldCtx(f), f.Regex, err)
	}

	if !rx.MatchString(f.Default) {
		return fmt.Errorf("%s: default %q does not match @regex=%q", fieldCtx(f), f.Default, f.Regex)
	}

	return nil
}

func (regexConstraint) Emit(p *printer.Printer, f *model.Field, src, rxVar string, lang spec.Lang) {
	switch lang {
	case spec.LangTS:
		p.Linef("if (!%s.test(%s)) {", rxVar, src)
		p.Indent()
		writeTSConstraintError(p, f,
			"value does not match the required pattern",
			fmt.Sprintf(`value "${%s}" does not match the required pattern`, src))
		p.Dedent()
		p.Line("}")
	default:
		p.Linef("if !%s.MatchString(%s) {", rxVar, src)
		p.Indent()
		writeConstraintError(p, f, "value does not match the required pattern", "value %q does not match the required pattern", src, "")
		p.Dedent()
		p.Line("}")
	}
}

func (regexConstraint) Schema(f *model.Field) (FieldSchema, error) {
	return FieldSchema{Pattern: f.Regex}, nil
}
