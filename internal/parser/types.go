package parser

import "github.com/reglyph/g8n/internal/spec"

// Field describes one environment variable declared in the schema.
type Field struct {
	Key        string
	Kind       spec.Kind
	Required   bool
	Sensitive  bool
	HasDefault bool
	Default    string
	Enum       []string
	StartsWith string
	Regex      string
	Docs       []string
	Line       int
}

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
