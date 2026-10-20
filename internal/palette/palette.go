// Package palette holds colour-mode selection and colour maths for the
// smooth background renderer.
package palette

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/v4run/tcal/internal/scene"
)

// Mode is the colour depth used to draw backgrounds.
type Mode int

const (
	ANSI Mode = iota
	TrueColor
)

// Dark and Light are the reference backgrounds.
var (
	Dark  = scene.RGB{R: 0, G: 0, B: 0}
	Light = scene.RGB{R: 255, G: 255, B: 255}
)

// Parse reads auto|truecolor|ansi (case-insensitive). The bool is true for
// auto, in which case the mode is detected from the environment.
func Parse(s string) (Mode, bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "auto":
		return Detect(os.Getenv), true, nil
	case "truecolor":
		return TrueColor, false, nil
	case "ansi":
		return ANSI, false, nil
	}
	return ANSI, false, fmt.Errorf("invalid colour mode %q (want auto, truecolor or ansi)", s)
}

// Detect returns TrueColor when COLORTERM advertises it, otherwise ANSI.
func Detect(getenv func(string) string) Mode {
	switch strings.ToLower(getenv("COLORTERM")) {
	case "truecolor", "24bit":
		return TrueColor
	}
	return ANSI
}

// Blend mixes c over bg linearly: keep=1 gives c, keep=0 gives bg.
func Blend(c, bg scene.RGB, keep float64) scene.RGB {
	mix := func(a, b uint8) uint8 {
		v := float64(a)*keep + float64(b)*(1-keep)
		return uint8(math.Round(math.Max(0, math.Min(255, v))))
	}
	return scene.RGB{R: mix(c.R, bg.R), G: mix(c.G, bg.G), B: mix(c.B, bg.B)}
}

func luminance(c scene.RGB) float64 {
	lin := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

// Contrast is the WCAG 2 contrast ratio between a and b, from 1 to 21.
func Contrast(a, b scene.RGB) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// Nearest maps c to the closest hue among the ANSI colours. Low-saturation
// colours map to White (bright) or Default.
func Nearest(c scene.RGB) scene.Color {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	mx := math.Max(r, math.Max(g, b))
	mn := math.Min(r, math.Min(g, b))
	d := mx - mn
	var s float64
	if mx > 0 {
		s = d / mx
	}
	if s < 0.25 {
		if mx > 0.85 {
			return scene.White
		}
		return scene.Default
	}
	var h float64
	switch mx {
	case r:
		h = math.Mod((g-b)/d, 6)
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	switch {
	case h < 20 || h >= 340:
		return scene.Red
	case h < 70:
		return scene.Yellow
	case h < 160:
		return scene.Green
	case h < 200:
		return scene.Cyan
	case h < 260:
		return scene.Blue
	default:
		return scene.Magenta
	}
}
