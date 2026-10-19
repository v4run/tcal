# Holiday Events and Config File Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The `seasons` background shows a festive scene on 13 special days. A TOML config file supplies flag defaults, the birthday, and the `--bg random` pool.

**Architecture:**
- `internal/events` decides which event (if any) is active at an instant, using fixed dates plus a researched table of moving dates.
- `internal/eventart` draws one small scene per event.
- `seasons` hosts the active event's art alongside its particles.
- `internal/config` loads the TOML file. `cmd/tcal` merges it under the flags through a testable `configure` function.

**Tech Stack:** Go 1.26, Bubble Tea, `github.com/BurntSushi/toml` v1.4.0 (new dependency).

**Spec:** `docs/superpowers/specs/2026-10-19-events-and-config-design.md`

## Global Constraints

- Scene rules, all unchanged:
  - dim;
  - never inside the calendar's padded box;
  - every rune satisfies `ansi.StringWidth(string(r)) == 1`, so no `♥ ★ ❄ •` or other ambiguous-width symbols;
  - restrained palette via `scene.Color`;
  - `Cell`/`Glyph`/`Color` are O(1) lookups;
  - safe at any size, including 0×0.
- Event art that can't fit without touching the calendar box hides entirely and is never clipped.
- Events follow the real date shifted by `--bg-time`. Seasons follow the shown month.
- Only one event at a time. Priority: birthday first, then the shortest window, then table order.
- A wrong date is a failure; a missing one is acceptable. Never write a moving date from memory.
- Without a config file, behaviour is byte-for-byte unchanged.
- These flags are not allowed in the config: `date`, `year`, `print`, `bg-time`, `version`, `config`.
- Config path: `$XDG_CONFIG_HOME/tcal/config.toml`, else `~/.config/tcal/config.toml`, on every OS.
- Run `gofmt -w` and `goimports -w` on changed files. Commit after each task, ending the message with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Review Focus

