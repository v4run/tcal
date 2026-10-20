# Smooth Backgrounds Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `smooth` style for every live-mode background. Smooth scenes are drawn in braille dots, run at about 30 fps with time-based motion, and use true colour where the terminal supports it. Today's backgrounds stay frozen and remain the default `classic` style.

**Architecture:**
- New shared packages: `internal/palette` (colour modes, blending, contrast, nearest-ANSI) and `internal/dots` (a braille dot canvas with float plotting and sprites).
- `internal/scene` gains two optional interfaces: `TrueColored` and `Stepper`.
- `internal/tui` gets a second registry, style selection, elapsed-time stepping, a 24-bit colour path in `overlay`, frame caching and synchronized output.
- Smooth scenes live in `internal/smooth/<name>`, and smooth holiday art in `internal/smooth/eventart`.
- Classic scene packages are not edited.

**Tech Stack:** Go 1.26, Bubble Tea v1.3.10, lipgloss v1.1.0, `github.com/charmbracelet/x/ansi`.

**Spec:** `docs/superpowers/specs/2026-10-20-smooth-backgrounds-design.md`. Read it. Its "Builder's questions, settled" section is binding.

## Global Constraints

- Classic output must stay byte-for-byte unchanged. The classic default must stay unchanged too. No file under `internal/scenes/`, `internal/eventart/`, `internal/runner/` or `internal/life/` is edited.
- `--bg-style classic|smooth`: the default is `classic`, and it is also the config key `bg-style`.
- `--bg-palette auto|truecolor|ansi`: the default is `auto`, and it is also the config key `bg-palette`. `auto` means true colour when `COLORTERM` is `truecolor` or `24bit`.
- `NO_COLOR` / `--no-color` beat `--bg-palette`: no tint at all. `--bg-color=false` means dim foreground glyphs in both styles.
- Smooth scenes tick at 33 ms. `dt` comes from the tick message timestamp and is clamped to the range 0 to 100 ms.
- Braille cells U+2800–U+28FF. Single-width runes only, so `ansi.StringWidth(string(r)) == 1`.
- Dim without faint: every emitted true-colour value, after blending, has a WCAG contrast ratio of **≤ 4.5** against the background. This holds on both a black and a white background (white text on black is 21).
- The canvas colour rule: the last dot drawn into a cell sets that cell's colour.
- Never draw inside the calendar box plus padding. Art hides whole and is never clipped.
- Per-frame cost: one `Step(33ms)` plus a full `Glyph`/`RGB` sweep at 200×60 averages **≤ 2 ms**. *(Ruling: the spec cites a "0.2 ms existing test" that does not exist. 2 ms is 6% of a 33 ms frame.)*
- Run `gofmt -w` and `goimports -w` on changed files before committing. Commits end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. Stage by explicit path, and never commit `cmd/tcal/tcal`.

## Review Focus

1. Resizing while smooth is running, including to 0×0 or 1×1. There must be no panic and no stale dots outside the new size. Owned by Task 2 (canvas) and the shared tests in Task 3.
2. A terminal suspended or backgrounded for minutes and then resumed. `dt` is clamped, so nothing jumps or teleports. Owned by Task 3, in `TestStep_ClampsLongGap`.
3. Light-theme terminals. Blended colours must stay dim against a light background, not just a dark one. Owned by Task 3, in `TestSmooth_DimOnLightAndDark`.
4. `tmux`, where `COLORTERM` is often unset. ANSI fallback must be used under `auto`, and `--bg-palette truecolor` forces 24-bit. Owned by Task 1, in `TestDetect`.
5. Switching the layout with `1` or `2` while smooth is running. The cached frame must be invalidated so the calendar never shows a stale layout. Owned by Task 3, in `TestFrameCache_InvalidatesOnLayoutKey`.

---

### Task 1: `internal/palette` and the scene interfaces

**Files:**
- Create: `internal/palette/palette.go`, `internal/palette/palette_test.go`
- Modify: `internal/scene/scene.go` (additions only)

