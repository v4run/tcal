# Thin look

## Why

The classic look is clean; this explores a second look for the clock and
calendar. Agreed in a browser mockup session (option B "thin type", dense
spacing).

## What

`--look classic|thin` (config key `look`), default `classic`.

- Clock: `HH:MM` in heavy box-drawing digits, 3 rows × 3 cols each; a bar
  under it fills across the minute (`━` filled, `─` empty) with `:SS`
  beside it; then a blank line and the date as "Wednesday · 21 October".
  `--clock-style` is ignored. Without `--with-clock` the one-line header is
  kept.
- Month: day numbers in 3-row × 2-col digits, cells 7 cols wide (49 per
  month), weeks stacked with no gaps (22 rows a month). Past days faint,
  weekends ANSI 8, today per `--highlight`: `color` accent, `combined`
  accent + bold, `reverse` reversed cell, `bracket` corner brackets. With
  color off any highlight but `none` becomes brackets.
- Several months sit side by side (horizontal) or 3 per row (grid) when
  they fit. If the thin block is wider or taller than the screen (or
  `--width` in print mode), the whole frame falls back to classic.
- Print mode supports the thin look the same way.
- Backgrounds never touch the thin calendar's padded box.

## Side fix

`--months 1` now shows the anchor month, not the month before it, in both
looks.
