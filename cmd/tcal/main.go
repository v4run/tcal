package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/v4run/tcal/internal/config"
	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/palette"
	"github.com/v4run/tcal/internal/printout"
	"github.com/v4run/tcal/internal/render"
	"github.com/v4run/tcal/internal/tui"
)

// fallbackVersion is reported only when build info is unavailable
// (which shouldn't happen for Go ≥ 1.18 binaries). `go install` users
// get the actual module version from runtime/debug.ReadBuildInfo.
const fallbackVersion = "v0.1.0"

func versionString() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return fallbackVersion
}

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

// runConfig is everything run needs after flags and config are resolved.
type runConfig struct {
	state  render.State
	opts   render.Options
	print  bool
	width  int
	bg     string
	bgOpts tui.BackgroundOptions
	bgMono bool
	bgTime *time.Time // nil when --bg-time is unset
	// bgStyle is tui.StyleClassic or tui.StyleSmooth (also in bgOpts.Style).
	bgStyle string
	// bgPalette is the colour depth for true-colour backgrounds.
	bgPalette palette.Mode
}

// getenv reads the environment; tests replace it.
var getenv = os.Getenv

// configForbidden lists flags the config file may never set.
var configForbidden = []string{"date", "year", "print", "bg-time", "version", "config", "init-config"}

func run(args []string, stdout, stderr *os.File) error {
	rc, err := configure(args, stderr, time.Now())
	if err != nil {
		if errors.Is(err, errVersion) {
			fmt.Fprintln(stdout, "tcal", versionString())
		}
		if w, ok := errors.AsType[configWritten](err); ok {
			fmt.Fprintln(stdout, "tcal: wrote", w.path)
			return nil
		}
		return err
	}

	if rc.print {
		rc.state.Height = 0
		rc.state.Width = resolveWidth(rc.width, stdout)
		return printout.Run(rc.state, rc.opts, stdout)
	}

	rc.state.Width, rc.state.Height = currentTermSize(stdout)
	model := tui.New(rc.state, rc.opts)
	// The background time must be set first: Random consults it.
	if rc.bgTime != nil {
		model = model.WithBackgroundTime(*rc.bgTime)
	}
	if rc.bg != "" {
		model = model.WithBackground(rc.bg, rc.bgOpts)
	}
	if rc.bgMono {
		model = model.WithBackgroundMono()
	}
	if rc.bg != "" {
		// Ask the terminal for its background once, before Bubble Tea owns it.
		// Only smooth, coloured output needs it; the query would change
		// classic startup.
		termBg := palette.Dark
		if rc.bgStyle == tui.StyleSmooth && !rc.bgMono && rc.opts.Color && !lipgloss.HasDarkBackground() {
			termBg = palette.Light
		}
		model = model.WithBackgroundPalette(rc.bgPalette, termBg)
	}
	progOpts := []tea.ProgramOption{tea.WithAltScreen()}
	if rc.bgStyle == tui.StyleSmooth {
		// Bubble Tea v1.3.10 never emits DEC 2026; bracket each frame flush.
		progOpts = append(progOpts, tea.WithOutput(tui.NewSyncWriter(os.Stdout)))
	}
	prog := tea.NewProgram(model, progOpts...)
	_, err = prog.Run()
	return err
}

