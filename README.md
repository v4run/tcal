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
| [ / ]           | Background behind the calendar: less / more (see `--bg-behind`) |
| q / Ctrl-C      | Quit                            |

## Flags

| Flag             | Default        | Notes                                      |
|------------------|----------------|--------------------------------------------|
| `--layout`       | `horizontal`   | `horizontal \| grid`; see [Grid layout](#grid-layout) |
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
| `--look`         | `classic`      | `classic \| thin`; see [Thin look](#thin-look) |
| `--small-neighbors` | `true`      | thin look: when three thin months don't fit, show this month between small neighbours; `false` shows it alone |
| `--bg`           | `none`         | `none`, `random`, or a name from [Backgrounds](#backgrounds) (live mode only) |
| `--bg-behind`    | `0`            | background opacity behind the calendar and clock, `0`–`100`; `0` keeps it clear. Every step shows in true colour; other palettes are only off or faint |
| `--bg-color`     | `true`         | tint the background; `false` draws it dim in the foreground color |
| `--bg-time`      | now            | time seen by `sky`/`hourglass` and by [events](#events): `HH:MM`, `YYYY-MM-DD` or `YYYY-MM-DDTHH:MM` |
| `--bg-style`     | `classic`      | `classic \| smooth`; see [Smooth style](#smooth-style) |
| `--bg-palette`   | `auto`         | `auto \| truecolor \| ansi`; colors used by the smooth style |
| `--events`       | `true`         | show holiday scenes in the `seasons` background; `--events=false` turns them off |
| `--config`       | see [Config file](#config-file) | read this config file instead of the default; an error if it is missing |
| `--init-config`  | `false`        | write a config file with every default to `--config` or the default location, then exit; never overwrites |

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
| `coast`     | waves rolling in to burst into spray on the rocks and wash up a sandy beach |
| `trees`     | a tree in each margin that follows the season of the month on screen, falling leaves, a bench and lamp post below the calendar, and a bird that flits between them |
| `hourglass` | sand piling up along the bottom over the hour |
| `rain`      | a light drizzle with splashes on the ground |
| `starfield` | slow parallax stars and the odd shooting star |
| `aquarium`  | fish, rising bubbles, and swaying seaweed |
| `lava`      | lava-lamp blobs merging and splitting |
| `flow`      | particles drifting along a changing current |
| `langton`   | Langton's ant |
| `automaton` | fractal cellular automata scrolling upward |

Bottom-strip scenes (`container`, `harbor`, `train`, `coast`) hide themselves when the terminal is too short to fit them below the calendar. `trees` fits a tree into each margin at least 12 columns wide; the bench and lamp appear when there is room below the calendar.

Backgrounds are drawn dim and tinted from your terminal's 16-color palette, so they follow your theme; `--bg-color=false` keeps them monochrome. To preview a time-aware background at another time, shift its clock with `--bg-time` (it keeps ticking from there; the calendar keeps the real time):

```
tcal --bg sky --bg-time 22:30               # tonight's sky
tcal --bg sky --bg-time 2024-04-23T23:00    # a full moon
tcal --bg hourglass --bg-time 10:55         # nearly full hourglass
```

`seasons` follows the month on screen — page with `h`/`l` to change the season.

## Grid layout

`--layout grid` (or `2` while running) shows the whole year. It picks the widest arrangement that fits the screen, from 6×2 through 4×3 and 3×4 to 2×6 months, so a wide terminal leaves room below the calendar for backgrounds. When none fits under the block or boxed clock, the clock shrinks to one line; when even that doesn't fit, it keeps 3×4. Print mode fits the arrangement to `--width` (or the terminal), and keeps 3×4 when the width is unknown.

The year view is an overview, so it always uses the classic look.

## Thin look

`--look thin` draws the clock and the day numbers three rows tall in thin lines. A bar under the time fills across each minute. Past days are dim, weekends grey, and today is a solid block in the accent colour (bracketed when color is off).

```
tcal --look thin --months 1
```

A thin month is 49 columns wide and 22 rows tall. When the calendar doesn't fit the screen (or `--width` in print mode), tcal steps down, dropping the least important thing first:

| Step | Shown | Needs about |
|------|-------|-------------|
| 1 | all the months in thin digits | 151 columns for three |
| 2 | this month in thin digits, the months either side small and dim (`--small-neighbors=false` skips this) | 95 columns |
| 3 | this month alone in thin digits | 50 columns |
| 4 | the classic look, then fewer months (5 → 3 → 1) | 64, then 20 columns |

Before each step down, a tall clock shrinks to one line, so one thin month fits an 80×24 terminal. The months you can't see are always a keypress away with `h`/`l`. The grid layout always uses the classic look. `--clock-style` is ignored in the thin look. Set `look = "thin"` in the [config file](#config-file) to make it your default.

## Smooth style

`--bg-style smooth` redraws every background with braille dots, at about 30 frames per second, and in true colour where the terminal supports it. It looks finer and moves more fluidly than the default. `life`, `langton`, `automaton` and `ants` are cell grids, so they look the same in both styles.

```
tcal --bg-style smooth --bg rain
```

`classic` is still the default, and will be retired in a later release. To make smooth your default, set it in the [config file](#config-file):

```toml
bg-style = "smooth"
```

**True colour.** With `--bg-palette auto` (the default), smooth uses true colour when `COLORTERM` is `truecolor` or `24bit`, and otherwise falls back to the 16-color palette like classic. Some terminals support true colour without setting `COLORTERM`, and tmux often hides it. To force it, use `--bg-palette truecolor` or `bg-palette = "truecolor"` in the config; `--bg-palette ansi` forces the 16 colors. `NO_COLOR` and `--no-color` win over `--bg-palette`: with no color there is no tint at all.

**Synchronized output.** In smooth style, tcal wraps each screen update in the synchronized-output sequence (mode 2026) so that terminals which support it draw the frame at once without tearing. Bubble Tea does not send this itself, so tcal does. Terminals that do not support it ignore it. Classic style does not send it.

## Events

On special days the `seasons` background adds a small, dim holiday scene beside the calendar; the season's particles keep running. Only one event shows at a time.

| Event | Shown | Scene |
|-------|-------|-------|
| New Year | Dec 31 from 18:00 through Jan 1 | fireworks in the top margin, busiest around midnight |
| Makar Sankranti / Pongal | Makar Sankranti and the day after | kites with tails and a flyer, left column |
| Lunar New Year | the day and the 2 days after | red lanterns swaying along the top edge |
| Valentine's Day | Feb 14 | ASCII hearts rising up the right column |
| Holi | the evening before (from 18:00) and the day of colours | bursts of colored powder, right column |
| Vishu | the day | a lit nilavilakku lamp with falling konna flowers, bottom-right |
| Eid al-Fitr | the day, plus or minus 1 day | a crescent moon and hanging lanterns, top-right |
| Independence Day (India) | Aug 15 | a tricolor flag on a pole, top-left |
| Onam | Atham through Thiruvonam | a pookalam in the bottom-left corner that gains a ring each day and is complete on Thiruvonam |
| Halloween | Oct 31 | a jack-o'-lantern and bats, bottom-left column |
| Diwali | Lakshmi Puja day, plus or minus 2 days | a row of flickering diyas along the bottom, with small bursts above |
| Christmas | Dec 1 through Dec 26 | a tree with twinkling lights, bottom-right corner |
| Birthday | the `birthday` in your [config](#config-file) | a cake with candles and confetti, bottom-right |

Each scene sits in one side column, corner or edge band beside the calendar and hides itself entirely when it would not fit without touching the calendar, which typically happens below about 100 columns wide.

Events follow the real date, while seasons follow the month on screen. To preview one, shift the date with `--bg-time`:

```
tcal --bg seasons --bg-time 2026-12-25      # Christmas
tcal --bg seasons --bg-time 2026-10-31      # Halloween
tcal --bg seasons --bg-time 2026-12-31T23:30   # New Year, near midnight
```

If two events overlap, your birthday wins; otherwise the one with the shorter window does. With `--bg random`, a day with an active event picks `seasons` (if it is in the random pool) so the event is seen. Some scenes need room beside the calendar (roughly 100+ columns), so on a narrow terminal the background can be `seasons` with no visible scene. To avoid that, remove `seasons` from `[random] backgrounds`, or set `events = false`. Turn events off with `--events=false` or `events = false` in the config; `seasons` then behaves as it always did.

Fixed-date events are computed. Moving dates (Sankranti, Lunar New Year, Holi, Vishu, Eid, Onam, Diwali) come from a lookup table of published calendars covering 2026-2040, with a few documented gaps: Vishu 2032-2040, Onam 2031-2040, Eid 2037 and Diwali 2037. In a gap year that event does not show; a missing date is preferred to a wrong one. Sources are cited in `internal/events/dates.go`.

## Config file

An optional TOML file sets defaults for the flags, your birthday and the pool `--bg random` picks from. It lives at `$XDG_CONFIG_HOME/tcal/config.toml`, falling back to `~/.config/tcal/config.toml` on every platform (macOS included). `--config <path>` reads a different file. With no file, tcal behaves exactly as it does without this feature.

To start one, let tcal write it: every key set to its built-in default with a comment listing the choices, and the birthday and random pool commented out. It refuses to replace an existing file.

```
tcal --init-config                      # writes ~/.config/tcal/config.toml
tcal --init-config --config ./tcal.toml # or anywhere else
```

```toml
# ~/.config/tcal/config.toml — every key is optional

# Defaults for command-line flags (flags still win)
layout      = "grid"        # horizontal | grid
week-start  = "mon"         # sun | mon
highlight   = "combined"    # combined | reverse | bracket | color | none
clock-style = "block"       # block | inline | boxed
look        = "classic"     # classic | thin
small-neighbors = true      # thin look: small months either side when three big ones don't fit
with-clock  = true
no-color    = false
bg          = "random"      # none | random | <background name>
bg-color    = true
bg-behind   = 0             # 0-100, background opacity behind the calendar
bg-style    = "classic"    # classic | smooth
bg-palette  = "auto"       # auto | truecolor | ansi
events      = true

# Personal
birthday = "05-17"          # MM-DD (YYYY-MM-DD also accepted; the year is ignored)

[random]
# Backgrounds --bg random may pick. Omit to allow all of them.
backgrounds = ["sky", "seasons", "aquarium", "harbor", "train", "life"]
```

`months` and `width` can be set too. Precedence is flag on the command line, then config value, then built-in default.

Config values go through the same validation as the flags. A bad value, an unknown key, an unknown name in `random.backgrounds`, an empty list or a missing `--config` file is an error naming the file, and tcal exits non-zero:

```
$ tcal --config bad.toml          # layout = "diagonal"
tcal: config bad.toml: layout = diagonal: unknown --layout "diagonal" (expected horizontal|grid)
```

These flags cannot be set in the config: `--date`, `--year`, `--print`, `--bg-time`, `--version`, `--config` and `--init-config`. A missing file at the default location is not an error.

## Manual smoke checklist

After any change to layout/centering code, run through:

- [ ] `tcal` — horizontal layout, clock ticks, terminal resize re-centers.
- [ ] `tcal --layout grid` — the year fits: 4×3 at 120×36, 6×2 from about 132 columns; resize to watch it rearrange.
- [ ] `tcal --print --layout grid --date 2026-01 > /tmp/y.txt && cat /tmp/y.txt` — clean static output.
- [ ] `NO_COLOR=1 tcal` — no color escapes in output; today still distinguishable via reverse video.
- [ ] Narrow the terminal below 64 columns — three months become one; below 20 the "too narrow" hint appears.
- [ ] `tcal --bg life` — dim cells animate in the margins only, never inside the calendar/clock box; resize reseeds.
- [ ] `NO_COLOR=1 tcal --bg life` — cells render as plain `·` with no escapes.
- [ ] `tcal --bg aquarium` vs `tcal --bg aquarium --bg-color=false` — tinted vs plain dim; `NO_COLOR=1` shows no escapes.
- [ ] `tcal --bg sky --bg-time 22:30` — night sky and moon, while the calendar header shows the real time.
- [ ] `tcal --bg-style smooth --bg rain` — braille drizzle, smooth motion.
- [ ] `tcal --look thin --months 1` — thin clock with a seconds bar over this month in big day numbers; shrink the terminal and it falls back to classic.
- [ ] `tcal --look thin` at 120 columns — thin October between small, dim September and November; widen past 151 for three thin months, narrow below 95 for October alone.
- [ ] `NO_COLOR=1 tcal --look thin --months 1` — today in corner brackets, no escapes.
- [ ] `tcal --look thin --months 1 --bg-style smooth --bg lava --bg-behind 30` — lava faint behind the dates, today's block clean; `[`/`]` fade it out and in.
- [ ] `COLORTERM= tcal --bg-style smooth --bg sky` — ANSI fallback: 16-color tinting only, still recognisable.
- [ ] `COLORTERM= tcal --bg-style smooth --bg sky --bg-palette truecolor` — full true colour even with `COLORTERM` unset.
- [ ] `tcal --bg-style smooth --bg seasons --bg-time 2026-12-25` — a smooth Christmas scene; `--events=false` removes it.
- [ ] `tcal --bg random` a few times — a different background each launch; none ever draw over the calendar.
- [ ] `tcal --bg container` in a tall terminal — a shipping container slides along the bottom and hops every bush; shrink the window until the strip would touch the calendar and it disappears.
- [ ] `tcal --bg seasons --bg-time 2026-12-25` — a Christmas tree with twinkling lights in a corner; `--events=false` removes it.
- [ ] A config with `bg = "random"` and `[random] backgrounds = ["sky", "life"]` — only those two ever appear; a flag such as `--bg none` overrides the config.
- [ ] A config with `layout = "diagonal"` — exits non-zero with a one-line error naming the file, key and value.
- [ ] `tcal --date 2026-13` — exits non-zero with a one-line error.

## License

MIT
