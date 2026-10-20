package tui

import (
	"math/rand/v2"

	"github.com/v4run/tcal/internal/scene"
)

// Background styles. Classic draws one glyph per cell in the 16-colour
// palette; smooth draws braille dots in true colour at about 30 fps.
const (
	StyleClassic = "classic"
	StyleSmooth  = "smooth"
)

// Styles returns the background style names, default first.
func Styles() []string { return []string{StyleClassic, StyleSmooth} }

// keepFactor is how much of a true-colour scene's colour survives blending
// toward the terminal background. The worst case is pure white on black:
// at 0.45 it becomes {115,115,115}, a WCAG contrast of 4.43 against black
// (the limit, 4.5, is crossed at 0.46). Pure black on white becomes
// {140,140,140}, a contrast of 3.36.
const keepFactor = 0.45

// smoothBackgrounds holds the smooth style's constructors; it is the only
// place they are listed. A name missing here falls back to its classic
// constructor, so every name in Backgrounds exists in both styles.
var smoothBackgrounds = map[string]func(w, h int, rng *rand.Rand, o BackgroundOptions) scene.Scene{
	"life":      backgrounds["life"],
	"langton":   backgrounds["langton"],
	"automaton": backgrounds["automaton"],
	"ants":      backgrounds["ants"],
}