**Interfaces:**
- Produces in `scene`:
  - `type RGB struct{ R, G, B uint8 }`
  - `type TrueColored interface { RGB(x, y int, b Box) RGB }`. It is called only where `Glyph` returned a rune.
  - `type Stepper interface { Step(dt time.Duration) }`. When a scene implements it, the TUI calls `Step` instead of `Tick`.
- Produces in `palette`:
  - `type Mode int` with `ANSI`, `TrueColor`.
  - `func Parse(s string) (Mode, bool, error)`. It accepts `auto|truecolor|ansi`, case-insensitive. The bool is true for `auto`.
  - `func Detect(getenv func(string) string) Mode`.
  - `func Blend(c, bg scene.RGB, keep float64) scene.RGB`. This is a linear mix: `keep` = 1 gives `c`, and 0 gives `bg`.
  - `func Contrast(a, b scene.RGB) float64`. This is the WCAG 2 contrast ratio, from 1 to 21.
  - `func Nearest(c scene.RGB) scene.Color`. It picks a hue-preserving nearest from Red, Green, Yellow, Blue, Magenta, Cyan, White. Low-saturation colours map to `scene.Default`.
  - `var Dark = scene.RGB{0, 0, 0}`, `Light = scene.RGB{255, 255, 255}`.

- [ ] **Step 1: Write the failing tests** in `palette_test.go`:
  - `TestParse`: `"auto"` gives `(TrueColor or ANSI, true, nil)`. `"TrueColor"` gives `(TrueColor, false, nil)`. `"ansi"` gives `(ANSI, false, nil)`. `"256"` returns an error naming the value.
  - `TestDetect`: `COLORTERM=truecolor` and `COLORTERM=24bit` give `TrueColor`. An empty value or `"yes"` gives `ANSI`.
  - `TestContrast`: `Contrast(Dark, Light)` is within 0.01 of 21. `Contrast(x, x)` is 1. The function is symmetric.
  - `TestBlend`: keep 1 gives c. Keep 0 gives bg. Keep 0.5 of `{200,0,0}` over black gives `{100,0,0}`.
  - `TestNearest`:
    - `{200,40,40}` maps to Red, `{40,180,60}` to Green, `{60,90,220}` to Blue, `{230,200,60}` to Yellow, `{60,200,210}` to Cyan, `{190,60,190}` to Magenta, `{235,235,235}` to White.
    - `{128,128,128}` maps to Default.
- [ ] **Step 2:** Run `go test ./internal/palette/`. Expected: FAIL (package missing).
- [ ] **Step 3:** Implement. For `Nearest`:
  - Convert to HSV.
  - If saturation < 0.25: return White when value > 0.85, otherwise Default.
  - Otherwise pick by hue sector: red <20° or ≥340°, yellow <70°, green <160°, cyan <200°, blue <260°, magenta <340°.
- [ ] **Step 4:** Run `go test ./internal/palette/ ./internal/scene/... && go vet ./...`. Expected: PASS.
- [ ] **Step 5:** Commit: `feat(palette): colour modes, blending and contrast`.

### Task 2: `internal/dots` braille canvas

**Files:**
- Create: `internal/dots/dots.go`, `internal/dots/sprite.go`, `internal/dots/dots_test.go`

**Interfaces:**
- Consumes: `scene.RGB`.
- Produces:
  - `type Canvas struct` (zero value usable).
  - `func (c *Canvas) Reset(w, h int)`. The size is in cells, and negatives are clamped to 0.
  - `func (c *Canvas) Clear()`.
  - `func (c *Canvas) Dot(x, y float64, col scene.RGB)`. `x` and `y` are in **cell units**, and the dot lands at column `floor(x*2)` and row `floor(y*4)` in dot space. Off-canvas dots are ignored. The last dot drawn sets the cell colour.
  - `func (c *Canvas) Glyph(cx, cy int) rune`. It returns 0 for an empty or off-canvas cell.
  - `func (c *Canvas) RGB(cx, cy int) scene.RGB`.
  - `func (c *Canvas) Size() (w, h int)`.
  - `func (c *Canvas) Text(cx, cy int, s string, col scene.RGB)`. It places ordinary single-width runes, for details braille can't carry, and overwrites any dots in those cells.
  - `type Sprite struct` and `func ParseSprite(rows ...string) Sprite`. Rows are in dot space: `'#'` is a dot and any other character is empty.
  - `func (s Sprite) Size() (dw, dh int)`.
  - `func (c *Canvas) DrawSprite(s Sprite, x, y float64, col scene.RGB)`. Its top-left corner is at cell-unit `(x, y)`.

