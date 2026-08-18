package constraints

import (
	"strings"

	"github.com/reglyph/g8n/internal/model"
	"github.com/reglyph/g8n/internal/printer"
	"github.com/reglyph/g8n/internal/spec"
)

// WriteConvCheck emits the conversion error check after strconv parsing.
func WriteConvCheck(p *printer.Printer, f *model.Field, what, src string) {
	p.Line("if convErr != nil {")
	p.Indent()
	writeParseError(p, f, what, src)
	p.Dedent()
	p.Line("}")
}

func writeParseError(p *printer.Printer, f *model.Field, what, src string) {
	if f.Sensitive {
		p.Linef("return e, fmt.Errorf(\"env: %s: value is not %s\")", f.Key, what)
		return
	}

	p.Linef("return e, fmt.Errorf(\"env: %s: value %%q is not %s: %%w\", %s, convErr)", f.Key, what, src)
}

// WritePortError emits the port range violation error line.
func WritePortError(p *printer.Printer, f *model.Field) {
	if f.Sensitive {
		p.Linef("return e, fmt.Errorf(\"env: %s: port is out of range %d..%d\")", f.Key, spec.PortMin, spec.PortMax)
		return
	}

	p.Linef("return e, fmt.Errorf(\"env: %s: port %%d is out of range %d..%d\", n)", f.Key, spec.PortMin, spec.PortMax)
}

// WriteValueError emits the value validation error line; extra renders as an additional format argument.
func WriteValueError(p *printer.Printer, f *model.Field, sensitiveMsg, msg, src, extra string) {
	if f.Sensitive {
		p.Linef("return e, fmt.Errorf(\"env: %s: %s\")", f.Key, sensitiveMsg)
		return
	}

	if extra != "" {
		p.Linef("return e, fmt.Errorf(\"env: %s: %s\", %s, %s)", f.Key, msg, src, extra)
		return
	}

	p.Linef("return e, fmt.Errorf(\"env: %s: %s\", %s)", f.Key, msg, src)
}

// writeConstraintError emits the constraint violation error line.
func writeConstraintError(p *printer.Printer, f *model.Field, sensitiveMsg, msg, src, extra string) {
	if f.Sensitive {
		p.Linef("return e, fmt.Errorf(\"env: %s: %s\")", f.Key, sensitiveMsg)
		return
	}

	if extra != "" {
		p.Linef("return e, fmt.Errorf(\"env: %s: %s\", %s, %s)", f.Key, msg, src, extra)
		return
	}

	p.Linef("return e, fmt.Errorf(\"env: %s: %s\", %s)", f.Key, msg, src)
}

// writeTSConstraintError emits the constraint violation throw statement for the TypeScript generator.
func writeTSConstraintError(p *printer.Printer, f *model.Field, sensitiveMsg, fullMsg string) {
	if f.Sensitive {
		p.Linef("throw new Error(`env: %s: %s`)", f.Key, sensitiveMsg)
		return
	}

	p.Linef("throw new Error(`env: %s: %s`)", f.Key, fullMsg)
}

// JSEscape escapes s so it can be embedded in a JS string or template literal without changing the rendered text.
func JSEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `'`, `\'`, "`", "\\`", "${", "\\${").Replace(s)
}
