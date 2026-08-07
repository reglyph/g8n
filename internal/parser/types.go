package parser

import "github.com/whoqmi/g8n/internal/spec"

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

type Schema struct {
	Package    string
	OutPath    string
	SourcePath string
	Fields     []*Field
	Warnings   []string
}

func (s *Schema) FieldByKey(key string) *Field {
	for _, f := range s.Fields {
		if f.Key == key {
			return f
		}
	}

	return nil
}
