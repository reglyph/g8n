package constraints

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/reglyph/g8n/internal/model"
	"github.com/reglyph/g8n/internal/printer"
	"github.com/reglyph/g8n/internal/spec"
)

// StartsWith returns the @startsWith constraint.
func StartsWith() Constraint { return startsWithConstraint{} }

type startsWithConstraint struct{}

func (startsWithConstraint) Name() string                { return "startsWith" }
func (startsWithConstraint) Present(f *model.Field) bool { return f.StartsWith != "" }
func (startsWithConstraint) Validate(*model.Field) error { return nil }

func (startsWithConstraint) ValidateDefault(f *model.Field) error {
	if !strings.HasPrefix(f.Default, f.StartsWith) {
		return fmt.Errorf("%s: default %q does not start with %q", fieldCtx(f), f.Default, f.StartsWith)
	}

	return nil
}

func (startsWithConstraint) Emit(p *printer.Printer, f *model.Field, src, _ string, lang spec.Lang) {
	switch lang {
	case spec.LangTS:
		p.Linef("if (!%s.startsWith(%s)) {", src, strconv.Quote(f.StartsWith))
		p.Indent()
		writeTSConstraintError(p, f,
			"value does not start with the required prefix",
			fmt.Sprintf(`value "${%s}" does not start with %s`, src, JSEscape(strconv.Quote(f.StartsWith))))
		p.Dedent()
		p.Line("}")
	default:
		p.Linef("if !strings.HasPrefix(%s, %q) {", src, f.StartsWith)
		p.Indent()
		writeConstraintError(p, f, "value does not start with the required prefix", "value %q does not start with %q", src, strconv.Quote(f.StartsWith))
		p.Dedent()
		p.Line("}")
	}
}

func (startsWithConstraint) Schema(f *model.Field) (FieldSchema, error) {
	return FieldSchema{Pattern: "^" + regexp.QuoteMeta(f.StartsWith)}, nil
}
