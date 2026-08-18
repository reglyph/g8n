//go:build unix

package typescript

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func lock(t *testing.T) func() {
	t.Helper()

	//nolint:mnd // 0o600 lock file permissions
	f, err := os.OpenFile(filepath.Join(os.TempDir(), "g8n-npx.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		//nolint:errcheck // best-effort close on a test temp file
		_ = f.Close()

		t.Fatal(err)
	}

	return func() {
		//nolint:errcheck // best-effort unlock on a test temp file
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		//nolint:errcheck // best-effort close on a test temp file
		_ = f.Close()
	}
}
