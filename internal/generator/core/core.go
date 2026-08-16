// Package core holds the checks shared by all language generators.
package core

import (
	"fmt"

	"github.com/reglyph/g8n/internal/parser"
)

// CheckFieldNameCollisions reports two variables whose generated field names collide under the given naming scheme.
func CheckFieldNameCollisions(s *parser.Schema, fieldName func(string) string) error {
	seen := make(map[string]*parser.Field, len(s.Fields))

	for _, f := range s.Fields {
		name := fieldName(f.Key)

		if first, ok := seen[name]; ok {
			return fmt.Errorf("line %d: variable %q: field name %q collides with variable %q declared on line %d",
				f.Line, f.Key, name, first.Key, first.Line)
		}

		seen[name] = f
	}

	return nil
}
