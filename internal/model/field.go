// Package model holds the core domain types shared across g8n packages.
package model

import "github.com/reglyph/g8n/internal/spec"

// Field describes one environment variable declared in the schema.
type Field struct {
	Key         string
	Kind        spec.Kind
	Required    bool
	Sensitive   bool
	HasDefault  bool
	Default     string
	Enum        []string
	StartsWith  string
	Regex       string
	Docs        []string
	Source      string
	SourceSpecs []spec.SourceSpec
	Line        int
}

// HasRegex reports whether the field declares a @regex constraint.
func (f *Field) HasRegex() bool {
	return f.Regex != ""
}
