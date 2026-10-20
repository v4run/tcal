// Package harbor is the smooth harbour scene: container ships drawn in
// braille dots glide across a shimmering water line along the bottom of the
// screen, now and then.
package harbor

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/dots"
	"github.com/v4run/tcal/internal/scene"
)

// Height is the number of rows the strip occupies at the bottom of the
// screen, the same as classic: the water line plus the tallest ship.
const Height = 6

// MinWidth is the narrowest screen the scene will draw on.
const MinWidth = 40

// Classic timing, converted from its 50 ms ticks.
const (
	step       = 33 * time.Millisecond
	classicTk  = 50 * time.Millisecond
	firstWait  = 45 * classicTk   // max wait before the first ship
	minWait    = 450 * classicTk  // minimum empty sea between ships
	waitSpread = 1350 * classicTk // extra random empty sea
	minPeriod  = 2                // fastest ship: classic ticks per column
	periodSpan = 3                // periods are minPeriod..minPeriod+periodSpan-1
	minGap     = 12               // minimum columns between a ship and its companion
	gapSpread  = 30               // extra random columns between companions
)

// waterSpeed is the drift of the water pattern in columns per second
// (classic: one column every 15 ticks).
var waterSpeed = 1 / (15 * classicTk.Seconds())

// Dot rows of the strip (Height*4 = 24): the bridge stands on rows 0-11,
// cargo on rows 4-11, the hull fills rows 12-19 and the water rows 20-23.
const (
	deckRow   = 12
	hullBot   = 19
	cargoTop  = 4
	boxH      = 4  // dot rows in a small container
	bridgeX   = 4  // dot column of the bridge's left wall
	bridgeW   = 6  // dot width of the bridge
	cargoX    = 13 // dot column where cargo starts
	cargoGap  = 2  // dots between containers
	bowDeck   = 10 // dots of deck ahead of the cargo
	bigW      = 29 // dot width of the big container
	waterLine = 21 // dot row of the ripple's crest
)

// smallW are the dot widths of small containers (classic: 4 or 5 columns).
// Odd widths put a rib on both end walls.
var smallW = [2]int{9, 11}

var (
	hullCol  = scene.RGB{R: 150, G: 110, B: 80}
	sailCol  = scene.RGB{R: 220, G: 220, B: 215}
	waterCol = scene.RGB{R: 80, G: 120, B: 170}
	bigCol   = scene.RGB{R: 175, G: 80, B: 65} // the container scene's container
	// boxCols are muted container paints.
	boxCols = []scene.RGB{
		{R: 170, G: 85, B: 70},
		{R: 190, G: 150, B: 80},
		{R: 90, G: 125, B: 160},
		{R: 105, G: 145, B: 100},
		{R: 110, G: 150, B: 150},
		{R: 145, G: 105, B: 135},
	}
)

var (
	_ scene.Scene       = (*Scene)(nil)
	_ scene.Stepper     = (*Scene)(nil)
	_ scene.TrueColored = (*Scene)(nil)
)

// pix is one dot of a sprite, offset from its top-left corner.
type pix struct {
	dx, dy int
	col    scene.RGB
}

type ship struct {
	art   []pix   // facing its direction of travel
	w     int     // width in dots
	x     float64 // screen column of the left edge
	dir   int     // +1 sails right, -1 sails left
	speed float64 // columns per second
}

func (sh *ship) width() float64 { return float64(sh.w) / 2 }

// Scene is the harbour.
type Scene struct {
	rng   *rand.Rand
	w, h  int
	t     float64 // seconds elapsed, for the water
	ships []*ship
	wait  float64 // seconds until the next ship on an empty sea
	// pending is a companion waiting for room behind ships[0].
	pending    *ship
	pendingGap float64
	canvas     dots.Canvas // the strip, Height rows
}

// New returns a w×h scene that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
	s.wait = rng.Float64() * firstWait.Seconds()
	s.Resize(w, h)
	return s
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return step }

// Tick advances one interval.
func (s *Scene) Tick() { s.Step(step) }

// Resize changes the screen size, dropping ships now off screen.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.cull()
	s.canvas.Reset(s.w, Height)
	s.render()
}

// Step advances the ships and the water by dt.
func (s *Scene) Step(dt time.Duration) {
	sec := max(dt, 0).Seconds()
	s.t += sec
	for _, sh := range s.ships {
		sh.x += float64(sh.dir) * sh.speed * sec
	}
	s.cull()
	if s.pending != nil && (len(s.ships) == 0 || s.roomBehind(s.ships[0]) >= s.pendingGap) {
		s.ships = append(s.ships, s.pending)
		s.pending = nil
	}
	if len(s.ships) == 0 && s.pending == nil {
		if s.wait -= sec; s.wait <= 0 {
			s.launch()
			s.wait = minWait.Seconds() + s.rng.Float64()*waitSpread.Seconds()
		}
	}
	s.render()
}

// Glyph returns the rune at screen (x, y), or 0 for empty. The scene hides
// whole when its strip would touch the calendar box b.
func (s *Scene) Glyph(x, y int, b scene.Box) rune {
	if s.w < MinWidth || b.Y1 >= s.h-Height {
		return 0
	}
	return s.canvas.Glyph(x, y-(s.h-Height))
}

// RGB returns the colour of the glyph at (x, y).
func (s *Scene) RGB(x, y int, _ scene.Box) scene.RGB {
	return s.canvas.RGB(x, y-(s.h-Height))
}

// speed converts a classic period (ticks per column) to columns per second.
func speed(period int) float64 { return 1 / (float64(period) * classicTk.Seconds()) }

