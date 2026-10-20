# Smooth backgrounds

**Date:** 2026-10-20
**Status:** Approved to build (the user asked for design, planning and implementation in one go)

## Summary

This adds a second style for every live-mode background: **smooth**.

- Smooth scenes are drawn with braille dots, so things move in sub-cell steps.
- They tick at about 30 fps.
- They use true colour where the terminal supports it.
- Today's backgrounds stay exactly as they are, as the **classic** style.
- Classic remains the default. `--bg-style smooth` (or `bg-style = "smooth"` in the config) switches to smooth.
- Classic will be removed in a later, separate change once smooth has proved itself.

## Why

The backgrounds feel steppy. Rain is the worst: every 70 ms a drop moves 0.6–1.05 rows but can only be drawn on a whole row, so it jumps on most frames and stalls on others. The user wants the backgrounds to feel calm and fluid.

A throwaway spike settled the look; it was not committed, and this document is the reference for what it did. The user compared:

- the current rain;
- half-block rain at 30 fps;
- braille rain at 30 fps;
- braille rain with gusting wind.

They preferred **braille** and said the **wind** was "about right".

The user only opens tcal now and then to check the time. CPU cost should stay sensible, but it is not a hard budget.

## Goals

- Every background has a smooth version that glides instead of hopping. This includes the 13 holiday scenes inside `seasons`.
- True colour where supported, falling back to the 16-colour palette elsewhere.
- Classic stays byte-for-byte what it is today, and stays the default.
- Removing classic later is a deletion, not a rewrite.

## Non-goals

- Image protocols (kitty graphics, sixel). They were discussed and deferred to a possible later project.
- Changing print mode.
- New backgrounds or new holiday events.
- Improving classic. Classic is frozen and gets fixes only if something breaks.

## Anti-goals (a failure even if it technically works)

- **A smooth background that makes the calendar harder to read.** All the existing rules hold:
  - dim;
  - never inside the calendar box (plus padding);
  - art hides as a whole rather than being clipped;
  - single-width runes only.

  True colour must not make backgrounds louder.
- **Any change to classic output.** This includes the default run with no config file.
- **Smooth looking worse on a 16-colour terminal than classic does.**
- **Visible tearing or flicker** in common terminals at the higher frame rate.

## How it looks and moves

### Drawing: braille dots

- Every smooth scene draws onto a **dot canvas**. Each terminal cell is a 2×4 grid of braille dots (U+2800–U+28FF, all single-width).
- Objects have float positions. They are plotted at half-column, quarter-row resolution, so motion is in sub-cell steps.
- A braille cell carries **one colour**. When dots of different colours land in one cell, the canvas keeps a deterministic winner. The builder chooses the rule, for example "last drawn wins" or "the colour with most dots wins", and documents it.
- Art should be designed so differently coloured parts mostly sit in different cells.

### Motion: about 30 fps, even pace

- Smooth scenes tick at about 30 fps (≈33 ms).
- Motion must be **even**: an object moving at a steady speed covers equal distances in equal times.
- Positions advance from **elapsed time**, not a fixed per-tick step, so timer jitter doesn't show as stutter. The mechanism is the builder's choice, such as an optional interface that receives the measured `dt`. Classic scenes must not see any change.

### Colour: true colour with fallback