- [ ] **Step 1: Write the failing tests:**
  - `TestDot_BitLayout`: dots at `(0,0)`, `(0.5,0)`, `(0,0.25)`, `(0,0.75)` and `(0.5,0.75)` give the braille bits 0x01, 0x08, 0x02, 0x40 and 0x80, OR-ed into one cell. The result is `'⣋'`.
  - `TestDot_SubCellSteps`: moving a dot by x += 0.5 changes the dot column each step. Moving it by y += 0.25 changes the dot row each step.
  - `TestDot_LastColourWins`.
  - `TestDot_OffCanvasIgnored`: negative positions, positions ≥ size, and a 0×0 canvas cause no panic.
  - `TestReset_ClearsAndResizes`.
  - `TestText_OverwritesDots`.
  - `TestSprite_ParseAndDraw`: a 2×4 all-`#` sprite drawn at (1,1) fills cell (1,1) with `'⣿'`. Drawn at (1.5,1), it spans cells 1 and 2.
  - `TestGlyphs_SingleWidth`: every rune `Glyph` can return has `ansi.StringWidth` 1.
- [ ] **Step 2:** Run `go test ./internal/dots/`. Expected: FAIL.
- [ ] **Step 3:** Implement. Store a `[]uint8` dot mask and a `[]scene.RGB` per cell, plus a `[]rune` for text cells. `Glyph` and `RGB` are O(1) lookups.
- [ ] **Step 4:** Run `go test ./internal/dots/ && go vet ./internal/dots/`. Expected: PASS.
- [ ] **Step 5:** Commit: `feat(dots): braille dot canvas and sprites`.

### Task 3: TUI plumbing — styles, stepping, true-colour overlay, flags

**Files:**
- Modify: `internal/tui/background.go`, `internal/tui/model.go`, `cmd/tcal/main.go`
- Create: `internal/tui/smooth.go`, which holds the smooth registry and is the only place smooth constructors are listed
- Create: `internal/tui/smooth_test.go`, `internal/tui/classic_golden_test.go`, `internal/tui/testdata/classic/*.golden`
- Modify tests: `cmd/tcal/main_test.go`, `internal/tui/model_test.go`

**Interfaces:**
- Consumes: Task 1 (`scene.TrueColored`, `scene.Stepper`, `palette.*`).
- Produces:
  - `const StyleClassic = "classic"` and `StyleSmooth = "smooth"`. `func Styles() []string`.
  - `BackgroundOptions.Style string`. Empty means classic.
  - `var smoothBackgrounds map[string]func(w, h int, rng *rand.Rand, o BackgroundOptions) scene.Scene` in `smooth.go`.
    - It starts with `life`, `langton`, `automaton` and `ants` pointing at the same constructors as classic.
    - Every other name falls back to its classic constructor **until its smooth task registers it**. `newScene` uses the smooth entry if present, otherwise classic.
    - `Backgrounds()` is unchanged: the same names exist in both styles.
  - `func (m Model) WithBackgroundPalette(mode palette.Mode, bg scene.RGB) Model`. The default is ANSI with a `palette.Dark` background.
  - `bgTickMsg` becomes `bgTickMsg time.Time`. The model remembers the last tick time. For a `scene.Stepper` it calls `Step(clamp(now-last, 0, 100ms))`, and on the first tick it uses `Interval()`. Other scenes are ticked exactly as before.
  - `overlay` true-colour path:
    - when the style is `styleColor`, the scene implements `scene.TrueColored` and the mode is `TrueColor`, emit `\x1b[38;2;R;G;Bm` with `palette.Blend(sceneRGB, bg, keepFactor)`;
    - in ANSI mode, emit `sgrColor(palette.Nearest(blended))`;
    - `keepFactor` is a package constant chosen so the dimness test passes.
  - A frame cache: `View` reuses the `render.Frame` string until the state changes. That means any change to Width, Height, Now (per second), Anchor, Layout or Today.
  - main:
    - `--bg-style` with a validator, `parseBgStyle`;
    - `--bg-palette` with a validator using `palette.Parse`;
    - before starting the program, `lipgloss.HasDarkBackground()` selects `palette.Dark` or `palette.Light`;
    - `NO_COLOR`/`--no-color` keep the plain style regardless of palette.

