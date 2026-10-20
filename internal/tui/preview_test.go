package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/palette"
)

// TestPreview writes smooth-style preview frames for every background and
// every event, in true colour and ANSI, to $TCAL_PREVIEW_DIR. View them with
// `less -R`. Skipped unless the variable is set.
func TestPreview(t *testing.T) {
	dir := os.Getenv("TCAL_PREVIEW_DIR")
	if dir == "" {
		t.Skip("set TCAL_PREVIEW_DIR to write preview frames")
	}
	const w, h = 120, 40
	bday := &events.MonthDay{Month: time.May, Day: 17}
	opts := BackgroundOptions{Style: StyleSmooth, Events: true, Birthday: bday}

	type shot struct {
		title, bg string
		at        time.Time
	}
	var shots []shot
	noon := time.Date(2026, 4, 15, 12, 0, 0, 0, time.Local)
	for _, name := range Backgrounds() {
		shots = append(shots, shot{name, name, noon})
	}
	shots = append(shots,
		shot{"sky at dusk", "sky", time.Date(2026, 4, 15, 18, 20, 0, 0, time.Local)},
		shot{"sky at night", "sky", time.Date(2024, 4, 23, 23, 0, 0, 0, time.Local)},
	)
	for _, occ := range events.Occurrences(2026, bday) {
		at := occ.Start.Add(time.Hour)
		switch occ.Kind {
		case events.NewYear:
			at = time.Date(2026, 12, 31, 23, 50, 0, 0, time.Local)
		case events.Onam:
			at = occ.End.Add(-2 * time.Hour) // full pookalam
		}
		shots = append(shots, shot{"event: " + occ.Kind.String(), "seasons", at})
	}

	for _, mode := range []struct {
		name string
		m    palette.Mode
	}{{"truecolor", palette.TrueColor}, {"ansi", palette.ANSI}} {
		var out []byte
		for _, s := range shots {
			m := localModel(w, h).
				WithBackgroundPalette(mode.m, palette.Dark).
				WithBackgroundTime(s.at).
				WithBackground(s.bg, opts)
			if m.bg == nil {
				t.Fatalf("%s: no background", s.title)
			}
			tick := time.Unix(0, 0)
			for range 90 { // 3 s at 30 fps
				tick = tick.Add(33 * time.Millisecond)
				next, _ := m.Update(bgTickMsg(tick))
				m = next.(Model)
			}
			hdr := fmt.Sprintf("===== %s (%s, %s) =====\n", s.title, s.at.Format("2006-01-02 15:04"), mode.name)
			out = append(out, hdr+m.View()+"\n\n"...)
		}
		if err := os.WriteFile(filepath.Join(dir, "smooth-"+mode.name+".txt"), out, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
