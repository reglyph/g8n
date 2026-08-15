package parser

import (
	"github.com/reglyph/g8n/internal/model"
	"github.com/reglyph/g8n/internal/parser/constraints"
)

// Field is the environment variable declaration, defined in internal/model.
type Field = model.Field

// Constraint is the decorator-backed rule on a field, defined in
// internal/parser/constraints.
type Constraint = constraints.Constraint

// FieldSchema describes the JSON Schema keywords a constraint
// contributes, defined in internal/parser/constraints.
type FieldSchema = constraints.FieldSchema

// Schema is the parsed representation of an .env.schema file.
type Schema struct {
	Package    string
	OutPath    string
	SourcePath string
	Fields     []*Field
	Warnings   []string
}

// FieldByKey returns the field with the given key, or nil.
func (s *Schema) FieldByKey(key string) *Field {
	for _, f := range s.Fields {
		if f.Key == key {
			return f
		}
	}

	return nil
}