- [ ] **Step 1: Write the failing tests:**
  - `TestClassicGolden`: classic output must not change.
    - For every classic background, at 100×30, with rng seed `(1,2)` and fixed time `2026-04-15 10:00 local`, run 20 ticks and compare `View()` to `testdata/classic/<name>.golden`.
    - Generate the goldens **from `main` before any change** with `go test ./internal/tui -run TestClassicGolden -update` (add an `-update` flag), and commit them in this task's first commit.
    - Build each scene through its `backgrounds[name]` constructor with the fixed rng. Don't use `newScene`, which seeds from the wall clock.
    - If any classic scene has nondeterminism beyond the rng, such as wall-clock reads, fix it via the test's inputs, not by editing the scene.
  - `TestStyle_DefaultIsClassic`: `WithBackground("rain", BackgroundOptions{})` gives a scene from package `scenes/rain`.
  - `TestStep_UsesElapsedTime`: a fake `Stepper` scene records the `dt` it receives. Two `bgTickMsg`s 50 ms apart give `dt` = 50 ms.
  - `TestStep_ClampsLongGap`: ticks 10 minutes apart give a `dt` of 100 ms.
  - `TestOverlay_TrueColorEmits24Bit`: a fake `TrueColored` scene in TrueColor mode produces `38;2;`. In ANSI mode it produces only `3[0-7]m` / `39m`, with no `38;2`.
  - `TestOverlay_NoColorStillPlain`: plain style with TrueColor mode has no escapes.
  - `TestSmooth_DimOnLightAndDark`:
    - For every smooth background, against both `palette.Dark` and `palette.Light`, run 100 steps at 120×40 and collect every blended RGB.
    - For seasons, also cover each event via `WithBackgroundTime`.
    - Assert `palette.Contrast(c, bg) <= 4.5`.
  - `TestSmooth_SharedRules`:
    - For every smooth background, run the existing checks: calendar intact, colour well-formed, single-width runes.
    - Reuse the helpers behind `TestAllBackgrounds_NeverTouchCalendar` and `TestAllBackgrounds_ColorWellFormed`, parameterised by style and run in TrueColor mode as well.
    - Mid-run, resize to 1×1, then 0×0, then back to 120×40, and check that nothing panics and no glyph appears outside the current size.
  - `TestSmooth_FrameCost`: for every smooth background at 200×60, the average over 30 frames of `Step(33ms)` plus a full `Glyph`/`RGB` sweep is ≤ 2 ms. Skip it under `-short`.
  - `TestFrameCache_InvalidatesOnLayoutKey`: after `View()`, pressing `2` makes the next `View()` show grid layout.
  - `TestFrameCache_SameStateReusesFrame`: count `render.Frame` calls through a package-level hook var, or compare string identity.
  - In main tests:
    - `TestConfigure_BgStyleFlagAndConfig`: config `bg-style = "smooth"` gives smooth, and `--bg-style classic` overrides it.
    - `TestConfigure_BgStyleInvalid`.
    - `TestConfigure_BgPalette`: `auto` with `COLORTERM` unset gives ANSI, and `truecolor` gives TrueColor.
    - `TestConfigure_NoColorBeatsPalette`.
- [ ] **Step 2:** Run `go test ./internal/tui/ ./cmd/...`. Expected: the new tests FAIL, and the golden test passes against the freshly generated goldens.
- [ ] **Step 3:** Implement the interfaces above. `runConfig` gains `bgStyle string` and `bgPalette palette.Mode`.
- [ ] **Step 4:** Run `go test ./... && go vet ./...`. Expected: PASS. `TestClassicGolden` is still green.
- [ ] **Step 5:** Commit: `feat(tui): smooth style plumbing, true colour and elapsed-time stepping`.