- **Detection.** True colour is on when `COLORTERM` is `truecolor` or `24bit`. Otherwise it's off.
  - A new flag forces the mode: `--bg-palette auto|truecolor|ansi`, default `auto`. *(Claude's call: many terminals support true colour without announcing it, and tmux can hide it.)*
  - It is also a config key, like every flag.
- **Smooth scenes describe colours as RGB.**
  - In true-colour mode they are emitted as 24-bit SGR.
  - In ANSI mode each colour maps to the nearest of the existing 8 palette colours (`scene.Color`), so the output looks like today's tinting.
- **Staying dim in true colour** *(Claude's call)*. True colour ignores the terminal theme, so a fixed muted colour could be loud on a light background. Instead:
  - Each smooth scene's colour is **blended toward the terminal's background colour** by a dimming factor.
  - The background colour is detected with the standard query, via lipgloss/termenv's dark-background detection or OSC 11. When unknown, it defaults to dark.
  - The faint attribute is still applied, as today.
  - An automated test checks that every colour a smooth scene emits, after blending, has lower contrast against both a dark and a light background than the default foreground text has. The exact contrast measure is the builder's choice, for example WCAG relative luminance contrast with a stated cap.
- **`--bg-color=false`** keeps its meaning in both styles: no tint, dim foreground glyphs.

### Tearing

- Frames should be shown atomically. Use **synchronized output** (DEC private mode 2026, begin/end around each frame) where it can be done with the Bubble Tea version in use (v1.3.x). Terminals that don't support it ignore it.
- If Bubble Tea v1 makes that impractical, the builder documents why and what was done instead. Upgrading Bubble Tea is a valid option only if it is contained.

## The two styles side by side

*(The user chose this: approach "A", freeze classic and build smooth alongside.)*

- **Classic** is the code as it is today, unchanged. The packages stay where they are.
- **Smooth** is new code in new packages. *(Claude's call on layout, adjustable by the builder:)*
  - `internal/dots`: the dot canvas;
  - `internal/smooth/<scene>`: smooth scenes;
  - `internal/smooth/eventart`: smooth holiday art;
  - colour and true-colour helpers sit beside the scene package or in their own small package.
- **Registry.**
  - Each background name has a classic and a smooth constructor.
  - `--bg-style classic|smooth` picks which set is used. The default is `classic`, and it is also a config key.
  - Names, `--bg random`, the random pool, `--bg-time`, `--events` and the event-day rule work identically in both styles.
- **Step-by-step scenes keep their look.** `life`, `langton`, `automaton` and `ants` step one cell at a time by nature. Their smooth entries reuse the classic implementations unchanged, at the same pace. *(The user agreed these keep their look.)* They may get true-colour tints if that is cheap, but it is optional.
- **Removing classic later** means deleting the classic scene packages, the classic registry entries and the `--bg-style` flag and config key.

## Scenes to build in smooth style

| Background | Smooth version |
|---|---|
| rain | Braille drops, two-dot streaks tilted along the wind, gusting wind that eases toward a new target every 2–7 s (calm about 30% of the time, otherwise up to about ±1.2 columns per row fallen). Drops wrap horizontally. Ground line and splashes as today, in dots. *This is what the user saw and liked in the spike.* |
| starfield | Stars streaming at sub-cell speed, brightness by depth (a true-colour fade). |
| seasons | Snow, leaves, petals and rain in dots with gentle drift. The holiday art is drawn by `internal/smooth/eventart`. |
| flow | Particles following the flow field at sub-cell resolution, with trails fading in true colour. |
| lava | Blobs as smooth dot fields, colour graded by heat. |
| sky | Sun disc and rays, moon phases (already braille), stars and clouds, gliding along the arc. Dawn and dusk tints in true colour. |
| hourglass | Sand grains as dots falling and piling. |
| harbor | Ships redrawn as dot pictures sailing smoothly, water shimmering. |
| train | Train and carriages as a dot picture gliding along, smoke drifting. |
| container runner | Container, bushes and jump arc in dots. The jump is a smooth parabola. |
| aquarium | Fish as dot pictures swimming smoothly, bubbles rising. |
| life, langton, automaton, ants | Reuse classic (see above). |

**The 13 holiday scenes** (Christmas, New Year, Onam, Halloween, Diwali, Vishu, Holi, Sankranti, Independence Day, Eid, Lunar New Year, Valentine's, Birthday):

- Each is redrawn in dots under the same rules as today: one bounds region, hides when it touches the calendar, a restrained palette.
- The ideas and placements signed off in the classic version carry over.
- Each must still be **recognisable at a glance**. Where braille can't carry a detail, such as text-like features, the builder may mix in ordinary single-width characters.

## User checkpoint

Before the final review there is **one preview checkpoint**: frames, or a way to run each smooth background and holiday scene. The user signs off on the look, as they did for the classic holiday art. Art changes after sign-off get their own review.

## Testing expectations

**Every smooth background and holiday scene passes the existing shared tests:**

- the calendar box is untouched;
- colours are well-formed and don't leak;
- single-width runes only;
- the per-frame timing budget, re-checked for 30 fps at 200×60.

**Added tests:**

- **Smoothness.** A steadily moving object's plotted dot position advances by consistent amounts per unit of time, with no stalls and no double steps beyond a tolerance. This is the property the spike fixed.
- **Classic unchanged.**
  - Classic output for a fixed seed and size is identical before and after this work, through a golden test or a recorded frame hash.
  - A default run with no config uses classic.
- **Style selection.**
  - `--bg-style` and the config key work, with flag-beats-config precedence.
  - An invalid value is an error naming the value, as for every other flag.
- **Palette.**
  - `--bg-palette` and `COLORTERM` detection select the right mode.
  - ANSI-mode output only uses the 16-colour SGR codes.
  - True-colour output uses 24-bit SGR.
- **Dimness.** The contrast check described above, against both dark and light backgrounds.
- **Event art in smooth.** The same window and priority behaviour as classic, through the smooth `seasons` background.

## Left to the builder

- Package layout details, the dot canvas API, and how elapsed time reaches scenes.
- The colour blending factor and contrast measure, within the anti-goals.
- Each smooth scene's art and motion details, within the table above, shown to the user at the checkpoint.
- Whether synchronized output needs a Bubble Tea upgrade.
- README updates:
  - `--bg-style` and `--bg-palette`;
  - a short note that classic will be retired later;
  - an updated smoke checklist.

## Builder's questions, settled *(Claude's calls)*

1. **Shared interfaces.**
   - Smooth may add **optional** interfaces to `internal/scene`, for example an RGB colour interface and an elapsed-time ticker.
   - It may also add a second colour path in `tui/background.go`.
   - The classic scene *packages* are not edited.
   - The "classic unchanged" golden tests guard classic *output*.
   - Removing classic later also removes any interface that only classic used.
2. **Detecting the background colour.**
   - Detect it once, before the Bubble Tea program starts, using lipgloss/termenv's `HasDarkBackground()`. Cache the result. Don't run an OSC 11 query of our own under Bubble Tea.
   - The contrast test assumes white text on a black background for dark themes, and black text on a white background for light themes.
3. **Faint.** Colours must be dim enough **without** faint. Faint is still emitted as extra softening, but it is not counted.
4. **Cost.**
   - The calendar frame changes at most once a second, so it may be cached between clock ticks so that only the overlay is recomputed at 30 fps.
   - The existing shared timing test allows 0.2 ms per frame for a scene. A smooth scene gets the same budget at 200×60.
   - Optimising the overlay is in scope if 30 fps needs it.
5. **The 16-colour fallback.**
   - Like classic, the fallback keeps **hue**, not brightness.
   - "Not worse than classic" means each scene is at least as recognisable in ANSI mode as its classic version. That is checked at the preview checkpoint, which shows ANSI mode as well.
6. **Synchronized output.**
   - First check whether Bubble Tea v1.3.10 already emits mode 2026.
   - If it doesn't, bracket each renderer flush with an output-writer wrapper (`tea.WithOutput`).
   - If neither works, this becomes a contained task: document the outcome, or upgrade Bubble Tea within this branch only if the change stays small.
7. **Colour winner.** One rule for the whole canvas: the last dot drawn sets the cell's colour. Scenes control what wins through their draw order.
8. **No colour wins.** `NO_COLOR` / `--no-color` beat `--bg-palette truecolor`. With no colour, there is no tint at all, the same as today.
9. **The checkpoint is a stop.**
   - Execution stops there and hands over to the user.
   - Previews are frame dumps from an env-gated test helper, plus live commands using `--bg-style smooth --bg-time …`.
   - There is no dev-only flag in `cmd/tcal`.
10. **Time source.**
    - `dt` comes from the tick message's timestamp, clamped to a maximum of 100 ms, so a suspended terminal doesn't make scenes jump.
    - Tests pass `dt` directly.
