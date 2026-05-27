package golden

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAssert_PassesWhenContentMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.golden")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	Assert(t, path, "hello\n")
}

func TestAssert_FailsWhenContentDiffers(t *testing.T) {
	if *update {
		t.Skip("skipping mismatch-detection test in -update mode")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.golden")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fake := &recordingT{}
	Assert(fake, path, "goodbye\n")
	if !fake.failed {
		t.Error("expected Assert to fail on mismatch")
	}
}

type recordingT struct {
	testing.TB
	failed bool
}

func (r *recordingT) Errorf(format string, args ...any) { r.failed = true }
func (r *recordingT) Helper()                           {}
