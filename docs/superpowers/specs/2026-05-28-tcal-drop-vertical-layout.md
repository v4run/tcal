# Drop vertical layout

**Date:** 2026-05-28
**Status:** Accepted

## Summary

Remove `LayoutVertical` from tcal. The only remaining multi-month layouts are `LayoutHorizontal` (default) and `LayoutGrid`. The TUI layout-switch key `2` is remapped from vertical to grid, and key `3` is unbound.

## Rationale

Three layouts were originally specified, but in practice the vertical stack adds little over horizontal — both show three months in sequence; horizontal reads better in modern wide terminals and reuses screen space the user already has. Carrying the extra branch in `RenderLayout`, the dedicated `joinVertical` helper, a golden file, a model key binding, and three rows of doc/README isn't justified by the value the layout adds.

This continues the trend of collapsing layout variants started in `9efedf1` (drop `LayoutFocus`, which was visually identical to `LayoutHorizontal`).

## Changes

| File | Change |
|---|---|
| `internal/render/types.go` | Remove `LayoutVertical` constant. |
| `internal/render/layouts.go` | Remove `case LayoutVertical` from `RenderLayout`; remove `joinVertical` helper; fall through to `joinHorizontal` in the `default` branch. |
| `internal/render/frame.go` | Update comments and `minWidthFor` default branch (`// vertical` → `// horizontal`). |
| `internal/render/layouts_test.go` | Delete `TestLayout_Vertical_3Months`. |
| `internal/render/testdata/layout_vertical_3.golden` | Delete. |
| `internal/tui/model.go` | Remap key `2` to `LayoutGrid`; drop key `3` binding. |
| `internal/tui/model_test.go` | Update layout-keys table to `{'1', Horizontal}, {'2', Grid}`. |
| `cmd/tcal/main.go` | Drop `"vertical", "v"` from `parseLayout`; update flag description and error message to `horizontal\|grid`. |
| `README.md` | Drop `--layout vertical` from quick-start; update keys table (`1 / 2`) and flags table. |

## Out of scope

- Focus mode and larger-font day numbers were discussed and deferred — the brainstorming explored DECDHL real double-height text and half-block ASCII-art glyph fonts; the user chose to step back and not introduce focus mode at this time.
- Historical spec/plan docs under `docs/superpowers/{specs,plans}/2026-05-27-tcal-*.md` are not rewritten; they remain as historical artifacts of the original three-layout design.

## Testing

- `go test ./...` — existing horizontal and grid tests still pass; the deleted vertical test no longer runs.
- `gofmt -l .` — empty.
- `go vet ./...` — clean.

## Migration

No external migration. Users with shell history or scripts passing `--layout vertical` will get the existing "unknown --layout" error and a clear expected-values message.
