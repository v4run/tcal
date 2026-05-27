// Package golden provides a tiny golden-file test helper with an -update flag.
//
// Usage in a test:
//
//	golden.Assert(t, "testdata/april_sunday.golden", got)
//
// To refresh fixtures after an intentional render change:
//
//	go test ./... -update
package golden

import (
	"flag"
	"os"
	"path/filepath"
)

var update = flag.Bool("update", false, "update golden files in place")

// tb is the subset of testing.TB we use; the indirection lets tests fake it.
type tb interface {
	Helper()
	Errorf(format string, args ...any)
}

// Assert compares got to the contents of the file at path. When -update is
// passed to the test binary, Assert writes got to path instead of comparing.
func Assert(t tb, path, got string) {
	t.Helper()
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Errorf("golden: mkdir: %v", err)
			return
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Errorf("golden: write %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("golden: read %s: %v (run `go test ./... -update` to create)", path, err)
		return
	}
	if string(want) != got {
		t.Errorf("golden mismatch in %s\n--- want ---\n%s\n--- got ---\n%s", path, string(want), got)
	}
}
