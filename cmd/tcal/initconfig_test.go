package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestInitConfig_DefaultsLoadLikeNoFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	want, err := configure(nil, io.Discard, cfgNow)
	if err != nil {
		t.Fatal(err)
	}
	var w configWritten
	if _, err := configure([]string{"--init-config"}, io.Discard, cfgNow); !errors.As(err, &w) {
		t.Fatalf("--init-config: %v", err)
	}
	path := filepath.Join(dir, "tcal", "config.toml")
	if w.path != path {
		t.Errorf("wrote %q, want %q", w.path, path)
	}
	got, err := configure(nil, io.Discard, cfgNow)
	if err != nil {
		t.Fatalf("generated config does not load: %v", err)
	}
	// The calendar state carries the wall clock; compare everything else.
	for _, rc := range []*runConfig{&got, &want} {
		rc.state.Anchor, rc.state.Today, rc.state.Now = time.Time{}, time.Time{}, time.Time{}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("generated config changes behaviour:\n got %+v\nwant %+v", got, want)
	}
}

func TestInitConfig_ListsEveryAllowedKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "tcal.toml")
	if _, err := configure([]string{"--init-config", "--config", path}, io.Discard, cfgNow); !errors.As(err, new(configWritten)) {
		t.Fatalf("--init-config --config: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, key := range configKeys(t) {
		if !strings.Contains(text, "\n"+key+" = ") {
			t.Errorf("generated config lacks %q", key)
		}
	}
	for _, key := range configForbidden {
		if strings.Contains(text, "\n"+key+" = ") {
			t.Errorf("generated config sets forbidden key %q", key)
		}
	}
	for _, want := range []string{"\n# birthday = ", "\n# [random]", "\n# backgrounds = "} {
		if !strings.Contains(text, want) {
			t.Errorf("generated config lacks commented %q", strings.TrimSpace(want))
		}
	}
}

func TestInitConfig_RefusesToOverwrite(t *testing.T) {
	path := writeCfg(t, "layout = \"grid\"\n")
	_, err := configure([]string{"--init-config", "--config", path}, io.Discard, cfgNow)
	if err == nil || errors.As(err, new(configWritten)) || !strings.Contains(err.Error(), path) {
		t.Fatalf("overwrite not refused: %v", err)
	}
	if body, _ := os.ReadFile(path); string(body) != "layout = \"grid\"\n" {
		t.Errorf("existing file changed: %q", body)
	}
}

func TestInitConfig_IgnoresBrokenExistingConfig(t *testing.T) {
	// A broken default config must not block writing a new file elsewhere.
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	os.MkdirAll(filepath.Join(dir, "tcal"), 0o755)
	os.WriteFile(filepath.Join(dir, "tcal", "config.toml"), []byte("layout = \"diagonal\"\n"), 0o644)
	other := filepath.Join(t.TempDir(), "fresh.toml")
	if _, err := configure([]string{"--init-config", "--config", other}, io.Discard, cfgNow); !errors.As(err, new(configWritten)) {
		t.Fatalf("%v", err)
	}
}

func TestInitConfig_NotAConfigKey(t *testing.T) {
	if !slices.Contains(configForbidden, "init-config") {
		t.Error("init-config must not be settable from the config file")
	}
}

// configKeys lists the flags the config file may set, read from --help so
// configure's flag list stays the source of truth.
func configKeys(t *testing.T) []string {
	t.Helper()
	var help bytes.Buffer
	configure([]string{"--help"}, &help, cfgNow)
	var keys []string
	for _, ln := range strings.Split(help.String(), "\n") {
		f := strings.Fields(ln)
		if len(f) == 0 || !strings.HasPrefix(f[0], "-") {
			continue
		}
		if k := strings.TrimPrefix(f[0], "-"); !slices.Contains(configForbidden, k) {
			keys = append(keys, k)
		}
	}
	if len(keys) < 10 {
		t.Fatalf("found only %d config keys in --help", len(keys))
	}
	return keys
}
