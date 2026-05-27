package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/varun/tcal/internal/printout"
	"github.com/varun/tcal/internal/render"
	"github.com/varun/tcal/internal/tui"
)

const version = "0.1.0"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, errVersion) || errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "tcal:", err)
		os.Exit(2)
	}
}

var errVersion = errors.New("--version")

func run(args []string, stdout, stderr *os.File) error {
	fs := flag.NewFlagSet("tcal", flag.ContinueOnError)
	fs.SetOutput(stderr)

	layout := fs.String("layout", "focus", "horizontal|vertical|grid|focus")
	date := fs.String("date", "", "YYYY-MM or YYYY-MM-DD (default today)")
	year := fs.Int("year", 0, "shorthand for --date=YYYY-01")
	months := fs.Int("months", 0, "number of months (0 = layout default)")
	weekStart := fs.String("week-start", "sun", "sun|mon")
	highlight := fs.String("highlight", "combined", "combined|reverse|bracket|color|none")
	noColor := fs.Bool("no-color", false, "disable ANSI color (also honors NO_COLOR env)")
	print := fs.Bool("print", false, "one-shot render to stdout, no TUI")
	width := fs.Int("width", 0, "terminal width (print mode only)")
	withClock := fs.Bool("with-clock", false, "include block clock in print mode")
	clockStyle := fs.String("clock-style", "block", "block|inline|boxed")
	showVersion := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Fprintln(stdout, "tcal", version)
		return errVersion
	}

	state, opts, err := buildState(*layout, *date, *year, *months, *weekStart, *highlight, *clockStyle, *noColor, *withClock)
	if err != nil {
		return err
	}

	if *print {
		state.Height = 0
		state.Width = resolveWidth(*width, stdout)
		return printout.Run(state, opts, stdout)
	}

	state.Width, state.Height = currentTermSize(stdout)
	model := tui.New(state, opts)
	prog := tea.NewProgram(model, tea.WithAltScreen())
	_, err = prog.Run()
	return err
}

func buildState(layoutS, dateS string, year, months int, weekStartS, highlightS, clockStyleS string, noColor, withClock bool) (render.State, render.Options, error) {
	now := time.Now()
	anchor := now

	switch {
	case dateS != "":
		t, err := parseDate(dateS)
		if err != nil {
			return render.State{}, render.Options{}, err
		}
		anchor = t
	case year != 0:
		anchor = time.Date(year, 1, 1, 0, 0, 0, 0, now.Location())
	}

	layout, err := parseLayout(layoutS)
	if err != nil {
		return render.State{}, render.Options{}, err
	}
	weekStart, err := parseWeekStart(weekStartS)
	if err != nil {
		return render.State{}, render.Options{}, err
	}
	highlight, err := parseHighlight(highlightS)
	if err != nil {
		return render.State{}, render.Options{}, err
	}
	clockStyle, err := parseClockStyle(clockStyleS)
	if err != nil {
		return render.State{}, render.Options{}, err
	}

	color := !noColor && os.Getenv("NO_COLOR") == ""

	return render.State{
			Anchor: anchor,
			Today:  now,
			Now:    now,
			Layout: layout,
			Months: months,
		},
		render.Options{
			WeekStart:  weekStart,
			Color:      color,
			Highlight:  highlight,
			ClockStyle: clockStyle,
			WithClock:  withClock,
		}, nil
}

func parseDate(s string) (time.Time, error) {
	formats := []string{"2006-01-02", "2006-01"}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf(`invalid --date %q: expected YYYY-MM or YYYY-MM-DD`, s)
}

func parseLayout(s string) (render.Layout, error) {
	switch strings.ToLower(s) {
	case "horizontal", "h":
		return render.LayoutHorizontal, nil
	case "vertical", "v":
		return render.LayoutVertical, nil
	case "grid", "g":
		return render.LayoutGrid, nil
	case "focus", "f":
		return render.LayoutFocus, nil
	}
	return 0, fmt.Errorf("unknown --layout %q (expected horizontal|vertical|grid|focus)", s)
}

func parseWeekStart(s string) (time.Weekday, error) {
	switch strings.ToLower(s) {
	case "sun", "sunday":
		return time.Sunday, nil
	case "mon", "monday":
		return time.Monday, nil
	}
	return 0, fmt.Errorf("unknown --week-start %q (expected sun|mon)", s)
}

func parseHighlight(s string) (render.HighlightMask, error) {
	switch strings.ToLower(s) {
	case "combined":
		return render.HighlightAll, nil
	case "reverse":
		return render.HighlightReverse, nil
	case "bracket":
		return render.HighlightBracket, nil
	case "color":
		return render.HighlightColor, nil
	case "none":
		return render.HighlightNone, nil
	}
	return 0, fmt.Errorf("unknown --highlight %q (expected combined|reverse|bracket|color|none)", s)
}

func parseClockStyle(s string) (render.ClockStyle, error) {
	switch strings.ToLower(s) {
	case "block":
		return render.ClockBlock, nil
	case "inline":
		return render.ClockInline, nil
	case "boxed":
		return render.ClockBoxed, nil
	}
	return 0, fmt.Errorf("unknown --clock-style %q (expected block|inline|boxed)", s)
}

func resolveWidth(flagW int, f *os.File) int {
	if flagW > 0 {
		return flagW
	}
	if w, _, err := term.GetSize(int(f.Fd())); err == nil && w > 0 {
		return w
	}
	return 80
}

func currentTermSize(f *os.File) (int, int) {
	if w, h, err := term.GetSize(int(f.Fd())); err == nil && w > 0 && h > 0 {
		return w, h
	}
	return 80, 24
}
