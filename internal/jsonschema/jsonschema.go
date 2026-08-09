package jsonschema

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/whoqmi/g8n/internal/parser"
	"github.com/whoqmi/g8n/internal/spec"
)

// Field describes the JSON Schema constraints of one environment variable.
type Field struct {
	Type        string          `json:"type"`
	Description string          `json:"description,omitempty"`
	Default     json.RawMessage `json:"default,omitempty"`
	Enum        []string        `json:"enum,omitempty"`
	Format      string          `json:"format,omitempty"`
	Pattern     string          `json:"pattern,omitempty"`
	Min         *int            `json:"minimum,omitempty"`
	Max         *int            `json:"maximum,omitempty"`
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
		portMin, portMax := spec.PortMin, spec.PortMax
		out.Min, out.Max = &portMin, &portMax
	}

	if f.Kind == spec.KindEnum {
		out.Enum = f.Enum
	}

	if f.StartsWith != "" {
		out.Pattern = "^" + regexp.QuoteMeta(f.StartsWith)
	}

	if f.Regex != "" {
		if out.Pattern != "" {
			out.Pattern = "(?:" + out.Pattern + ")|(?:" + f.Regex + ")"
		} else {
			out.Pattern = f.Regex
		}
	}

	return out, nil
}

// defaultJSON encodes the schema default with its JSON type.
func defaultJSON(f *parser.Field) (json.RawMessage, error) {
	//goland:noinspection GoSwitchMissingCasesForIotaConsts
	switch f.Kind {
	case spec.KindInt, spec.KindPort, spec.KindInt64:
		n, err := strconv.ParseInt(f.Default, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid default %q for variable %s: not an integer", f.Default, f.Key)
		}

		return json.Marshal(n)

	case spec.KindBool:
		v, err := strconv.ParseBool(f.Default)
		if err != nil {
			return nil, fmt.Errorf("invalid default %q for variable %s: not a boolean", f.Default, f.Key)
		}

		return json.Marshal(v)

	case spec.KindFloat64:
		v, err := strconv.ParseFloat(f.Default, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid default %q for variable %s: not a float64", f.Default, f.Key)
		}

		return json.Marshal(v)
	}

	b, err := json.Marshal(f.Default)
	if err != nil {
		return nil, fmt.Errorf("invalid default %q for variable %s: %w", f.Default, f.Key, err)
	}

	return b, nil
}