### Task 4: Synchronized output

**Files:** `cmd/tcal/main.go`, plus a new `internal/tui/syncwriter.go` and `syncwriter_test.go` if a wrapper is needed.

**Interfaces:** Produces `func NewSyncWriter(w io.Writer) io.Writer` if it is needed.

- [ ] **Step 1:** Check Bubble Tea v1.3.10's standard renderer source in the module cache for `2026` / `SyncdUpdates`. Record the finding in the commit message and the report.
- [ ] **Step 2:** If it already brackets frames, add a test asserting that the program's output for one frame contains `\x1b[?2026h` … `\x1b[?2026l`, using `tea.WithOutput` into a buffer and `tea.WithInput(nil)`. Stop there.
- [ ] **Step 3:** Otherwise, write `TestSyncWriter_BracketsEachFlush`: each `Write` call is wrapped as `\x1b[?2026h` + payload + `\x1b[?2026l`, and empty writes pass through. Implement `NewSyncWriter`, which wraps each `Write` because the renderer flushes once per frame. Wire it with `tea.WithOutput(tui.NewSyncWriter(os.Stdout))` for **smooth style only**, so classic output is unchanged.
- [ ] **Step 4:** If neither approach works, document the reason in README's smoke checklist and the report. Do not upgrade Bubble Tea unless the change is confined to `go.mod` and `cmd/tcal`.
- [ ] **Step 5:** Run `go test ./... && go vet ./...`. Commit: `feat(tui): synchronized output for smooth frames`.

### Task 5: Smooth rain (the spike, made real)

**Files:** Create `internal/smooth/rain/rain.go`, `rain_test.go`. Register `"rain"` in `internal/tui/smooth.go`.

**Interfaces:**
- Consumes: `dots.Canvas`, `scene.Stepper`, `scene.TrueColored`.
- Produces:
  - `func New(w, h int, rng *rand.Rand) *Scene`, implementing `scene.Scene`, `Stepper` and `TrueColored`.
  - `Interval()` is 33 ms.
  - `Tick()` = `Step(33ms)`.

The behaviour is pinned by the spec's table and the spike:
- Density: one drop per 70 cells.
- Fall speed: 0.6–1.05 rows per 70 ms, which works out to 8.6–15 rows per second.
- Each drop is a two-dot streak a quarter row apart, tilted by the wind.
- Wind:
  - The wind eases toward a new goal every 2–7 s.
  - About 30% of goals are 0; the rest are uniform in ±1.2 columns per row fallen.
  - The easing rate is 0.02 per 33 ms step, scaled by dt.
- Drops wrap horizontally.
- The ground line is the bottom row as `─` (a text cell). Splashes are dots just above the ground.
- Colours: drops are soft blue `{110,150,230}`, splashes cyan `{120,200,210}`, and the ground a neutral `{150,150,150}`.

- [ ] **Step 1: Write the failing tests:**
  - `TestRain_DropGlides`: a single drop with speed v and no wind, stepped 100× by 33 ms. At each step its plotted dot row is within 1 of `floor((y0 + v*t)*4)`. Across the run, the row never decreases and never jumps by more than `ceil(v*0.033*4)+1`.
  - `TestRain_WindSlantsAndChanges`: with a fixed seed, over 20 s of steps the wind takes at least 2 distinct goals, and |wind| stays ≤ 1.2.
  - `TestRain_WrapsHorizontally`: a drop pushed past the right edge reappears at the left.
  - `TestRain_DensityAndResize`: resizing keeps the drop count at `max(1, w*(h-1)/70)`. Resizing to 0×0 and 1×1 doesn't panic.
  - `TestRain_GroundAndSplash`: a landing produces splash dots on row h-2 and respawns the drop above the screen.
- [ ] **Step 2:** Run `go test ./internal/smooth/rain/`. Expected: FAIL.
- [ ] **Step 3:** Implement, and register it in `smooth.go`.
- [ ] **Step 4:** Run `go test ./internal/smooth/... ./internal/tui/ && go vet ./...`. Expected: PASS. This includes `TestSmooth_*` for rain.
- [ ] **Step 5:** Commit: `feat(smooth): rain with sub-cell drops and gusting wind`.

