package constraints

import (
	"fmt"

	"github.com/reglyph/g8n/internal/model"
	"github.com/reglyph/g8n/internal/printer"
)

// FieldSchema describes the JSON Schema keywords a constraint contributes.
type FieldSchema struct {
	Pattern string
}

// Constraint is one self-contained decorator-backed rule on a field.
type Constraint interface {
	Name() string

	Present(f *model.Field) bool

	// Validate checks the constraint definition itself, regardless of the field default. It runs before any default comparison.
	Validate(f *model.Field) error

	// ValidateDefault checks the field default against the constraint
	ValidateDefault(f *model.Field) error

	// Emit writes the runtime check into the generated load function
	Emit(p *printer.Printer, f *model.Field, src, rxVar string)

	// Schema returns the JSON Schema keywords contribusted by the constraint
	Schema(f *model.Field) (FieldSchema, error)
}

// fieldCtx prefixes validation messages with the field location.
func fieldCtx(f *model.Field) string {
	return fmt.Sprintf("line %d: variable %s", f.Line, f.Key)
}
