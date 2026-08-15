package jsonschema

import (
	"encoding/json"
	"fmt"

	"github.com/reglyph/g8n/internal/parser"
	"github.com/reglyph/g8n/internal/spec"
)

// Field describes the JSON Schema constraints of one environment variable.
type Field struct {
	Type        string          `json:"type"`
	Description string          `json:"description,omitempty"`
	Default     json.RawMessage `json:"default,omitempty"`
	Enum        []string        `json:"enum,omitempty"`
	Format      string          `json:"format,omitempty"`
	Pattern     string          `json:"pattern,omitempty"`
	Min         *float64        `json:"minimum,omitempty"`
	Max         *float64        `json:"maximum,omitempty"`
}

// Schema is a JSON Schema draft-07 document for an env schema.
type Schema struct {
	Schema     string            `json:"$schema"`
	Title      string            `json:"title,omitempty"`
	Type       string            `json:"type"`
	Required   []string          `json:"required,omitempty"`
	Properties map[string]*Field `json:"properties"`
}

// Generate renders the schema as a JSON Schema document.
func Generate(s *parser.Schema) ([]byte, error) {
	doc := &Schema{
		Schema:     "http://json-schema.org/draft-07/schema#",
		Title:      s.SourcePath,
		Type:       "object",
		Properties: map[string]*Field{},
	}

	for _, f := range s.Fields {
		if f.Required {
			doc.Required = append(doc.Required, f.Key)
		}

		fs, err := fieldSchema(f)
		if err != nil {
			return nil, err
		}

		doc.Properties[f.Key] = fs
	}

	return json.MarshalIndent(doc, "", "  ")
}

func fieldSchema(f *parser.Field) (*Field, error) {
	sp := f.Kind.Spec()

	out := &Field{Type: sp.JSONType}
	if sp.JSONFormat != "" {
		out.Format = sp.JSONFormat
	}

	if len(f.Docs) > 0 {
		out.Description = f.Docs[0]
	}

	if f.HasDefault && !f.Sensitive {
		def, err := defaultJSON(f)
		if err != nil {
			return nil, err
		}

		out.Default = def
	}

	if f.Kind.IsPort() {
		portMin, portMax := float64(spec.PortMin), float64(spec.PortMax)
		out.Min, out.Max = &portMin, &portMax
	}

	for _, c := range parser.BuildConstraints(f) {
		cs, err := c.Schema(f)
		if err != nil {
			return nil, err
		}

		mergeSchemaKeywords(out, cs)
	}

	if f.Kind == spec.KindEnum {
		out.Enum = f.Enum
	}

	return out, nil
}

func mergeSchemaKeywords(out *Field, cs parser.FieldSchema) {
	if cs.Pattern == "" {
		return
	}

	if out.Pattern != "" {
		out.Pattern = "(?=" + out.Pattern + ")(?:" + cs.Pattern + ")"
	} else {
		out.Pattern = cs.Pattern
	}
}

// defaultJSON encodes the schema default with its JSON type.
func defaultJSON(f *parser.Field) (json.RawMessage, error) {
	lit, err := f.Kind.ParseLiteral(f.Default)
	if err != nil {
		return nil, fmt.Errorf("variable %s: %w", f.Key, err)
	}

	switch f.Kind {
	case spec.KindInt, spec.KindPort, spec.KindInt64:
		return json.Marshal(lit.Int)
	case spec.KindBool:
		return json.Marshal(lit.Bool)
	case spec.KindFloat64:
		return json.Marshal(lit.Float)
	default:
		return json.Marshal(lit.Str)
	}
}
