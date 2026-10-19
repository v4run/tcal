# tcal

A live terminal calendar widget with a block-digit clock and three multi-month layouts. Also runs as a one-shot static renderer for piping or printing.

## Install

```
go install github.com/v4run/tcal/cmd/tcal@latest
```

Or from source:

```
git clone <this-repo>
cd tcal
go build -o tcal ./cmd/tcal
```

## Quick start

```
tcal                          # live horizontal layout, block clock, centered in terminal
tcal --layout grid            # full-year wallchart
tcal --print --layout grid --date 2026-04 > year.txt
tcal --print | lpr
```

## Keys (live mode)

| Key             | Action                          |
|-----------------|---------------------------------|
| h / ←           | Previous month                  |
| l / →           | Next month                      |
| j / ↓           | Previous year                   |
| k / ↑           | Next year                       |
| 1 / 2           | Layout: horizontal / grid                    |
| t               | Jump to today                   |
| q / Ctrl-C      | Quit                            |

## Flags

| Flag             | Default        | Notes                                      |
|------------------|----------------|--------------------------------------------|
| `--layout`       | `horizontal`   | `horizontal \| grid`                       |
| `--date`         | today          | `YYYY-MM` or `YYYY-MM-DD`                  |
| `--year`         | (anchor year)  | shorthand for `--date=YYYY-01`             |
| `--months`       | layout default | per-layout: 3 / 12                         |
| `--week-start`   | `sun`          | `sun \| mon`                               |
| `--highlight`    | `combined`     | `combined \| reverse \| bracket \| color \| none` |
| `--no-color`     | off            | also honors `NO_COLOR`                     |
| `--print`        | off            | one-shot render to stdout                  |
| `--width`        | TTY width / 80 | print mode only                            |
| `--with-clock`   | off            | include block clock in print mode          |
| `--clock-style`  | `block`        | `block \| inline \| boxed`                 |
| `--bg`           | `none`         | `none`, `random`, or a name from [Backgrounds](#backgrounds) (live mode only) |
| `--bg-color`     | `true`         | tint the background; `false` draws it dim in the foreground color |
| `--bg-time`      | now            | time seen by `sky`/`hourglass`: `HH:MM`, `YYYY-MM-DD` or `YYYY-MM-DDTHH:MM` |

## Backgrounds

`--bg <name>` draws a dim ambient animation in the empty space around the calendar; the calendar itself is never drawn over. `--bg random` picks one at startup.

| Name        | What you see |
|-------------|--------------|
| `life`      | Conway's Game of Life, reseeding when it settles |
| `container` | a shipping container sliding along the bottom, hopping bushes |
| `harbor`    | container ships gliding across a strip of water |
| `train`     | a freight train passing every few minutes |
| `ants`      | an ant trail marching around the calendar |
| `sky`       | the sun crossing the sky by day; stars and the real moon phase by night |
| `seasons`   | snow, petals, fireflies, or leaves — by the month on screen |
| `hourglass` | sand piling up along the bottom over the hour |
| `rain`      | a light drizzle with splashes on the ground |
| `starfield` | slow parallax stars and the odd shooting star |
| `aquarium`  | fish, rising bubbles, and swaying seaweed |
| `lava`      | lava-lamp blobs merging and splitting |
| `flow`      | particles drifting along a changing current |
| `langton`   | Langton's ant |
| `automaton` | fractal cellular automata scrolling upward |

Bottom-strip scenes (`container`, `harbor`, `train`) hide themselves when the terminal is too short to fit them below the calendar.

Backgrounds are drawn dim and tinted from your terminal's 16-color palette, so they follow your theme; `--bg-color=false` keeps them monochrome. To preview a time-aware background at another time, shift its clock with `--bg-time` (it keeps ticking from there; the calendar keeps the real time):

```
tcal --bg sky --bg-time 22:30               # tonight's sky
tcal --bg sky --bg-time 2024-04-23T23:00    # a full moon
tcal --bg hourglass --bg-time 10:55         # nearly full hourglass
```

`seasons` follows the month on screen — page with `h`/`l` to change the season.

## Manual smoke checklist

After any change to layout/centering code, run through:

- [ ] `tcal` — horizontal layout, clock ticks, terminal resize re-centers.
- [ ] `tcal --layout grid` — 12-month wallchart fits in a standard terminal.
- [ ] `tcal --print --layout grid --date 2026-01 > /tmp/y.txt && cat /tmp/y.txt` — clean static output.
- [ ] `NO_COLOR=1 tcal` — no color escapes in output; today still distinguishable via reverse video.
- [ ] Run in a 40×10 terminal — "too narrow" hint appears at the top.
- [ ] `tcal --bg life` — dim cells animate in the margins only, never inside the calendar/clock box; resize reseeds.
- [ ] `NO_COLOR=1 tcal --bg life` — cells render as plain `·` with no escapes.
- [ ] `tcal --bg aquarium` vs `tcal --bg aquarium --bg-color=false` — tinted vs plain dim; `NO_COLOR=1` shows no escapes.
- [ ] `tcal --bg sky --bg-time 22:30` — night sky and moon, while the calendar header shows the real time.
- [ ] `tcal --bg random` a few times — a different background each launch; none ever draw over the calendar.
- [ ] `tcal --bg container` in a tall terminal — a shipping container slides along the bottom and hops every bush; shrink the window until the strip would touch the calendar and it disappears.
- [ ] `tcal --date 2026-13` — exits non-zero with a one-line error.

## License

MIT
