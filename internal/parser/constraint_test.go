package parser

import (
	"slices"
	"testing"

	"github.com/reglyph/g8n/internal/parser/constraints"
)

// TestConstraintOrderCoversRegistry keeps the decorator registry and the constraints.Order list in sync.
func TestConstraintOrderCoversRegistry(t *testing.T) {
	ordered := map[string]bool{}

	for _, c := range constraints.Order() {
		ordered[c.Name()] = true
	}

	var registryNames []string

	for _, d := range decorators {
		if !d.isConstraint {
			continue
		}

		if !ordered[d.name] {
			t.Errorf("constraint decorator %q is not present in constraints.Order", d.name)
		}

		registryNames = append(registryNames, d.name)
	}

	for name := range ordered {
		if !slices.Contains(registryNames, name) {
			t.Errorf("constraints.Order entry %q has no constraint-style decorator in the registry", name)
		}
	}
}
