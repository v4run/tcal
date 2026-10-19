# Holiday events and config file

**Date:** 2026-10-19
**Status:** Draft

## Summary

Two related additions to tcal's live-mode backgrounds:

1. **Events**: on special days, the `seasons` background adds a small, festive scene for the occasion (a Christmas tree, Diwali lamps, an Onam pookalam…). Thirteen events, including the user's birthday.
2. **Config file**: a file that sets defaults for the command-line flags, holds the user's birthday, and chooses which backgrounds `--bg random` picks from.

## Why

tcal now has fifteen ambient backgrounds (see the README *Backgrounds* section). They are pleasant but unchanging. The user wants them to feel *eventful*: small surprises on days that matter, which is what makes a calendar feel like theirs instead of a screensaver. tcal is a calendar, so it has a natural reason to know what day it is.

The config file follows from the birthday (it is personal and shouldn't need to be typed as a flag each launch). While we're there, the user wants it to hold defaults for the existing flags and to control the `random` pool, since not every background suits every taste.

## Goals

- Thirteen events, each with its own small scene, shown on the right days.
- Dates that move each year (Onam, Diwali, Holi, Vishu, Sankranti, Eid, Lunar New Year) come from a lookup table that is **correct**: sourced from published calendars, not from memory.
- A config file with defaults for flags, the birthday, and the random pool, fully optional.

## Non-goals

- Events on backgrounds other than `seasons` (except via the random rule below). A Christmas tree in the aquarium would look odd.
- User-defined custom events beyond the birthday.
- Regional variants of dates (e.g. Eid differing by country); the window absorbs the usual ±1 day instead.
- Southern-hemisphere seasons.
- Print mode: events and config-driven backgrounds are live-mode only. Config defaults for print-mode flags (`--width`, `--with-clock`) still apply.

## Anti-goals (would make this a failure even if it technically works)

- An event that draws over the calendar or makes it harder to read. All existing rules hold: dim, quiet, never inside the calendar box, single-width runes only.
- A wrong date for a moving holiday. A missing date is acceptable; a wrong one is not.
- A config file that's required, or whose absence or errors change today's behaviour silently. No file means behaviour stays exactly as it is now.

## Events

### The list

| Event | Window (inclusive, local time) | Scene idea |
|---|---|---|
| New Year | Dec 31 18:00 → Jan 1 23:59 | fireworks bursting in the top margin, densest around midnight |
| Makar Sankranti / Pongal | table date (Makar Sankranti) and the day after, which covers Thai Pongal when it falls a day later | kites bobbing and drifting in the sky, line art with tails |
| Lunar New Year | table date and the 2 days after | red lanterns swaying along the top margin |
| Valentine's Day | Feb 14 | a few small ASCII hearts (`<3`-style or block art, **not** `♥`) floating up |
| Holi | table date = the day of colours (Rangwali Holi / Dhulandi), plus the evening before (Holika Dahan) from 18:00 | soft bursts of colored powder, several colors |
| Vishu | table date | golden konna (laburnum) flowers drifting down, a small nilavilakku lamp |
| Eid al-Fitr | table date ±1 day | a crescent moon and hanging lanterns |
| Independence Day (India) | Aug 15 | a small tricolor flag fluttering, or tricolor kites |
| Onam | Atham → Thiruvonam, both from the table (usually 10 days, sometimes 9 or 11) | a pookalam (round flower carpet) in a corner, growing day by day so it is complete on Thiruvonam, scaled to however many days that year has |
| Halloween | Oct 31 | a jack-o'-lantern and a bat or two flitting about |
| Diwali | main day (Lakshmi Puja) ±2 days | a row of flickering diyas along the bottom, occasional small fireworks |
| Christmas | Dec 1 → Dec 26 | a small tree with twinkling lights in a bottom corner (snow keeps falling from the winter season) |
| Birthday | the configured month-day | a cake with candles, or gentle confetti |

The visuals are sketches; the builder designs the art within the constraints below and shows the user previews before finalizing.

### When an event shows

- Events follow **the real date** (today), unlike seasons, which follow the month on screen. An event is about *today*. `--bg-time` shifts the date events see, so any event can be previewed (e.g. `tcal --bg seasons --bg-time 2026-12-25`).
- The event layer is part of the `seasons` background: the season's particles keep running, and the event's scene is drawn alongside them.
- **Random on event days** *(Claude's call)*: when `--bg random` would run on a day with an active event, it picks `seasons` so the event is actually seen, provided `seasons` is in the random pool.
- **Overlaps**: only one event shows at a time. Priority: birthday first; otherwise the event with the shorter window wins (a single-day event beats Christmas's month-long tree); ties go to the earlier row in the table.
- **Turning events off**: `--events=false`, or `events = false` in the config. Seasons then behave exactly as today.

### Constraints on event scenes

- All existing scene rules: dim, single-width runes only (avoid `♥ ★ ❄ •` and other East-Asian-ambiguous symbols), implement `scene.Colored` with a small, restrained palette, cheap `Glyph`/`Color` lookups, safe at any terminal size.
- Event scenes take a bounded area: a corner, the top margin, or the bottom strip. Like the bottom-strip backgrounds, an event scene that can't fit in the margins without touching the calendar's box hides itself entirely rather than being clipped.
- An event scene must be recognizable at a glance, at small size, while drawn dim.

### The dates table

- Moving dates: Sankranti/Pongal (Makar Sankranti day), Lunar New Year, Holi (day of colours), Vishu, Eid al-Fitr, Onam (**both** Atham and Thiruvonam: the festival is not always 10 days, because a nakshatra can be skipped or repeated), Diwali (Lakshmi Puja day).
- **Sourcing** *(Claude's call)*: the builder researches the dates with web search/fetch from the sources below and records each source. No date is written from memory.
- Coverage target: **2026 through 2040**. Gaps are allowed where no reliable published date exists (e.g. far-future Eid, years where sources disagree and none is clearly dominant); a gap is documented in a comment with the reason. **2026–2030 must be complete** for every event.
- Each date must come from published calendars (e.g. drikpanchang.com for Hindu festivals, timeanddate.com / official announcements for Eid and Lunar New Year), and the source must be cited in a comment next to the table. Where sources disagree (common for Eid, Holi, Vishu), pick the date most widely observed in India for the Indian festivals, use the astronomical/expected date for Eid, and note the disagreement in a comment.
- A year missing from the table means that event doesn't show that year. No fallback guessing.
- Fixed dates (Valentine's, Aug 15, Oct 31, Dec 25, Jan 1) are computed, not tabled.

## Config file

### Location

`$XDG_CONFIG_HOME/tcal/config.toml`, falling back to `~/.config/tcal/config.toml` on all platforms, including macOS *(Claude's call: CLI users expect `~/.config`; Go's `os.UserConfigDir` would pick `~/Library/Application Support` on macOS)*. A `--config <path>` flag points elsewhere. If no file exists, nothing changes.

### Format

TOML *(Claude's call: it supports the list the random pool needs, and is the de facto config format for CLI tools)*, parsed with a well-established library such as `github.com/BurntSushi/toml`. Keys use the same names as the flags:

```toml
# ~/.config/tcal/config.toml — every key is optional

# Defaults for command-line flags (flags still win)
layout      = "grid"        # horizontal | grid
week-start  = "mon"         # sun | mon
highlight   = "combined"    # combined | reverse | bracket | color | none
clock-style = "block"       # block | inline | boxed
with-clock  = true
no-color    = false
bg          = "random"      # none | random | <background name>
bg-color    = true
events      = true

# Personal
birthday = "05-17"          # MM-DD (YYYY-MM-DD also accepted; the year is ignored)

[random]
# Backgrounds --bg random may pick. Omit to allow all of them.
backgrounds = ["sky", "seasons", "aquarium", "harbor", "train", "life"]
```

Any flag in tcal can have a config default, **except** these, which make no sense as defaults: `--date`, `--year`, `--print`, `--bg-time`, `--version`, `--config`.

### Precedence and errors

- Flag given on the command line > config value > built-in default.
- Config values go through the **same validation** as the flags; an invalid value is an error naming the file, key and value, and tcal exits non-zero (as it does for bad flags).
- Unknown keys are an error with the same treatment, so typos don't fail silently.
- An unknown name in `random.backgrounds` is an error. An empty list is an error ("no backgrounds to pick from").
- A missing or unreadable `--config` path given explicitly is an error; a missing default-location file is not.

### Random pool

`--bg random` picks uniformly from `random.backgrounds` (default: all registered backgrounds), subject to the event-day rule above.

## Testing expectations

- Event windows: boundaries for every event (first and last day/hour in, the day before and after out), including an Onam year with a non-10-day span, New Year crossing the year boundary, Holi's evening-before start, and Eid's ±1 day.
- The dates table: every moving event has an entry for every year 2026–2030; later years may have documented gaps; every entry is well-formed (Onam's Atham precedes Thiruvonam by 8–10 days, Diwali in Oct/Nov, etc.); a few years are spot-checked in the test against the cited sources.
- Overlap priority, the event-day random rule, `--events=false`.
- Each event scene: single-width runes, colors from its palette, stays inside its bounded area, hides when there isn't room, never draws in the calendar box (the existing shared test covers this if events are reachable through the registered `seasons` background with a forced time).
- Config: precedence (flag beats config beats default), every key's validation, unknown-key error, the random pool, the `--config` flag, and no-file behaviour unchanged.

## Left to the builder

- Event scene art, placement and animation, within the constraints above. Preview frames go to the user before finalizing.
- Package layout, e.g. an `internal/events` package (calendar + table) and event scenes alongside or inside `internal/scenes/seasons`.
- The config loader's package and how it threads into `cmd/tcal/main.go`.
- README updates: config file section, events list, `--events` and `--config` flags.