**The pattern for Tasks 6–12** (every smooth scene follows it):
- The package is `internal/smooth/<name>`, with `New(w, h int, rng *rand.Rand) *Scene`.
- The scene implements `Scene`, `Stepper` and `TrueColored`. It also implements `Clocked` where the classic version does.
- Motion is in units per second, scaled by `dt`. `Interval()` is 33 ms, and `Tick()` = `Step(33ms)`.
- The scene renders once into a `dots.Canvas` per `Step`/`Resize`/`SetTime`, so `Glyph`/`RGB` are lookups.
- Each scene is registered in `smooth.go`.
- Each scene has a `TestX_Glides` test. A named moving object's plotted dot position stays within 1 dot of its ideal position `p0 + v*t` over 100 steps of 33 ms.
- Each scene has its own palette of RGB values. The dimness test in Task 3 polices them after blending, so pick natural colours.
- Character art is redrawn as `dots.Sprite`s. The classic look and placement are the reference, and recognisability matters most. `Canvas.Text` is allowed for details braille can't carry.
- Hiding rules match the classic scene: bottom-strip scenes hide whole when they don't fit beside the calendar.

### Task 6: Smooth starfield and flow

**Files:** `internal/smooth/starfield/*`, `internal/smooth/flow/*`, `smooth.go`

- Starfield:
  - Stars stream outward from the centre at sub-cell speed.
  - Brightness comes from depth: a far star is `{90,90,120}`, a near one `{200,200,230}`.
  - Test: `TestStarfield_Glides`, which tracks one star's radial position.
  - Test: `TestStarfield_DepthBrightens`, which checks that a star's RGB luminance rises as it approaches.
- Flow:
  - Particles follow classic flow's field function, reimplemented in this package. Classic is never imported, because it gets deleted later.
  - Each particle has a fading trail of the last 3 positions, with RGB fading toward dim.
  - Test: `TestFlow_Glides`.
  - Test: `TestFlow_TrailFades`, which checks that the trail dots get monotonically darker.
- [ ] Steps: write the failing tests, run them (FAIL), implement both scenes and register them, run `go test ./internal/smooth/... ./internal/tui/ && go vet ./...` (PASS), then commit `feat(smooth): starfield and flow`.

### Task 7: Smooth lava and hourglass

**Files:** `internal/smooth/lava/*`, `internal/smooth/hourglass/*`, `smooth.go`

- Lava:
  - Blobs are metaballs sampled per dot, with the threshold field drawn as dots.
  - Colour is graded by field strength, from `{150,40,20}` to `{230,140,40}`.
  - Blobs rise and sink smoothly.
  - Test: `TestLava_Glides`, which tracks a blob's centre.
  - Test: `TestLava_ColourByHeat`.
  - Mind the frame budget: sample the field at dot resolution only inside each blob's bounding box.
- Hourglass:
  - Same behaviour and clock coupling as classic: `SetTime` drives the fill.
  - Grains fall as dots and pile in dot rows.
  - Test: `TestHourglass_GrainsGlide`.
  - Test: `TestHourglass_FillFollowsTime`, which checks that the lower pile grows with elapsed minutes as in classic.
- [ ] Steps: same as Task 6. Commit `feat(smooth): lava and hourglass`.

### Task 8: Smooth sky

**Files:** `internal/smooth/sky/*`, `smooth.go`

- Same arc, `IsDay`, moon-phase maths and `SetTime` contract as classic. Reimplement them in this package; don't import classic.
- The sun is a dot disc 4 cells wide, with shimmering rays. It glides along the arc at dot resolution.
- The moon reuses the braille phase approach.
- Stars twinkle. Clouds drift at sub-cell speed.
- True-colour tints:
  - the sun is `{240,200,90}` at noon, warming to `{240,140,70}` within 1 h of sunrise or sunset;
  - the moon is `{210,210,225}`;
  - clouds are `{170,170,180}`.
