//go:build !unix

package typescript

import "testing"

func lock(t *testing.T) func() {
	t.Helper()

	return func() {}
}
