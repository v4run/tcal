# tcal — Design Spec

Date: 2026-05-27
Status: Draft (pending review)

## Overview

`tcal` is a terminal calendar widget written in Go. By default it renders a live, center-aligned calendar with a large block-digit clock in the terminal, ticking every second. The same binary supports a one-shot `--print` mode that emits a static rendered calendar to stdout (suitable for piping to `lpr`, redirecting to a file, or embedding in scripts).

The calendar can be displayed in four arrangements — horizontal strip, vertical stack, year-wallchart grid, and focus (large current month with smaller neighbors) — and the user switches between them at runtime via flag or keypress.

## Goals

- Live always-on widget suitable for a tmux pane or dedicated terminal.
- Four interchangeable multi-month layouts, all sharing one rendering pipeline.
- Print mode that uses the same renderer and produces clean stdout output.
- Content centered horizontally and vertically in the terminal viewport.
- Single static binary, easy to install and run.
- Renderer is pure and golden-file-testable in isolation.

## Non-goals

- Events, reminders, tasks, or any per-day annotations.
- Time-zone handling beyond the system's local time.
- Configurable themes or color palettes (beyond `--no-color`).
- Mouse support.
- Persistent configuration files.
- Internationalized month/day names (English only for v1).

## Architecture

### Package layout

```
tcal/
├── cmd/tcal/main.go          entry — parses flags, dispatches to tui or printout
├── internal/
│   ├── calendar/             pure date logic; no rendering
│   │   ├── month.go          build a Month value (weeks of days, today flag)
│   │   └── week.go           week-start helpers, day-of-year, ISO week#
│   ├── render/               pure rendering; no TUI, no I/O
│   │   ├── month.go          render one Month block to a string
│   │   ├── layouts.go        4 layouts: horizontal / vertical / grid / focus
│   │   ├── clock.go          block-digit ASCII glyphs (0–9, colon, space)
│   │   ├── highlight.go      today decoration (reverse + bracket + accent color)
│   │   └── center.go         H+V center a block within (width, height)
│   ├── tui/                  Bubble Tea wrapper (live mode)
│   │   └── model.go          model/update/view; tick every 1s; keybindings
│   └── printout/             print mode (one-shot, no TUI)
│       └── print.go          render once with --width fallback, print to stdout
├── go.mod
└── README.md
```

### Boundaries

- `calendar` knows dates; nothing about strings or terminals.
- `render` knows strings and styling; takes data from `calendar`; nothing about Bubble Tea or stdout.
- `tui` and `printout` are thin orchestrators — pull state, call `render`, hand the result to either Bubble Tea's `View()` or `fmt.Print`.

The renderer is the heart of the app and is fully unit-testable with no TUI dependencies. Print mode and live mode share 100% of layout/centering code.

### External dependencies

- `github.com/charmbracelet/bubbletea` — event loop, tick scheduling.
- `github.com/charmbracelet/lipgloss` — ANSI styling, width measurement, color handling (honors `NO_COLOR` automatically).
- `golang.org/x/term` — terminal size detection in print mode.

## Render pipeline

Both `tui` and `printout` invoke a single top-level function:

```go
render.Frame(state State, opts Options) string

type State struct {
    Anchor time.Time  // first day of focus month
    Today  time.Time
    Now    time.Time  // for clock
    Layout Layout     // Horizontal | Vertical | Grid | Focus
    Months int        // count; 0 = layout default
    Width  int        // terminal cols
    Height int        // terminal rows
}

type Options struct {
    WeekStart  time.Weekday   // Sunday by default
    Color      bool           // false disables ANSI
    Highlight  HighlightMask  // bits: Reverse | Bracket | Color
    ClockStyle ClockStyle     // Block by default
    WithClock  bool           // print mode only; default false
}
```

### Stages (pure, left-to-right)