1. **Midnight crossing:** a terminal left open across midnight into or out of an event window should start or stop the art on the next frame without a restart. Task 3 test `TestSeasons_EventStartsAtMidnight`.
2. **Feb 29 birthday in a non-leap year:** it should show on Feb 28 (Claude's call), never be silently skipped. Task 1 test `TestActive_Feb29BirthdayNonLeap`.
3. **`--bg-time` with events:** previewing `2026-12-25` must show the tree even when the calendar is paged to July. Task 3 test `TestSeasons_EventFollowsNowNotShown`.
4. **Config value invalid for a flag the user didn't pass:** for example `layout = "vertical"`. The error must name the file, key and value, not a bare `--layout` error. Task 8 test `TestConfigure_InvalidValueNamesFile`.
5. **Resize while an event shows:** shrinking the terminal so the art no longer fits must hide it, not clip or panic. The Task 2 generic test `TestAllArt_HidesWhenTouchingBox` runs at several sizes.

---

### Task 1: `internal/events` — kinds, windows, priority

**Files:**
- Create: `internal/events/events.go`, `internal/events/dates.go` (table scaffolding with an empty `moving` map), `internal/events/events_test.go`

**Interfaces:**
- Produces:
  - `type Kind int`, with constants in this order: `NewYear, Sankranti, LunarNewYear, Valentines, Holi, Vishu, Eid, IndependenceDay, Onam, Halloween, Diwali, Christmas, Birthday`.
  - `func (k Kind) String() string` returns, in the same order: `"new-year"`, `"sankranti"`, `"lunar-new-year"`, `"valentines"`, `"holi"`, `"vishu"`, `"eid"`, `"independence-day"`, `"onam"`, `"halloween"`, `"diwali"`, `"christmas"`, `"birthday"`.
  - `type MonthDay struct{ Month time.Month; Day int }`.
  - `func ParseBirthday(s string) (MonthDay, error)` accepts `MM-DD` or `YYYY-MM-DD` and ignores the year. It rejects impossible dates, but `02-29` is valid.
  - `type Occurrence struct{ Kind Kind; Start, End time.Time }`. `End` is exclusive.
  - `func (o Occurrence) Day(now time.Time) (day, days int)` returns a 0-based day index and the total number of calendar days. Onam uses it.
  - `func Active(now time.Time, birthday *MonthDay) (Occurrence, bool)`.
  - `func Occurrences(year int, birthday *MonthDay) []Occurrence` returns every occurrence that starts in `year`, for tests and previews.
  - In `dates.go`, the table types:
    - `type day struct{ m time.Month; d int }`
    - `type span struct{ start, end day }`. `end == start` for single-date events; Onam is Atham→Thiruvonam.
    - `var moving map[Kind]map[int]span` (year → span), filled in Task 4.

**Windows** (in `now`'s location; `[start, end)`):

| Kind | Window |
|---|---|
| NewYear | Dec 31 18:00 → Jan 2 00:00 |
| Sankranti | d → d+2 |
| LunarNewYear | d → d+3 |
| Valentines | Feb 14 → Feb 15 |
| Holi | (d−1) 18:00 → d+1 |
| Vishu | d → d+1 |
| Eid | d−1 → d+2 |
| IndependenceDay | Aug 15 → Aug 16 |
| Onam | start → end+1 |
| Halloween | Oct 31 → Nov 1 |
| Diwali | d−2 → d+3 |
| Christmas | Dec 1 → Dec 27 |
| Birthday | md → md+1; Feb 29 maps to Feb 28 in non-leap years |

Windows that may contain `now` can start in `now.Year()` or `now.Year()-1` (New Year), so check both years.

- [ ] **Step 1: Write failing tests.** In each test, swap `moving` for a fixture with `t.Cleanup` (for example Diwali 2026 = Nov 8, Onam 2026 = Aug 17→Aug 26, Holi 2026 = Mar 4, Eid 2026 = Mar 20):
  - `TestActive_Boundaries`: table-driven. For every kind, the first instant in the window is active, the last instant (End−1ns) is active, and Start−1ns and End are not.
  - `TestActive_NewYearCrossesYear`: 2026-12-31 23:59 and 2027-01-01 12:00 are NewYear; 2026-12-31 17:59 is Christmas.
  - `TestActive_Priority`: with the birthday on Dec 25, 2026-12-25 gives Birthday. On 2026-10-31, with Diwali fixture-placed on Nov 1 (window Oct 30 → Nov 4), it gives Halloween, the shorter window.
  - `TestActive_Feb29BirthdayNonLeap`: on 2027-02-28 the birthday is active, and on 2028-02-29 it is too.
  - `TestActive_MissingYearIsInactive`: Diwali with no 2031 entry gives no event on 2031-11-01.
  - `TestOccurrence_Day`: for Onam Aug 17→26, Aug 17 gives `(0,10)` and Aug 26 gives `(9,10)`.
  - `TestParseBirthday`: `"05-17"` and `"1990-05-17"` give `{May,17}`; `"02-29"` is OK; `"13-01"`, `"02-30"` and `"may 17"` are errors.

- [ ] **Step 2:** Run `go test ./internal/events/`. Expected: FAIL, because nothing is defined yet.

- [ ] **Step 3:** Implement the interfaces above. `Active` collects every candidate occurrence containing `now`, then picks one by priority: `Birthday` first, then the shortest `End.Sub(Start)`, then the lowest `Kind`.

- [ ] **Step 4:** Run `go test ./internal/events/ && go vet ./internal/events/`. Expected: PASS.

- [ ] **Step 5:** Commit with `feat(events): event windows and priority`.

---

### Task 2: `internal/eventart` — framework and Christmas tree

**Files:**
- Create: `internal/eventart/eventart.go`, `internal/eventart/christmas.go`, `internal/eventart/eventart_test.go`

**Interfaces:**
- Consumes: `events.Kind`, `events.Occurrence`, `scene.Box`, `scene.Color`.
- Produces:
  - `type Region struct{ X0, Y0, W, H int }` and `func (r Region) Touches(b scene.Box) bool`. An empty box (`Y0 == -1`) never touches.
  - The `Art` interface:
    ```go
    // Art draws one event's scene within Bounds, hiding entirely when
    // Bounds touches the calendar box.
    type Art interface {
        Resize(w, h int)
        Tick()
        SetTime(now time.Time, occ events.Occurrence)
        Bounds() Region // the full area the art may ever draw in, for the current size
        Cell(x, y int, b scene.Box) (rune, scene.Color) // 0 rune = empty
    }
    ```
  - `func New(k events.Kind, rng *rand.Rand) Art` returns nil for kinds with no art yet.
  - `var arts = map[events.Kind]func(*rand.Rand) Art{}`. Each art file registers itself in `init()`.
  - `func Kinds() []events.Kind` returns the registered kinds, sorted.
- **Christmas art:**
  - A tree about 7–9 rows tall in the bottom-right corner, made of line or block art.
  - Lights that twinkle every few ticks.
  - A star on top drawn with an ASCII `*`.
  - Colors: Green tree, lights cycling among Red, Yellow and Blue, a Yellow star.

- [ ] **Step 1: Write the generic failing tests.** They iterate `Kinds()`, so every later art is covered automatically:
  - `TestAllArt_SingleWidth`: at sizes 120×40 and 80×24, for 300 ticks, with `SetTime` on a sample occurrence, every non-zero rune has width 1.
  - `TestAllArt_StaysInBounds`: with an empty box, every non-zero cell lies inside `Bounds()`.
  - `TestAllArt_HidesWhenTouchingBox`: when the box overlaps `Bounds()` by one cell, every cell is 0. Run at 120×40, 60×20 and 30×10.
  - `TestAllArt_TinySizesNoPanic`: `Resize` and `Cell` at 0×0, 1×1, 3×2 and -1×-1, including out-of-range coordinates.
  - `TestAllArt_DrawsSomething`: at 120×40 with an empty box, at least one non-zero cell.
  - `TestChristmas_StarYellowTreeGreen`.

- [ ] **Step 2:** Run `go test ./internal/eventart/`. Expected: FAIL.

- [ ] **Step 3:** Implement the framework and the Christmas art. Use a per-tick rune and color buffer, the same pattern as `internal/scenes/rain`.

- [ ] **Step 4:** Run `go test ./internal/eventart/ && go vet ./internal/eventart/`. Expected: PASS.

- [ ] **Step 5:** Commit with `feat(eventart): art framework and christmas tree`.

---

### Task 3: Events in `seasons`, background options, random pool, `--events`

**Files:**
- Modify: `internal/scenes/seasons/seasons.go`, `internal/tui/background.go`, `internal/tui/model.go`, `cmd/tcal/main.go`
- Test: `internal/scenes/seasons/events_test.go`, `internal/tui/background_test.go`, `internal/tui/model_test.go`

**Interfaces:**
- Consumes: `events.Active`, `events.MonthDay`, `eventart.New`, `eventart.Art`.
- Produces:
  - `func (s *seasons.Scene) EnableEvents(birthday *events.MonthDay)`.
  - `SetTime(now, shown)` runs as follows:
    - The season comes from `shown`, as today.
    - The event comes from `events.Active(now, birthday)`.
    - When the active kind changes, it builds `eventart.New` and calls `Resize`.
    - It always calls `art.SetTime`.
  - `Tick` also ticks the art.
  - `Glyph` and `Color` return the art's cell when it's non-zero, otherwise the particle. Both now use their `b` argument.
  - `func (s *seasons.Scene) Event() (events.Kind, bool)` reports the active event, for tests.
  - In `tui`: `type BackgroundOptions struct{ Events bool; Birthday *events.MonthDay; RandomPool []string }`. A nil pool means all backgrounds.
  - `func (m Model) WithBackground(name string, o BackgroundOptions) Model` replaces the old signature. Registry constructors become `func(w, h int, rng *rand.Rand, o BackgroundOptions) scene.Scene`. The seasons constructor calls `EnableEvents(o.Birthday)` when `o.Events` is set.
  - Random selection draws uniformly from the pool. However, if `o.Events`, `"seasons"` is in the pool, and `events.Active(m.state.Now.Add(m.bgOffset), o.Birthday)` is true, it picks `"seasons"`.
  - Rule: `WithBackgroundTime` must be called **before** `WithBackground`, so the random rule sees the shifted time. Document this on both methods.
  - `cmd/tcal`: add `--events` (bool, default true, help text "show holiday scenes in the seasons background"). Call `WithBackgroundTime` before `WithBackground`.

- [ ] **Step 1: Write failing tests.**
  - `TestSeasons_EventFollowsNowNotShown`: with `SetTime(2026-12-25 12:00, shown=2026-07-01)`, `Event()` is Christmas and `Season()` is Summer.
  - `TestSeasons_EventStartsAtMidnight`: at 2026-11-30 23:59 there is no event; after `SetTime` to 2026-12-01 00:00, it is Christmas.
  - `TestSeasons_EventsDisabledByDefault`: `seasons.New` without `EnableEvents` reports no event on Dec 25.
  - `TestSeasons_ArtDrawsOverParticles`: some cell inside the art's `Bounds()` returns the art's rune.
  - In tui, `TestRandom_PoolOnly`: over 200 picks with pool `{"rain","lava"}`, only those two appear.
  - In tui, `TestRandom_EventDayPicksSeasons`: with the pool `{"rain","seasons"}`, `Events: true`, and the background time 2026-12-25, the pick is `seasons` 50 times out of 50.
  - In tui, `TestRandom_EventDayWithoutSeasonsInPool`: with the pool `{"rain"}`, the pick is `rain`.
  - In tui, `TestAllEvents_NeverTouchCalendar`: for each occurrence in `events.Occurrences(2026, &MonthDay{May,17})` whose kind is in `eventart.Kinds()`, a model at 100×40 with clock, using `WithBackgroundTime(occ.Start.Add(time.Hour))` and `WithBackground("seasons", {Events:true, Birthday:…})`, ticked 100 times, leaves the padded calendar box identical to the plain frame. Reuse the box-comparison helper from `TestAllBackgrounds_NeverTouchCalendar`, extracting it to a helper func.
  - Update the existing `WithBackground` call sites in tests to pass `BackgroundOptions{}`.

- [ ] **Step 2:** Run `go test ./internal/scenes/seasons/ ./internal/tui/`. Expected: FAIL.

- [ ] **Step 3:** Implement the interfaces above.

- [ ] **Step 4:** Run `go test ./... && go vet ./...`. Expected: PASS. Then check by hand that `go run ./cmd/tcal --bg seasons --bg-time 2026-12-25` shows the tree.

- [ ] **Step 5:** Commit with `feat(seasons): holiday events, random pool and --events`.

---

### Task 4: The moving-dates table (research)

**Files:**
- Modify: `internal/events/dates.go`
- Create: `internal/events/dates_test.go`

**Interfaces:**
- Consumes: `moving`, `span` and `day` from Task 1.
- Produces: `moving` filled for Sankranti, LunarNewYear, Holi, Vishu, Eid, Onam and Diwali, covering 2026–2040.

**Which day each entry holds:**
- Sankranti: the Makar Sankranti day.
- Holi: the day of colours (Rangwali Holi / Dhulandi), not Holika Dahan.
- Diwali: the Lakshmi Puja day.
- Onam: Atham as the start and Thiruvonam as the end, from the Malayalam calendar.
- Eid al-Fitr: the expected Gregorian date.
- Lunar New Year: the first day.
- Vishu: as observed in Kerala.

- [ ] **Step 1: Research.** For each event and year, look the date up with web search or fetch:
  - drikpanchang.com (Hindu festivals, with the city set to an Indian city, e.g. Kochi for Onam and Vishu, and New Delhi for Holi, Diwali and Sankranti);
  - timeanddate.com (Eid, Lunar New Year);
  - official government holiday lists where available.

  Record each date's source URL in a comment block above its event's map. Where sources disagree:
  - for the Indian festivals, use the date most widely observed in India;
  - for Eid, use the expected or astronomical date;
  - note the disagreement in a comment.

  If no reliable published date exists, leave the year out and write a comment with the reason. Never write a date from memory.

- [ ] **Step 2: Write tests.**
  - `TestTable_NearYearsComplete`: every moving kind has an entry for every year 2026–2030.
  - `TestTable_WellFormed`, which checks every entry:
    - the span's end is not before its start;
    - for Onam, end − start is 8–10 days;
    - Diwali falls in Oct or Nov;
    - Holi in Feb or Mar;
    - Vishu on Apr 13–15;
    - Sankranti on Jan 13–16;
    - Lunar New Year between Jan 20 and Feb 21;
    - Onam's Thiruvonam in Aug or Sep;
    - Eid's year matches its key.
  - `TestTable_SpotChecks`: three or more entries per event, asserted against the cited source values. Copy them from the sources, not from the table.

- [ ] **Step 3:** Run `go test ./internal/events/`. Expected: PASS.

- [ ] **Step 4:** Commit with `feat(events): moving holiday dates 2026-2040 with sources`.

---

### Task 5: Event art, set A — New Year, Diwali, Lunar New Year, Eid

**Files:**
- Create: `internal/eventart/newyear.go`, `diwali.go`, `lunarnewyear.go`, `eid.go`, plus specific tests in `internal/eventart/eventart_test.go`
- Consumes: the `Art` interface from Task 2. The generic tests cover all four automatically.

| Art | Placement and behaviour | Palette |
|---|---|---|
| New Year | Fireworks in the top margin: bursts that expand and fade (`*`, `·`, `+`, `.`). The burst rate depends on `now`'s distance from midnight and peaks in 23:30–00:30. | Red, Yellow, Magenta, Cyan, Green |
| Diwali | A row of 5–9 diyas along the bottom strip, each a small lamp with a flickering flame (alternating glyphs). Occasional small top-margin bursts, sharing a firework helper with New Year. | Yellow flames, Red lamps |
| Lunar New Year | 3–5 lanterns hanging from the top edge on strings, swaying one column back and forth. | Red lanterns, Yellow tassels |
| Eid | A crescent moon (half-block or braille) in a top corner, with 2–3 small hanging lanterns. | White moon, Yellow lanterns |

- [ ] **Step 1:** Write the specific failing tests:
  - `TestNewYear_MoreBurstsAtMidnight`: count non-zero cells over 200 ticks at 23:59 and at 20:00. Midnight must be more than 2× as busy.
  - `TestDiwali_FlamesFlicker`: some flame cell changes rune within 10 ticks.
  - `TestLanterns_Sway`: some lantern cell's x position changes over 40 ticks.
  - `TestEid_MoonWhite`.
- [ ] **Step 2:** Run `go test ./internal/eventart/`. Expected: FAIL.
- [ ] **Step 3:** Implement the four arts, registering each in `init()`. Put the shared firework burst in `fireworks.go`.
- [ ] **Step 4:** Run `go test ./internal/eventart/ ./internal/tui/ && go vet ./internal/eventart/`. Expected: PASS, and `TestAllEvents_NeverTouchCalendar` now covers these kinds.
- [ ] **Step 5:** Commit with `feat(eventart): new year, diwali, lunar new year, eid`.

---

### Task 6: Event art, set B — Sankranti, Independence Day, Holi, Valentine's, Onam, Vishu, Halloween, Birthday

**Files:**
- Create: `internal/eventart/{sankranti,independence,holi,valentines,onam,vishu,halloween,birthday}.go`, plus specific tests in `internal/eventart/eventart_test.go`

| Art | Placement and behaviour | Palette |
|---|---|---|
| Sankranti | 2–4 diamond kites (`/\` `\/` line art) with dangling tails, bobbing in the top half, each on a string toward the bottom edge. | Red, Yellow, Blue, Green, Magenta kites, Default strings |
| Independence Day | A small flag on a pole in a top corner, three bands with a centre wheel (`o`). The flag edge ripples between two frames. | Red (for saffron), White, Green; Blue wheel |
| Holi | Every few seconds, a soft burst of `·`, `.` and `:` dots blooms in the margins and fades out. | each burst one of Magenta, Yellow, Green, Cyan, Red |
| Valentine's | 3–6 ASCII or block hearts, e.g. `▄ ▄` over `▀█▀`, rising slowly and fading at the top. | Red, Magenta |
| Onam | A round pookalam in a bottom corner. The number of filled rings is `ceil((day+1)/days × maxRings)` from `occ.Day(now)`, so it is complete on the last day. | rings alternate Yellow, Red, Magenta, Green |
| Vishu | Golden konna flowers (`*`, `·`, `,`) drifting down from the top in light clusters, and a small lamp in a bottom corner. | Yellow flowers, Yellow/Red lamp |
| Halloween | A jack-o'-lantern (3–4 rows, carved face from `^`, `v`, `▀`) in a bottom corner, with 1–2 bats (`^v^` / `-v-` wing flap) flitting in the top half. | Yellow pumpkin, Default bats |
| Birthday | A cake (2–3 tiers) with 3–5 candles whose flames flicker, in a bottom corner, and gentle confetti falling. | Magenta/White cake, Yellow flames, mixed confetti |

- [ ] **Step 1:** Write the specific failing tests:
  - `TestOnam_RingsGrowWithDay`: drawn cells on day 0 < day 4 < the last day, and the last day equals the full pookalam.
  - `TestOnam_NineAndElevenDayYears`: with a 9-day or 11-day occurrence, the pookalam is full on the last day and not before.
  - `TestHalloween_BatsFlap`: a bat glyph changes within 10 ticks.
  - `TestBirthday_CandlesFlicker`.
  - `TestKites_HaveStrings`: some Default-colored cell lies below each kite.
- [ ] **Step 2:** Run `go test ./internal/eventart/`. Expected: FAIL.
- [ ] **Step 3:** Implement the eight arts, each registered in `init()`.
- [ ] **Step 4:** Run `go test ./internal/eventart/ ./internal/tui/ && go vet ./...`. Expected: PASS, and `Kinds()` returns all 13.
- [ ] **Step 5:** Commit with `feat(eventart): remaining holiday scenes`.

---

### Task 7: Preview checkpoint (user sign-off)

- [ ] **Step 1:** Write a temporary test that writes an ANSI gallery. For each of the 13 kinds, it renders a 100×34 frame of `seasons` with events on and the time forced to the middle of that kind's 2026 window, and writes all of them to `<scratchpad>/events-gallery.ansi`. Delete the test afterwards.
- [ ] **Step 2:** Send the user the `cat <path>` command and a plain-text snapshot of each frame, then **stop and wait for their sign-off or changes**. Apply the requested art changes and repeat until approved. Commit art changes with `change(eventart): …`.

---

### Task 8: `internal/config` and `configure` in main

**Files:**
- Create: `internal/config/config.go`, `internal/config/config_test.go`
- Modify: `cmd/tcal/main.go`, `cmd/tcal/main_test.go`, `go.mod` and `go.sum` (via `go get github.com/BurntSushi/toml@v1.4.0`)

**Interfaces:**
- Produces in `internal/config`:
  - `func DefaultPath() string` returns `$XDG_CONFIG_HOME/tcal/config.toml`, else `$HOME/.config/tcal/config.toml`.
  - `type Config struct{ Path string; Flags map[string]string; Birthday string; RandomPool []string }`. `Flags` maps flag name to a value formatted with `fmt.Sprint`; `RandomPool` is nil when unset.
  - `func Load(path string, mustExist bool, allowedFlags []string) (Config, error)`:
    - A missing file returns an empty `Config` and no error, unless `mustExist` is set.
    - Errors are prefixed with the path. These are all errors:
      - an unknown key (use `toml.MetaData.Undecoded()` or decode to `map[string]any` and check);
      - a non-scalar flag value;
      - `random.backgrounds` that isn't a list of strings;
      - an empty `random.backgrounds` list.
- Produces in `cmd/tcal`:
  - A `runConfig` struct: `state render.State`, `opts render.Options`, `print bool`, `bg string`, `bgOpts tui.BackgroundOptions`, `bgMono bool`, and `bgTime *time.Time`, which is nil when `--bg-time` is unset.
  - `func configure(args []string, stderr io.Writer, now time.Time) (runConfig, error)`. It holds everything `run` did before starting the program. `run` calls it, then launches print mode or the TUI.
  - Flow inside `configure`:
    1. Parse the flags.
    2. Record which flags were set, via `fs.Visit`.
    3. Load the config from `--config`, or from `DefaultPath()` if not given. `mustExist` is true only when `--config` was given. `allowedFlags` is every flag name minus the forbidden list.
    4. For each config flag the CLI didn't set, validate it with the same parser the flag uses (keep a `validators map[string]func(string) error` that wraps `parseLayout`, `parseWeekStart`, `parseHighlight`, `parseClockStyle` and `parseBackground`, plus `strconv.ParseBool` and `strconv.Atoi` for bool and int flags). On failure, return `config <path>: <key> = <value>: <err>`. On success, call `fs.Set`.
    5. Validate each name in `RandomPool` against `tui.Backgrounds()`. An unknown name is an error naming the path.
    6. Turn `Birthday` into a `*events.MonthDay` with `events.ParseBirthday`, wrapping the error with the path.
  - Add a `--config` flag (string, default "", help text "config file (default ~/.config/tcal/config.toml)").

- [ ] **Step 1: Write the failing tests.** Config files are written to `t.TempDir()`; for no-file cases, set `XDG_CONFIG_HOME` to an empty temp dir with `t.Setenv`.
  - Config package tests:
    - `TestLoad_MissingDefaultIsEmpty`.
    - `TestLoad_MissingExplicitErrors`.
    - `TestLoad_UnknownKeyErrors`, with the message containing the key and the path.
    - `TestLoad_ForbiddenKeyErrors`, for example `print = true`.
    - `TestLoad_RandomPool`.
    - `TestLoad_EmptyPoolErrors`.
    - `TestLoad_ScalarFormatting`: `with-clock = true` gives `"true"` and `months = 6` gives `"6"`.
  - Main tests:
    - `TestConfigure_NoFileUnchanged`: `configure` with no args gives the same state and options as before, using `XDG_CONFIG_HOME` pointed at an empty dir.
    - `TestConfigure_ConfigSetsDefault`: `layout = "grid"` gives `LayoutGrid`.
    - `TestConfigure_FlagBeatsConfig`: `layout = "grid"` together with `--layout horizontal` gives `LayoutHorizontal`.
    - `TestConfigure_InvalidValueNamesFile`: `layout = "vertical"` gives an error containing the path, `layout` and `vertical`.
    - `TestConfigure_Birthday`: `birthday = "05-17"` gives `bgOpts.Birthday == &{May,17}`.
    - `TestConfigure_RandomPoolUnknownName`.
    - `TestConfigure_EventsFalseFromConfig`.
    - `TestConfigure_ExplicitConfigFlag`.
- [ ] **Step 2:** Run `go test ./internal/config/ ./cmd/...`. Expected: FAIL.
- [ ] **Step 3:** Implement the interfaces above.
- [ ] **Step 4:** Run `go test ./... && go vet ./...`. Expected: PASS. Check that `go run ./cmd/tcal --print` output is identical before and after this change.
- [ ] **Step 5:** Commit with `feat(config): TOML config for flag defaults, birthday and random pool`.

---

### Task 9: README and final verification

**Files:** Modify `README.md`.

- [ ] **Step 1:** Update the README:
  - **Flags table:** add `--events` and `--config`.
  - **New *Events* section:**
    - a table with the 13 events, their windows and their scenes;
    - previewing with `--bg-time`;
    - that event dates follow the real date while seasons follow the shown month;
    - how to turn events off;
    - that moving dates cover 2026–2040.
  - **New *Config file* section:** the location, the full annotated example from the spec, precedence, and the forbidden keys.
  - **Smoke checklist:** add `tcal --bg seasons --bg-time 2026-12-25`, a config with `bg = "random"` and a pool, and an invalid config value.
- [ ] **Step 2:** Run `gofmt -l . && goimports -l . && go vet ./... && go test -count=1 ./...`. Expected: no files listed, and every package passes.
- [ ] **Step 3:** Commit with `docs: events and config file`.