- Everything stays in the upper third, as classic does.
- Tests:
  - `TestSky_SunGlides`: across 10 minutes of shifted time stepped by 1 s, the sun's dot centre moves monotonically with no step greater than 1 dot.
  - `TestSky_DuskWarmer`.
  - `TestSky_StaysInUpperThird`.
  - `TestSky_MoonPhaseMatchesClassicMaths`: reuse classic's known dates (2024-04-08 new, 2024-04-23 full).
- [ ] Steps: same pattern. Commit `feat(smooth): sky`.

### Task 9: Smooth seasons (particles only)

**Files:** `internal/smooth/seasons/*`, `smooth.go`

- Same season-by-shown-month rule, `SetTime` contract and `EnableEvents(birthday *events.MonthDay)` / `Event() (events.Kind, bool)` API as classic seasons.
- In this task, the event art hook calls a local interface `eventArt` that Task 10 satisfies. Until then `EnableEvents` stores the setting and draws no art, and `Event()` still reports the active event via `events.Active`.
- Particles:
  - winter snow is `{220,225,235}`, drifting with sway;
  - spring petals are `{230,160,190}`;
  - summer fireflies or rain are `{230,220,120}` / `{110,150,230}`;
  - autumn leaves are `{210,120,50}` / `{190,150,60}`, tumbling.
  - All of them move at sub-cell speed.
- Tests: `TestSeasons_ParticlesGlide`, `TestSeasons_FollowsShownMonth`, `TestSeasons_EventReported`.
- [ ] Steps: same pattern. Commit `feat(smooth): seasons particles`.

### Task 10: Smooth event art, framework and set A

**Files:** Create `internal/smooth/eventart/eventart.go`, `christmas.go`, `newyear.go`, `diwali.go`, `lunarnewyear.go`, `eid.go`, `holi.go`, `fireworks.go` and `eventart_test.go`. Modify `internal/smooth/seasons` to draw the art.

**Interfaces:**
- `type Region struct{ X0, Y0, W, H int }` with `Touches(b scene.Box) bool`. This is a copy of classic's, including the Task-final fix: empty only when `Y0 == -1 && Y1 == -1`. Don't import classic eventart.
- `type Art interface { Resize(w, h int); Step(dt time.Duration); SetTime(now time.Time, occ events.Occurrence); Bounds() Region; Cell(x, y int, b scene.Box) (rune, scene.RGB) }`.
- `func New(k events.Kind, rng *rand.Rand) Art`, `func Kinds() []events.Kind`.
- Placement and ideas follow the classic arts the user signed off:
  - Christmas is a bottom-right tree.
  - New Year is a top band of fireworks, peaking 23:30–00:30.
  - Diwali is a bottom row of diyas with bursts above them.
  - Lunar New Year is lanterns swaying from the top.
  - Eid is a top-right crescent and lanterns.
  - Holi is soft powder bursts in a right column. Make it busier than classic, with a burst at least every 1.5 s, because classic was noted as sparse.
  - Motion is smooth: sway, rise and fades use true colour.
- Generic tests:
  - copy and adapt classic `eventart_test.go`'s generic tests (`SingleWidth`, `StaysInBounds`, `HidesWhenTouchingBox` with 8 probes, `HidesWhenCalendarAtRowZero`, `TinySizesNoPanic`, `DrawsSomething`, `FitsTypicalMargins`);
  - add `TestAllArt_Glides`, where each art's moving element stays within 1 dot of ideal over 100 steps, for arts with steady motion.
- Specific tests: port `TestNewYear_MoreBurstsAtMidnight`, `TestDiwali_FlamesFlicker`, `TestLanterns_Sway`, `TestEid_MoonWhite`, and add `TestHoli_Busier`.
- [ ] Steps: write the failing tests, run them (FAIL), implement, wire them into smooth seasons, run `go test ./internal/smooth/... ./internal/tui/ && go vet ./...` (PASS; `TestSmooth_*` covers seasons with these events), then commit `feat(smooth): event art framework and first six scenes`.

### Task 11: Smooth event art, set B

