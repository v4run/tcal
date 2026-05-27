package printout

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/varun/tcal/internal/render"
)

func TestRun_WritesGrid(t *testing.T) {
	state := render.State{
		Anchor: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Now:    time.Date(2026, 4, 22, 14, 32, 7, 0, time.UTC),
		Layout: render.LayoutGrid,
		Width:  80,
		Height: 0, // print mode
	}
	opts := render.Options{
		WeekStart: time.Sunday,
		Highlight: render.HighlightNone,
	}

	var buf bytes.Buffer
	if err := Run(state, opts, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.Contains(out, "January 2026") || !strings.Contains(out, "December 2026") {
		t.Error("grid output should contain January and December 2026")
	}
	if !strings.HasSuffix(out, "\n") {
		t.Error("output should end with newline")
	}
}