// configure parses args, layers the config file under them (flags win) and
// validates everything.
func configure(args []string, stderr io.Writer, now time.Time) (runConfig, error) {
	fs := flag.NewFlagSet("tcal", flag.ContinueOnError)
	fs.SetOutput(stderr)

	layout := fs.String("layout", "horizontal", "horizontal|grid")
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
	bgS := fs.String("bg", "none", "ambient background in live mode: none|random|"+strings.Join(tui.Backgrounds(), "|"))
	bgStyleS := fs.String("bg-style", tui.StyleClassic, "background style: "+strings.Join(tui.Styles(), "|"))
	bgPaletteS := fs.String("bg-palette", "auto", "background colors: auto|truecolor|ansi")
	bgColor := fs.Bool("bg-color", true, "tint the background; false draws it dim in the foreground color")
	bgTimeS := fs.String("bg-time", "", "time seen by time-aware backgrounds: HH:MM, YYYY-MM-DD or YYYY-MM-DDTHH:MM")
	showEvents := fs.Bool("events", true, "show holiday scenes in the seasons background")
	configPath := fs.String("config", "", "config file (default ~/.config/tcal/config.toml)")
	initConfig := fs.Bool("init-config", false, "write a config file with every default (to --config or the default location), then exit")
	showVersion := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		return runConfig{}, err
	}
	if *showVersion {
		return runConfig{}, errVersion
	}
	if *initConfig {
		path := *configPath
		if path == "" {
			path = config.DefaultPath()
		}
		return runConfig{}, writeConfig(path, fs)
	}

	setOnCLI := map[string]bool{}
	var allowed []string
	fs.Visit(func(f *flag.Flag) { setOnCLI[f.Name] = true })
	fs.VisitAll(func(f *flag.Flag) {
		if !slices.Contains(configForbidden, f.Name) {
			allowed = append(allowed, f.Name)
		}
	})

	path, mustExist := *configPath, true
	if path == "" {
		path, mustExist = config.DefaultPath(), false
	}
	cfg := config.Config{Flags: map[string]string{}}
	if path != "" { // no default path (no home directory): behave as if no file exists
		var err error
		if cfg, err = config.Load(path, mustExist, allowed); err != nil {
			return runConfig{}, err
		}
	}

	validators := map[string]func(string) error{
		"layout":      func(s string) error { _, err := parseLayout(s); return err },
		"week-start":  func(s string) error { _, err := parseWeekStart(s); return err },
		"highlight":   func(s string) error { _, err := parseHighlight(s); return err },
		"clock-style": func(s string) error { _, err := parseClockStyle(s); return err },
		"bg":          func(s string) error { _, err := parseBackground(s); return err },
		"bg-style":    func(s string) error { _, err := parseBgStyle(s); return err },
		"bg-palette":  func(s string) error { _, err := parseBgPalette(s); return err },
		"no-color":    func(s string) error { _, err := strconv.ParseBool(s); return err },
		"with-clock":  func(s string) error { _, err := strconv.ParseBool(s); return err },
		"bg-color":    func(s string) error { _, err := strconv.ParseBool(s); return err },
		"events":      func(s string) error { _, err := strconv.ParseBool(s); return err },
		"months":      func(s string) error { _, err := strconv.Atoi(s); return err },
		"width":       func(s string) error { _, err := strconv.Atoi(s); return err },
	}
	keys := make([]string, 0, len(cfg.Flags))
	for k := range cfg.Flags {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		v := cfg.Flags[k]
		if check, ok := validators[k]; ok {
			if err := check(v); err != nil {
				return runConfig{}, fmt.Errorf("config %s: %s = %s: %w", cfg.Path, k, v, err)
			}
		}
		if setOnCLI[k] {
			continue
		}
		if err := fs.Set(k, v); err != nil {
			return runConfig{}, fmt.Errorf("config %s: %s = %s: %w", cfg.Path, k, v, err)
		}
	}

	bgStyle, err := parseBgStyle(*bgStyleS)
	if err != nil {
		return runConfig{}, err
	}
	bgPalette, err := parseBgPalette(*bgPaletteS)
	if err != nil {
		return runConfig{}, err
	}
	bgOpts := tui.BackgroundOptions{Events: *showEvents, Style: bgStyle}
	if cfg.RandomPool != nil {
		known := tui.Backgrounds()
		pool := make([]string, len(cfg.RandomPool))
		for i, name := range cfg.RandomPool {
			name = strings.ToLower(name)
			pool[i] = name
			if !slices.Contains(known, name) {
				return runConfig{}, fmt.Errorf("config %s: random.backgrounds: unknown background %q (expected %s)", cfg.Path, name, strings.Join(known, "|"))
			}
		}
		bgOpts.RandomPool = pool
	}
	if cfg.Birthday != "" {
		md, err := events.ParseBirthday(cfg.Birthday)
		if err != nil {
			return runConfig{}, fmt.Errorf("config %s: birthday: %w", cfg.Path, err)
		}
		bgOpts.Birthday = &md
	}

	bg, err := parseBackground(*bgS)
	if err != nil {
		return runConfig{}, err
	}
	var bgTime *time.Time
	if *bgTimeS != "" {
		t, err := parseBgTime(*bgTimeS, now)
		if err != nil {
			return runConfig{}, err
		}
		bgTime = &t
	}

	state, opts, err := buildState(*layout, *date, *year, *months, *weekStart, *highlight, *clockStyle, *noColor, *withClock)
	if err != nil {
		return runConfig{}, err
	}

	return runConfig{
		state:  state,
		opts:   opts,
		print:  *print,
		width:  *width,
		bg:     bg,
		bgOpts: bgOpts,
		bgMono: !*bgColor,
		bgTime: bgTime,

		bgStyle:   bgStyle,
		bgPalette: bgPalette,
	}, nil
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

	color := !noColor && getenv("NO_COLOR") == ""

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
	case "grid", "g":
		return render.LayoutGrid, nil
	}
	return 0, fmt.Errorf("unknown --layout %q (expected horizontal|grid)", s)
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

func parseBackground(s string) (string, error) {
	s = strings.ToLower(s)
	if s == "none" {
		return "", nil
	}
	if s == tui.Random || slices.Contains(tui.Backgrounds(), s) {
		return s, nil
	}
	return "", fmt.Errorf("unknown --bg %q (expected none|random|%s)", s, strings.Join(tui.Backgrounds(), "|"))
}

func parseBgStyle(s string) (string, error) {
	if l := strings.ToLower(s); slices.Contains(tui.Styles(), l) {
		return l, nil
	}
	return "", fmt.Errorf("unknown --bg-style %q (expected %s)", s, strings.Join(tui.Styles(), "|"))
}

// parseBgPalette parses --bg-palette; auto detects true colour from
// COLORTERM.
func parseBgPalette(s string) (palette.Mode, error) {
	mode, auto, err := palette.Parse(s)
	if err != nil {
		return 0, fmt.Errorf("unknown --bg-palette %q (expected auto|truecolor|ansi)", s)
	}
	if auto {
		mode = palette.Detect(getenv)
	}
	return mode, nil
}

// parseBgTime parses --bg-time. A bare time keeps today's date; a bare date
// keeps the current time of day.
func parseBgTime(s string, now time.Time) (time.Time, error) {
	if s == "" {
		return now, nil
	}
	loc := now.Location()
	if t, err := time.ParseInLocation("2006-01-02T15:04", s, loc); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
		h, m, sec := now.Clock()
		return time.Date(t.Year(), t.Month(), t.Day(), h, m, sec, 0, loc), nil
	}
	if t, err := time.ParseInLocation("15:04", s, loc); err == nil {
		y, mo, d := now.Date()
		return time.Date(y, mo, d, t.Hour(), t.Minute(), 0, 0, loc), nil
	}
	return time.Time{}, fmt.Errorf(`invalid --bg-time %q: expected HH:MM, YYYY-MM-DD or YYYY-MM-DDTHH:MM`, s)
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
