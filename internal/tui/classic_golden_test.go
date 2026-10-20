package tui

import (
	"flag"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var update = flag.Bool("update", false, "rewrite golden files")

// goldenTick is the background tick message sent at time at in a golden run.
func goldenTick(at time.Time) tea.Msg { return bgTickMsg(at) }

// TestClassicGolden pins the classic style's output: every background, built
// with a fixed seed and a fixed local time, must render exactly as recorded.
func TestClassicGolden(t *testing.T) {
	const w, h = 100, 30
	start := time.Date(2026, time.April, 15, 10, 0, 0, 0, time.Local)
	for _, name := range Backgrounds() {
		t.Run(name, func(t *testing.T) {
			m := runnerModel(w, h)
			m.state.Anchor = time.Date(2026, time.April, 1, 0, 0, 0, 0, time.Local)
			m.state.Today, m.state.Now = start, start
			m.bg = backgrounds[name](w, h, rand.New(rand.NewPCG(1, 2)), BackgroundOptions{})
			m.syncClock()
			at := start
			for range 20 {
				at = at.Add(m.bg.Interval())
				next, _ := m.Update(goldenTick(at))
				m = next.(Model)
			}
			got := m.View()

			path := filepath.Join("testdata", "classic", name+".golden")
			if *update {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (run with -update to create)", err)
			}
			if got != string(want) {
				t.Errorf("%s: classic output changed\n got: %q\nwant: %q", name, got, want)
			}
		})
	}
}
