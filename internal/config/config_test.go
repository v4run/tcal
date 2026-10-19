package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var allowed = []string{"layout", "months", "with-clock", "no-color", "events"}

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDefaultPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/x/cfg")
	if got, want := DefaultPath(), "/x/cfg/tcal/config.toml"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/h")
	if got, want := DefaultPath(), "/h/.config/tcal/config.toml"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLoad_MissingDefaultIsEmpty(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nope.toml"), false, allowed)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Flags) != 0 || c.Birthday != "" || c.RandomPool != nil {
		t.Errorf("want empty config, got %+v", c)
	}
}

func TestLoad_MissingExplicitErrors(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.toml"), true, allowed); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_UnknownKeyErrors(t *testing.T) {
	p := write(t, "colour = \"red\"\n")
	_, err := Load(p, false, allowed)
	if err == nil || !strings.Contains(err.Error(), "colour") || !strings.Contains(err.Error(), p) {
		t.Fatalf("got %v", err)
	}
	p = write(t, "[random]\nfoo = 1\n")
	if _, err := Load(p, false, allowed); err == nil || !strings.Contains(err.Error(), "random.foo") {
		t.Fatalf("got %v", err)
	}
}

func TestLoad_ForbiddenKeyErrors(t *testing.T) {
	p := write(t, "print = true\n")
	_, err := Load(p, false, allowed)
	if err == nil || !strings.Contains(err.Error(), "print") || !strings.Contains(err.Error(), p) {
		t.Fatalf("got %v", err)
	}
}

func TestLoad_NonScalarErrors(t *testing.T) {
	p := write(t, "layout = [\"grid\"]\n")
	if _, err := Load(p, false, allowed); err == nil || !strings.Contains(err.Error(), "layout") {
		t.Fatalf("got %v", err)
	}
}

func TestLoad_RandomPool(t *testing.T) {
	p := write(t, "birthday = \"05-17\"\n[random]\nbackgrounds = [\"rain\", \"life\"]\n")
	c, err := Load(p, false, allowed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c.RandomPool, []string{"rain", "life"}) || c.Birthday != "05-17" || c.Path != p {
		t.Errorf("got %+v", c)
	}
}

func TestLoad_BadPoolErrors(t *testing.T) {
	for _, body := range []string{
		"[random]\nbackgrounds = [1, 2]\n",
		"[random]\nbackgrounds = \"rain\"\n",
	} {
		if _, err := Load(write(t, body), false, allowed); err == nil {
			t.Errorf("%q: expected error", body)
		}
	}
}

func TestLoad_EmptyPoolErrors(t *testing.T) {
	p := write(t, "[random]\nbackgrounds = []\n")
	_, err := Load(p, false, allowed)
	if err == nil || !strings.Contains(err.Error(), "no backgrounds to pick from") || !strings.Contains(err.Error(), p) {
		t.Fatalf("got %v", err)
	}
}

func TestLoad_ScalarFormatting(t *testing.T) {
	p := write(t, "with-clock = true\nmonths = 6\nlayout = \"grid\"\n")
	c, err := Load(p, false, allowed)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"with-clock": "true", "months": "6", "layout": "grid"}
	if !reflect.DeepEqual(c.Flags, want) {
		t.Errorf("got %v, want %v", c.Flags, want)
	}
}
