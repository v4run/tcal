package eventart

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

const (
	kiteMin  = 2
	kiteMax  = 4
	kiteColW = 18 // column width
	kiteMinH = 12
	kiteMaxH = 20

	kiteBobX    = 0.6 // columns a kite drifts either side of home
	kiteBobY    = 0.4 // rows it rises and dips
	kiteDriftX  = 5.0 // seconds per sideways drift
	kiteDriftY  = 3.7 // seconds per rise and dip
	kiteTailN   = 8   // dots in a tail
	kiteTailAmp = 0.4 // columns the tail's end swings
	kiteTailHz  = 1.2 // waves per second running down a tail
)

var (
	kiteCols = [...]scene.RGB{
		{R: 205, G: 80, B: 70},   // red
		{R: 230, G: 150, B: 60},  // saffron
		{R: 85, G: 145, B: 205},  // blue
		{R: 100, G: 170, B: 95},  // green
		{R: 200, G: 100, B: 160}, // pink
	}
	kiteString = scene.RGB{R: 175, G: 170, B: 160} // cotton manja

	// kiteShape is a diamond kite, widest a third of the way down.
	kiteShape = parseShape(
		"..##..",
		".####.",
		"######",
		"######",
		".####.",
		".####.",
		"..##..",
		"..##..",
		"..##..",
		"...#..",
	)
	kiteDW, kiteDH = kiteShape.size()
)

// sankranti is a few diamond kites bobbing in the upper half of a column
// on the left, each with a wiggling tail and a string running down to one
// flyer at the column's bottom edge.
type sankranti struct {
	w, h  int
	t     float64 // seconds
	n     int
	home  [kiteMax]int // home row of each kite, scaled to the column in render
	phase [kiteMax][3]float64
	cols  [kiteMax]scene.RGB
	f     frame
}

func init() {
	arts[events.Sankranti] = func(rng *rand.Rand) Art {
		s := &sankranti{n: kiteMin + rng.IntN(kiteMax-kiteMin+1)}
		for i, p := range rng.Perm(len(kiteCols))[:kiteMax] {
			s.cols[i] = kiteCols[p]
		}
		for i := range s.home {
			s.home[i] = rng.IntN(1 << 16)
			for j := range s.phase[i] {
				s.phase[i][j] = rng.Float64() * 2 * math.Pi
			}
		}
		return s
	}
}

func (s *sankranti) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.f.reset(s.Bounds())
	s.render()
}

func (s *sankranti) Step(dt time.Duration) {
	s.t += max(dt, 0).Seconds()
	s.render()
}

// pos is kite i's top-left corner now, in region cells: spread across the
// column, in its upper half, drifting gently about home.
func (s *sankranti) pos(i int) (x, y float64) {
	w, h := float64(s.f.r.W), s.f.r.H
	cx := float64(kiteDW)/4 + 1
	if s.n > 1 {
		cx += float64(i) * (w - 2*cx) / float64(s.n-1)
	}
	span := max(1, h/2-4)
	x = cx - float64(kiteDW)/4 + kiteBobX*math.Sin(2*math.Pi*s.t/kiteDriftX+s.phase[i][0])
	y = float64(s.home[i]%span+1) + kiteBobY*math.Sin(2*math.Pi*s.t/kiteDriftY+s.phase[i][1])
	return x, y
}

// tip is the bottom tip of kite i, where its tail and string hang from.
func (s *sankranti) tip(i int) (x, y float64) {
	x, y = s.pos(i)
	return x + float64(kiteDW)/4, y + float64(kiteDH-1)/4
}

// flyer is where every string meets, at the bottom of the column.
func (s *sankranti) flyer() (x, y float64) {
	return float64(s.f.r.W) / 2, float64(s.f.r.H) - 0.125
}

func (s *sankranti) render() {
	c := &s.f.c
	c.Clear()
	if s.f.r.W == 0 {
		return
	}
	fx, fy := s.flyer()
	for i := range s.n {
		tx, ty := s.tip(i)
		line(c, tx, ty, fx, fy, kiteString)
	}
	for i := range s.n {
		tx, ty := s.tip(i)
		// The tail hangs from the tip, a wave running down it.
		for j := 1; j <= kiteTailN; j++ {
			f := float64(j) / kiteTailN
			off := kiteTailAmp * f * math.Sin(2*math.Pi*(f-kiteTailHz*s.t)+s.phase[i][2])
			c.Dot(tx+off, ty+float64(j)/4, s.cols[i])
		}
		x, y := s.pos(i)
		kiteShape.draw(c, x, y, s.cols[i], nil)
	}
}

func (s *sankranti) SetTime(time.Time, events.Occurrence) {}

// Bounds is a column down the left side, from near the top.
func (s *sankranti) Bounds() Region {
	return edgeRegion(s.w, s.h, kiteColW, columnHeight(s.h, kiteMinH, kiteMaxH), false, false)
}

func (s *sankranti) Cell(x, y int, b scene.Box) (rune, scene.RGB) { return s.f.cell(x, y, b) }