**Files:** `internal/smooth/eventart/{sankranti,independence,valentines,onam,vishu,halloween,birthday}.go`, tests.

- These follow the classic placements:
  - Sankranti is kites with strings in a side column.
  - Independence Day is a top-left flag that ripples.
  - Valentine's is rising hearts.
  - Onam is a bottom-left pookalam whose ring count is `ceil((day+1)*rings/days)`, with rings ≥ 11.
  - Vishu is falling konna flowers plus a lamp.
  - Halloween is a pumpkin with bats flitting above it.
  - Birthday is a cake with flickering candles and confetti.
- Tests:
  - port `TestOnam_RingsGrowWithDay` and `TestOnam_NineAndElevenDayYears`;
  - port the robust `TestHalloween_BatsFlap`, which compares per-cell glyphs;
  - port `TestBirthday_CandlesFlicker` and `TestKites_HaveStrings`;
  - add `TestKinds_AllThirteen`.
- [ ] Steps: same pattern. Commit `feat(smooth): remaining event art`.

### Task 12: Smooth harbor, train, container and aquarium

**Files:** `internal/smooth/{harbor,train,container,aquarium}/*`, `smooth.go`

- Harbor:
  - Ships are dot sprites with hull `{150,110,80}` and sail `{220,220,215}`. They sail at classic's speed, which is the faster one already on main, in columns per second derived from classic's interval and step.
  - Water shimmer dots are `{80,120,170}`.
  - The bottom strip hides the same way as classic.
- Train:
  - A locomotive and carriages as dot sprites gliding along the bottom strip.
  - Smoke dots rise and fade.
- Container:
  - The container sprite and bushes in dots.
  - The jump is a parabola at dot resolution, clearing every bush. Port classic runner's collision test idea.
  - It hides when its strip doesn't fit, like `runnerScene`.
- Aquarium:
  - Fish are dot sprites, mirrored when they turn, swimming with gentle vertical sine.
  - Bubbles rise at sub-cell speed.
- Tests: `TestHarbor_ShipGlides`, `TestTrain_Glides`, `TestContainer_JumpClearsBushes`, `TestContainer_Glides`, `TestAquarium_FishGlide`, `TestAquarium_FishTurnMirrored`, plus a hide-when-no-room test per bottom-strip scene.
- If this task's diff passes about 900 lines, the implementer may split it into two commits, but it stays one task.
- [ ] Steps: same pattern. Commit `feat(smooth): harbor, train, container and aquarium`.

### Task 13: README and final verification

**Files:** `README.md`

- [ ] **Step 1:** Update the README.
  - Flags table: `--bg-style` and `--bg-palette`.
  - A *Smooth style* section covering:
    - what smooth is;
    - classic as the default, with classic to be retired later;
    - true-colour detection and how to force it, including for tmux;
    - synchronized output and the Task 4 outcome;
    - config keys `bg-style` and `bg-palette` in the annotated config example.
  - Smoke checklist:
    - `tcal --bg-style smooth --bg rain`;
    - `COLORTERM= tcal --bg-style smooth --bg sky` (ANSI fallback);
    - `--bg-palette truecolor`;
    - a smooth holiday preview via `--bg-time`.
- [ ] **Step 2:** Run `gofmt -l . && goimports -l . && go vet ./... && go test -count=1 ./...`. Expected: nothing listed and all packages pass.
- [ ] **Step 3:** Commit: `docs: smooth backgrounds`.

### Task 14: Preview checkpoint (stop for the user)

- [ ] **Step 1:** Add an env-gated test, `TCAL_PREVIEW_DIR`, that writes preview frames for every smooth background and each of the 13 events. It writes each in TrueColor mode and in ANSI mode, at 120×40, to that directory. Keep it in `internal/tui/preview_test.go`; it skips when the env var is unset.
- [ ] **Step 2:** Give the user the `less -R` command for the frames and live commands (`--bg-style smooth --bg <name>`, `--bg-time` for events).
- [ ] **Step 3:** **Stop and wait for sign-off.** Changes after sign-off get their own review.
- [ ] **Step 4:** After sign-off, commit the preview test: `test: smooth preview frames (env-gated)`.
