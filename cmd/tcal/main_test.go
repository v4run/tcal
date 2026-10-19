package main

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"testing"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/render"
)

func TestParseBgTime(t *testing.T) {
	now := time.Date(2026, 10, 19, 9, 15, 42, 0, time.UTC)
	cases := []struct {
		in   string
		want time.Time
	}{
		{"", now},
		{"22:30", time.Date(2026, 10, 19, 22, 30, 0, 0, time.UTC)},
		{"2024-04-23", time.Date(2024, 4, 23, 9, 15, 42, 0, time.UTC)},
		{"2024-04-23T22:30", time.Date(2024, 4, 23, 22, 30, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		got, err := parseBgTime(tc.in, now)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if !got.Equal(tc.want) {
			t.Errorf("%q: got %v, want %v", tc.in, got, tc.want)
		}
	}
	for _, bad := range []string{"25:00", "noon", "2024-13-01", "2024-04-23 22:30"} {
		if _, err := parseBgTime(bad, now); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}

func writeCfg(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func emptyXDG(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

var cfgNow = time.Date(2026, 10, 19, 9, 15, 0, 0, time.UTC)

func TestConfigure_NoHomeIsNoFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	rc, err := configure(nil, io.Discard, cfgNow)
	if err != nil {
		t.Fatal(err)
	}
	if rc.bg != "" || !rc.bgOpts.Events || rc.bgOpts.Birthday != nil || rc.bgOpts.RandomPool != nil {
		t.Errorf("defaults changed: %+v", rc)
	}
}

func TestConfigure_NoFileUnchanged(t *testing.T) {
	emptyXDG(t)
	rc, err := configure(nil, io.Discard, cfgNow)
	if err != nil {
		t.Fatal(err)
	}
	if rc.state.Layout != render.LayoutHorizontal || rc.opts.WeekStart != time.Sunday ||
		rc.opts.Highlight != render.HighlightAll || rc.opts.ClockStyle != render.ClockBlock ||
		rc.opts.WithClock || rc.print || rc.bg != "" || rc.bgTime != nil || rc.bgMono ||
		!rc.bgOpts.Events || rc.bgOpts.Birthday != nil || rc.bgOpts.RandomPool != nil {
		t.Errorf("defaults changed: %+v", rc)
	}
}

func TestConfigure_ConfigSetsDefault(t *testing.T) {
	p := writeCfg(t, "layout = \"grid\"\nwith-clock = true\nbg = \"rain\"\nbg-color = false\n")
	rc, err := configure([]string{"--config", p}, io.Discard, cfgNow)
	if err != nil {
		t.Fatal(err)
	}
	if rc.state.Layout != render.LayoutGrid || !rc.opts.WithClock || rc.bg != "rain" || !rc.bgMono {
		t.Errorf("got %+v", rc)
	}
}

func TestConfigure_FlagBeatsConfig(t *testing.T) {
	p := writeCfg(t, "layout = \"grid\"\n")
	rc, err := configure([]string{"--config", p, "--layout", "horizontal"}, io.Discard, cfgNow)
	if err != nil {
		t.Fatal(err)
	}
	if rc.state.Layout != render.LayoutHorizontal {
		t.Errorf("layout = %v", rc.state.Layout)
	}
}

func TestConfigure_InvalidValueNamesFile(t *testing.T) {
	p := writeCfg(t, "layout = \"vertical\"\n")
	_, err := configure([]string{"--config", p}, io.Discard, cfgNow)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, s := range []string{p, "layout", "vertical"} {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("error %q missing %q", err, s)
		}
	}
}

func TestConfigure_InvalidTypeNamesFile(t *testing.T) {
	p := writeCfg(t, "months = \"six\"\n")
	_, err := configure([]string{"--config", p}, io.Discard, cfgNow)
	if err == nil || !strings.Contains(err.Error(), "months") {
		t.Fatalf("got %v", err)
	}
}

func TestConfigure_ForbiddenKey(t *testing.T) {
	p := writeCfg(t, "bg-time = \"22:00\"\n")
	if _, err := configure([]string{"--config", p}, io.Discard, cfgNow); err == nil {
		t.Fatal("expected error")
	}
}

func TestConfigure_Birthday(t *testing.T) {
	p := writeCfg(t, "birthday = \"05-17\"\n")
	rc, err := configure([]string{"--config", p}, io.Discard, cfgNow)
	if err != nil {
		t.Fatal(err)
	}
	if rc.bgOpts.Birthday == nil || *rc.bgOpts.Birthday != (events.MonthDay{Month: time.May, Day: 17}) {
		t.Errorf("birthday = %v", rc.bgOpts.Birthday)
	}
	p = writeCfg(t, "birthday = \"13-45\"\n")
	if _, err := configure([]string{"--config", p}, io.Discard, cfgNow); err == nil || !strings.Contains(err.Error(), p) {
		t.Errorf("got %v", err)
	}
}

func TestConfigure_RandomPool(t *testing.T) {
	p := writeCfg(t, "[random]\nbackgrounds = [\"rain\", \"life\"]\n")
	rc, err := configure([]string{"--config", p}, io.Discard, cfgNow)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(rc.bgOpts.RandomPool, []string{"rain", "life"}) {
		t.Errorf("pool = %v", rc.bgOpts.RandomPool)
	}
}

func TestConfigure_RandomPoolUnknownName(t *testing.T) {
	for _, body := range []string{
		"[random]\nbackgrounds = [\"rain\", \"nope\"]\n",
		"[random]\nbackgrounds = [\"random\"]\n",
		"[random]\nbackgrounds = []\n",
	} {
		p := writeCfg(t, body)
		_, err := configure([]string{"--config", p}, io.Discard, cfgNow)
		if err == nil || !strings.Contains(err.Error(), p) {
			t.Errorf("%q: got %v", body, err)
		}
	}
}

func TestConfigure_EventsFalseFromConfig(t *testing.T) {
	p := writeCfg(t, "events = false\n")
	rc, err := configure([]string{"--config", p}, io.Discard, cfgNow)
	if err != nil {
		t.Fatal(err)
	}
	if rc.bgOpts.Events {
		t.Error("events should be false")
	}
	rc, err = configure([]string{"--config", p, "--events"}, io.Discard, cfgNow)
	if err != nil || !rc.bgOpts.Events {
		t.Errorf("flag should beat config: %v %v", rc.bgOpts.Events, err)
	}
}

func TestConfigure_ExplicitConfigFlag(t *testing.T) {
	emptyXDG(t)
	missing := filepath.Join(t.TempDir(), "missing.toml")
	if _, err := configure([]string{"--config", missing}, io.Discard, cfgNow); err == nil {
		t.Error("explicit missing config should error")
	}
	// A default-location file is read when --config is absent.
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	os.MkdirAll(filepath.Join(dir, "tcal"), 0o755)
	os.WriteFile(filepath.Join(dir, "tcal", "config.toml"), []byte("layout = \"grid\"\n"), 0o644)
	rc, err := configure(nil, io.Discard, cfgNow)
	if err != nil || rc.state.Layout != render.LayoutGrid {
		t.Errorf("default file not used: %v %v", rc.state.Layout, err)
	}
}

func TestConfigure_BgTime(t *testing.T) {
	emptyXDG(t)
	rc, err := configure([]string{"--bg-time", "22:30"}, io.Discard, cfgNow)
	if err != nil || rc.bgTime == nil || rc.bgTime.Hour() != 22 {
		t.Errorf("got %v %v", rc.bgTime, err)
	}
}

func TestConfigure_OverriddenValueStillValidated(t *testing.T) {
	p := writeCfg(t, "layout = \"gird\"\n")
	_, err := configure([]string{"--config", p, "--layout", "grid"}, io.Discard, cfgNow)
	if err == nil || !strings.Contains(err.Error(), "gird") {
		t.Fatalf("got %v, want error naming the bad config value", err)
	}
}

func TestConfigure_RandomPoolCaseInsensitive(t *testing.T) {
	p := writeCfg(t, "[random]\nbackgrounds = [\"Sky\"]\n")
	rc, err := configure([]string{"--config", p}, io.Discard, cfgNow)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(rc.bgOpts.RandomPool, []string{"sky"}) {
		t.Errorf("pool = %v", rc.bgOpts.RandomPool)
	}
}