```
calendar.BuildMonths(anchor, count, weekStart) → []calendar.Month
   // each Month is [][]Day; each Day knows isToday, isInMonth

render.month.Build(Month, opts) → string
   // header + weekday row + day grid; today decorated via render.highlight

render.layouts.<layout>(monthBlocks, opts) → string
   // horizontal: lipgloss.JoinHorizontal with gutter
   // vertical:   lipgloss.JoinVertical with blank-line gutter
   // grid:       horizontal join within rows, vertical join across rows
   // focus:      big center month + small neighbors flanking it

render.clock.Render(now, ClockStyle) → string
   // block-digit ASCII; 5 rows tall, fixed col width per glyph

compose(clockBlock, layoutBlock) → string
   // clock centered above the layout block, blank-line gutter, then layout
   // uniform across all four layouts — keeps the pipeline simple and
   // accommodates the wide (~50-col) block-digit clock

render.center.Fit(block, width, height) → string
   // pads spaces L+R for horizontal centering
   // pads blank lines T+B for vertical centering
   // graceful degradation if content > viewport (see Edge cases)
```

All composition uses `lipgloss.JoinHorizontal` / `JoinVertical`, which correctly account for ANSI escape sequences when measuring visible width.

### Today highlight

The combined highlight (default) renders today with reverse video AND an accent-color foreground — no brackets, preserving grid alignment. The `[22]` bracketed style is available via `--highlight=bracket` for users who prefer it (note: produces rows 1 char wider for 2-digit days). The `--highlight` flag exposes `combined | reverse | bracket | color | none` for users who want a lighter touch.

### Centering edge cases

- **Block wider than terminal:** fall back to left-align; print a one-line hint at the top: `terminal too narrow — needs ≥ N cols`. Continue ticking; re-evaluate on resize.
- **Block taller than terminal:** top-align; allow clipping at the bottom. Same hint pattern.
- **Print mode:** vertical centering is disabled (no terminal height; padding would be junk in a piped file). Horizontal centering still applies, using `--width` flag or detected width, default 80.

### Clock tick & caching

Live mode emits a `tea.Tick` every 1 second. On each tick, the model updates `Now` and re-renders. The month-grid block is memoized and only rebuilt when `Anchor`, `Today`, `Layout`, or terminal size changes. Clock-only ticks reuse the cached grid and only re-render the clock region. This keeps the per-tick work to ~50 lines of string composition.

## Live mode (Bubble Tea)

```go
type Model struct {
    state render.State
    opts  render.Options
    cache layoutCache  // memoized month/layout blocks
}

Init():    schedule first 1-second tick
Update(msg):
  tea.WindowSizeMsg → update Width/Height; invalidate cache
  tickMsg           → update Now; keep cache; schedule next tick
  tea.KeyMsg        → handle key (see table); possibly mutate state & invalidate
View():    return render.Frame(state, opts)
```

### Keybindings

| Key             | Action                                  |
|-----------------|-----------------------------------------|
| `h` / `←`       | Anchor − 1 month                        |
| `l` / `→`       | Anchor + 1 month                        |
| `j` / `↓`       | Anchor − 1 year                         |
| `k` / `↑`       | Anchor + 1 year                         |
| `1` … `4`       | Layout: horizontal / vertical / grid / focus |
| `t`             | Jump anchor to today's month            |
| `?`             | Toggle help overlay                     |
| `q` / `Ctrl-C`  | Quit                                    |

Note: `j`/`k` map to year-down / year-up rather than strict vim semantics, since "earlier in time = down" matches the way users think about navigating calendars. Revisable.

## Print mode

```
tcal --print [flags]
```

Behavior:
1. Parse flags, build `State` and `Options`; the `tui` package is not imported.
2. Resolve width: explicit `--width` → detected TTY width via `golang.org/x/term` → fallback 80.
3. Call `render.Frame(state, opts)` once.
4. `fmt.Print` to stdout; exit 0.

