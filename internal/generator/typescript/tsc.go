package typescript

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

// TSCompile type-checks the file with the TypeScript compiler, preferring a local tsc and falling back to npx.
// The npx install is serialized with a flock because parallel test packages racing it corrupt the npx cache.
func TSCompile(t *testing.T, args ...string) []byte {
	t.Helper()

	if p, err := exec.LookPath("tsc"); err == nil {
		// #nosec G204 -- fixed args, only the tsc path varies
		return compile(t, exec.Command(p, args...))
	}

	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("neither tsc nor npx available")
	}

	unlock := lock(t)
	defer unlock()

	// #nosec G204 -- fixed args, only the temp file path varies
	cmd := exec.Command("npx", append([]string{"-p", "typescript", "tsc"}, args...)...)

	return compile(t, cmd)
}

func compile(t *testing.T, cmd *exec.Cmd) []byte {
	t.Helper()

	// #nosec G204 -- fixed args, only the temp file path varies
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tsc: %v\n%s", err, out)
	}

	return out
}

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
