# tcal

A live terminal calendar widget with a block-digit clock and four multi-month layouts. Also runs as a one-shot static renderer for piping or printing.

## Install

```
go install github.com/varun/tcal/cmd/tcal@latest
```

Or from source:

```
git clone <this-repo>
cd tcal
go build -o tcal ./cmd/tcal
```

## Quick start

```
tcal                          # live focus layout, block clock, centered in terminal
tcal --layout grid            # full-year wallchart
tcal --layout horizontal      # 3 months in a row
tcal --print --layout grid --date 2026-04 > year.txt
tcal --print --layout focus | lpr
```

## Keys (live mode)

| Key             | Action                          |
|-----------------|---------------------------------|
| h / ←           | Previous month                  |
| l / →           | Next month                      |
| j / ↓           | Previous year                   |
| k / ↑           | Next year                       |
| 1 / 2 / 3 / 4   | Layout: horizontal / vertical / grid / focus |
| t               | Jump to today                   |
| q / Ctrl-C      | Quit                            |

## Flags

| Flag             | Default        | Notes                                      |
|------------------|----------------|--------------------------------------------|
| `--layout`       | `focus`        | `horizontal \| vertical \| grid \| focus`  |
| `--date`         | today          | `YYYY-MM` or `YYYY-MM-DD`                  |
| `--year`         | (anchor year)  | shorthand for `--date=YYYY-01`             |
| `--months`       | layout default | per-layout: 3 / 3 / 12 / 3                 |
| `--week-start`   | `sun`          | `sun \| mon`                               |
| `--highlight`    | `combined`     | `combined \| reverse \| bracket \| color \| none` |
| `--no-color`     | off            | also honors `NO_COLOR`                     |
| `--print`        | off            | one-shot render to stdout                  |
| `--width`        | TTY width / 80 | print mode only                            |
| `--with-clock`   | off            | include block clock in print mode          |
| `--clock-style`  | `block`        | `block \| inline \| boxed`                 |

## Manual smoke checklist

After any change to layout/centering code, run through:

- [ ] `tcal` — focus layout, clock ticks, terminal resize re-centers.
- [ ] `tcal --layout grid` — 12-month wallchart fits in a standard terminal.
- [ ] `tcal --print --layout grid --date 2026-01 > /tmp/y.txt && cat /tmp/y.txt` — clean static output.
- [ ] `NO_COLOR=1 tcal` — combined highlight degrades to bracket-only; no ANSI escapes.
- [ ] Run in a 40×10 terminal — "too narrow" hint appears at the top.
- [ ] `tcal --date 2026-13` — exits non-zero with a one-line error.

## License

MIT