The block-digit clock is omitted by default in print mode (a 5-line ASCII clock isn't useful in a static file). Instead, a single-line header `Wed 22 Apr 2026 14:32` is printed above the calendar. `--with-clock` re-enables the block clock if requested.

## CLI surface

| Flag             | Default              | Notes                                      |
|------------------|----------------------|--------------------------------------------|
| `--layout`       | `focus`              | `horizontal \| vertical \| grid \| focus`  |
| `--date`         | today                | `YYYY-MM` or `YYYY-MM-DD`                  |
| `--year`         | (anchor year)        | shorthand for `--date=YYYY-01`             |
| `--months`       | layout default       | per-layout: 3 / 3 / 12 / 3                 |
| `--week-start`   | `sun`                | `sun \| mon`                               |
| `--highlight`    | `combined`           | `combined \| reverse \| bracket \| color \| none` |
| `--no-color`     | off                  | also honors `NO_COLOR` env var             |
| `--print`        | off                  | switch to one-shot mode                    |
| `--width`        | TTY width / 80       | print mode only                            |
| `--with-clock`   | off                  | include block clock in print mode          |
| `--clock-style`  | `block`              | `block \| inline \| boxed`                 |
| `--help`         |                      | usage; exits 0                             |
| `--version`      |                      | version; exits 0                           |

### Default per-layout month counts

| Layout      | Default months | Arrangement                                    |
|-------------|----------------|------------------------------------------------|
| horizontal  | 3              | prev / current / next, side-by-side            |
| vertical    | 3              | prev / current / next, stacked                 |
| grid        | 12             | full year, 3×4 (or 4×3 if terminal is wide)    |
| focus       | 3              | current month large, prev + next small on sides |

All overridable via `--months=N`.

## Error handling

Three categories, handled at their natural boundary:

1. **CLI parse errors** (bad flag, unparseable `--date`, unknown layout): exit code 2 with a one-line stderr message, e.g. `tcal: invalid --date "2026-13": month out of range`. No stack traces.
2. **Terminal too small** (live mode): don't crash. Print the hint line at the top of the viewport, keep ticking, redraw on resize.
3. **Print mode write failures:** swallow `EPIPE` silently and exit 0 (so `tcal --print | head` is well-behaved). Other write errors → stderr + exit 1.

No retries, no fallback layouts, no panic recovery. The renderer is pure and shouldn't return errors — invariant violations are programming bugs, asserted via `panic` and surfaced through tests.

## Testing strategy

The renderer is the high-leverage test target.

- **`internal/calendar`** — table-driven tests for month-grid construction across leap years, year boundaries, both week-start modes (~30 cases).
- **`internal/render/month`** — golden-file tests: render fixed months at fixed dates, compare byte-for-byte against checked-in `.golden` files. One golden per (week-start × today-position-in-grid) combination.
- **`internal/render/layouts`** — golden-file tests per layout at multiple widths (narrow / medium / wide). Verifies horizontal joining, vertical stacking, grid wrap-points, focus composition.
- **`internal/render/clock`** — golden-file tests for each digit 0–9 and a few full times.
- **`internal/render/center`** — unit tests for padding math at various `(content, viewport)` combinations including too-small cases.
- **`internal/tui`** — minimal: assert keybindings mutate state correctly by calling `Update` directly with synthetic `tea.KeyMsg`. Re-rendering is already exercised by the renderer tests.
- **`internal/printout`** — one integration test: invoke with `--print --layout grid --date 2026-04` against an in-memory buffer, diff against golden.

Goldens are updatable via a `-update` flag wired into the test binary (`go test ./... -update`). Visual changes show up as a single reviewable diff per PR.

### Manual smoke checklist (in README)

- Run with each `--layout`; verify centering survives terminal resize.
- Run `tcal --print --layout grid --date 2026-01 > /tmp/y.txt`; visually inspect.
- Run `NO_COLOR=1 tcal`; verify highlight degrades to bracket-only.
- Run in a 40×10 terminal; verify "too narrow" hint appears.

## Open questions / known revisitable choices

- **Vim `j`/`k` semantics:** Currently `j`=year-down, `k`=year-up (calendar-natural — "earlier" feels like "down"). Strict vim users may expect the opposite; can be flipped on feedback.
- **`--clock-style` in live mode:** Defaults to `block`. Inline and boxed variants are wired in but not the default; verify they read well at common terminal sizes during implementation.
