package tui

import (
	"math/rand/v2"

	"github.com/v4run/tcal/internal/scene"
	"github.com/v4run/tcal/internal/smooth/aquarium"
	"github.com/v4run/tcal/internal/smooth/container"
	"github.com/v4run/tcal/internal/smooth/flow"
	"github.com/v4run/tcal/internal/smooth/harbor"
	"github.com/v4run/tcal/internal/smooth/hourglass"
	"github.com/v4run/tcal/internal/smooth/lava"
	"github.com/v4run/tcal/internal/smooth/rain"
	"github.com/v4run/tcal/internal/smooth/seasons"
	"github.com/v4run/tcal/internal/smooth/sky"
	"github.com/v4run/tcal/internal/smooth/starfield"
	"github.com/v4run/tcal/internal/smooth/train"
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
// toward the terminal background. True-colour cells are not also drawn
// faint, so the blend alone keeps them dim. The worst case is pure white on
// black: at 0.75 it becomes {191,191,191}, a WCAG contrast of 11.4 against
// black, well under the 21 of the calendar's own text. Pure black on white
// becomes {64,64,64}, a contrast of 10.4.
const keepFactor = 0.75

// dimLimit is the highest WCAG contrast a blended true-colour background
// cell may have against the terminal background.
const dimLimit = 12.0

// smoothBackgrounds holds the smooth style's constructors; it is the only
// place they are listed. A name missing here falls back to its classic
// constructor, so every name in Backgrounds exists in both styles.
var smoothBackgrounds = map[string]func(w, h int, rng *rand.Rand, o BackgroundOptions) scene.Scene{
	"life":      backgrounds["life"],
	"langton":   backgrounds["langton"],
	"automaton": backgrounds["automaton"],
	"ants":      backgrounds["ants"],
	"rain": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene {
		return rain.New(w, h, rng)
	},
	"starfield": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene {
		return starfield.New(w, h, rng)
	},
	"flow": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene {
		return flow.New(w, h, rng)
	},
	"lava": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene {
		return lava.New(w, h, rng)
	},
	"hourglass": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene {
		return hourglass.New(w, h, rng)
	},
	"sky": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene {
		return sky.New(w, h, rng)
	},
	"seasons": func(w, h int, rng *rand.Rand, o BackgroundOptions) scene.Scene {
		s := seasons.New(w, h, rng)
		if o.Events {
			s.EnableEvents(o.Birthday)
		}
		return s
	},
	"harbor": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene {
		return harbor.New(w, h, rng)
	},
	"train": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene {
		return train.New(w, h, rng)
	},
	"container": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene {
		return container.New(w, h, rng)
	},
	"aquarium": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene {
		return aquarium.New(w, h, rng)
	},
}