// launch sends a ship onto the empty sea and, sometimes, queues a companion
// no faster than it to follow in the same direction.
func (s *Scene) launch() {
	dir := 1
	if s.rng.IntN(2) == 0 {
		dir = -1
	}
	period := minPeriod + s.rng.IntN(periodSpan)
	s.ships = append(s.ships, s.newShip(dir, speed(period)))
	if s.rng.IntN(4) == 0 {
		s.pending = s.newShip(dir, speed(period+s.rng.IntN(2)))
		s.pendingGap = float64(minGap + s.rng.IntN(gapSpread))
	}
}

// newShip builds a ship just off the edge it enters from.
func (s *Scene) newShip(dir int, v float64) *ship {
	art, w := s.buildArt(dir)
	sh := &ship{art: art, w: w, dir: dir, speed: v}
	if dir > 0 {
		sh.x = -sh.width()
	} else {
		sh.x = float64(s.w)
	}
	return sh
}

// roomBehind is the number of clear columns between the edge sh entered
// from and its stern.
func (s *Scene) roomBehind(sh *ship) float64 {
	if sh.dir > 0 {
		return sh.x
	}
	return float64(s.w) - (sh.x + sh.width())
}

// cull drops ships that have sailed fully off screen.
func (s *Scene) cull() {
	kept := s.ships[:0]
	for _, sh := range s.ships {
		if !((sh.dir > 0 && sh.x >= float64(s.w)) || (sh.dir < 0 && sh.x+sh.width() <= 0)) {
			kept = append(kept, sh)
		}
	}
	s.ships = kept
}

// buildArt draws a random right-facing ship, mirrored for dir < 0: a white
// bridge at the stern, containers on deck and a hull with a raked bow.
// Neighbouring parts are at least one dot apart, so no cell mixes colours.
func (s *Scene) buildArt(dir int) ([]pix, int) {
	var art []pix
	put := func(x, y int, c scene.RGB) { art = append(art, pix{x, y, c}) }
	box := func(x0, y0, w, h int, c scene.RGB) {
		for y := y0; y < y0+h; y++ {
			for x := x0; x < x0+w; x++ {
				edge := y == y0 || y == y0+h-1 || x == x0 || x == x0+w-1
				if edge || (x-x0)%2 == 0 {
					put(x, y, c)
				}
			}
		}
	}

	// Bridge: a roof, a row of windows and a solid tower down to the deck.
	for x := bridgeX - 1; x <= bridgeX+bridgeW; x++ {
		put(x, 0, sailCol)
	}
	for y := 1; y < deckRow; y++ {
		for x := bridgeX; x < bridgeX+bridgeW; x++ {
			if (y == 2 || y == 3) && (x-bridgeX)%2 == 1 {
				continue
			}
			put(x, y, sailCol)
		}
	}

	x := cargoX
	paint := s.painter()
	if s.rng.IntN(4) == 0 {
		box(x, cargoTop, bigW, deckRow-cargoTop, bigCol)
		x += bigW + cargoGap
	} else {
		for n := 1 + s.rng.IntN(3); n > 0; {
			w := smallW[s.rng.IntN(2)]
			box(x, deckRow-boxH, w, boxH, paint())
			n--
			if n > 0 && s.rng.IntN(2) == 0 {
				box(x, deckRow-2*boxH, w, boxH, paint())
				n--
			}
			x += w + cargoGap
		}
	}
	width := x - cargoGap + bowDeck

	// Hull in outline: a two-dot gunwale, the stern wall, a raked bow and
	// a two-dot keel.
	for y := deckRow; y <= hullBot; y++ {
		i := y - deckRow
		left := max(0, i-5)
		right := width - 1 - i*3/2
		for x := left; x <= right; x++ {
			if i < 2 || y > hullBot-2 || x < left+2 || x > right-2 {
				put(x, y, hullCol)
			}
		}
	}

	if dir < 0 {
		for i := range art {
			art[i].dx = width - 1 - art[i].dx
		}
	}
	return art, width
}

// painter returns a source of container paints in which neighbours differ.
func (s *Scene) painter() func() scene.RGB {
	prev := -1
	return func() scene.RGB {
		i := s.rng.IntN(len(boxCols))
		if i == prev {
			i = (i + 1) % len(boxCols)
		}
		prev = i
		return boxCols[i]
	}
}

// render redraws the strip: water first, ships over it.
func (s *Scene) render() {
	s.canvas.Clear()
	if s.w == 0 {
		return
	}
	s.water()
	for _, sh := range s.ships {
		for _, p := range sh.art {
			s.canvas.Dot(sh.x+float64(p.dx)/2, float64(p.dy)/4, p.col)
		}
	}
}

// water draws a rippling line that drifts as a whole, with foam glints that
// come and go.
func (s *Scene) water() {
	shift := int(math.Floor(s.t * waterSpeed * 2))
	glint := int(s.t / 0.4)
	for dx := range s.w * 2 {
		wx := dx + shift
		y := waterLine
		if (wx/3)%2 != 0 {
			y++
		}
		s.canvas.Dot(float64(dx)/2, float64(y)/4, waterCol)
		h := hash(wx, glint)
		switch {
		case h%23 == 0:
			s.canvas.Dot(float64(dx)/2, float64(waterLine-1)/4, waterCol)
		case h%31 == 0:
			s.canvas.Dot(float64(dx)/2, float64(waterLine+2)/4, waterCol)
		}
	}
}

// hash mixes two integers into a pseudo-random value.
func hash(a, b int) uint32 {
	k := uint32(a)*0x9E3779B1 ^ uint32(b)*0x85EBCA77
	k ^= k >> 15
	k *= 0x2C1B3C6D
	k ^= k >> 12
	return k
}
