package dots

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/v4run/tcal/internal/scene"
)

var (
	red  = scene.RGB{R: 255}
	blue = scene.RGB{B: 255}
)

func TestDot_BitLayout(t *testing.T) {
	var c Canvas
	c.Reset(2, 2)
	c.Dot(0, 0, red)
	c.Dot(0.5, 0, red)
	c.Dot(0, 0.25, red)
	c.Dot(0, 0.75, red)
	c.Dot(0.5, 0.75, red)
	if g := c.Glyph(0, 0); g != '⣋' {
		t.Fatalf("glyph = %q (%#x), want ⣋", g, g)
	}
}

func TestDot_SubCellSteps(t *testing.T) {
	var c Canvas
	c.Reset(3, 3)
	seen := map[rune]bool{}
	for i := 0; i < 2; i++ {
		c.Clear()
		c.Dot(float64(i)*0.5, 0, red)
		seen[c.Glyph(0, 0)] = true
	}
	if len(seen) != 2 {
		t.Fatalf("x steps not distinct: %v", seen)
	}
	seen = map[rune]bool{}
	for i := 0; i < 4; i++ {
		c.Clear()
		c.Dot(0, float64(i)*0.25, red)
		seen[c.Glyph(0, 0)] = true
	}
	if len(seen) != 4 {
		t.Fatalf("y steps not distinct: %v", seen)
	}
	c.Clear()
	c.Dot(1, 1, red)
	if c.Glyph(1, 1) != 0x2801 {
		t.Fatalf("cell step failed")
	}
}

func TestDot_LastColourWins(t *testing.T) {
	var c Canvas
	c.Reset(1, 1)
	c.Dot(0, 0, red)
	c.Dot(0.5, 0.5, blue)
	if got := c.RGB(0, 0); got != blue {
		t.Fatalf("rgb = %v", got)
	}
}

func TestDot_OffCanvasIgnored(t *testing.T) {
	var c Canvas
	c.Dot(0, 0, red) // zero value
	c.Reset(0, 0)
	c.Dot(0, 0, red)
	c.Reset(2, 2)
	for _, p := range [][2]float64{{-0.1, 0}, {0, -0.1}, {-5, -5}, {2, 0}, {0, 2}, {99, 99}} {
		c.Dot(p[0], p[1], red)
	}
	for y := -1; y <= 2; y++ {
		for x := -1; x <= 2; x++ {
			if c.Glyph(x, y) != 0 {
				t.Fatalf("cell %d,%d not empty", x, y)
			}
		}
	}
	c.Reset(-3, -3)
	if w, h := c.Size(); w != 0 || h != 0 {
		t.Fatalf("size %d,%d", w, h)
	}
}

func TestReset_ClearsAndResizes(t *testing.T) {
	var c Canvas
	c.Reset(2, 2)
	c.Dot(0, 0, red)
	c.Reset(4, 3)
	if w, h := c.Size(); w != 4 || h != 3 {
		t.Fatalf("size %d,%d", w, h)
	}
	if c.Glyph(0, 0) != 0 || c.RGB(0, 0) != (scene.RGB{}) {
		t.Fatal("not cleared")
	}
	c.Dot(3, 2, red)
	if c.Glyph(3, 2) == 0 {
		t.Fatal("dot lost")
	}
	c.Clear()
	if c.Glyph(3, 2) != 0 {
		t.Fatal("Clear failed")
	}
}

func TestText_OverwritesDots(t *testing.T) {
	var c Canvas
	c.Reset(4, 1)
	c.Dot(1, 0, red)
	c.Text(1, 0, "ab", blue)
	if c.Glyph(1, 0) != 'a' || c.Glyph(2, 0) != 'b' || c.RGB(1, 0) != blue {
		t.Fatal("text not placed")
	}
	c.Text(3, 0, "xyz", blue) // clipped
	if c.Glyph(3, 0) != 'x' {
		t.Fatal("clip")
	}
	c.Text(-1, 0, "qr", blue)
	if c.Glyph(0, 0) != 'r' {
		t.Fatal("left clip")
	}
	// a dot drawn after text goes back to braille
	c.Dot(1, 0, red)
	if g := c.Glyph(1, 0); g != 0x2801 {
		t.Fatalf("dot over text: %q", g)
	}
}

func TestSprite_ParseAndDraw(t *testing.T) {
	s := ParseSprite("##", "##", "##", "##")
	if w, h := s.Size(); w != 2 || h != 4 {
		t.Fatalf("size %d,%d", w, h)
	}
	var c Canvas
	c.Reset(4, 3)
	c.DrawSprite(s, 1, 1, red)
	if c.Glyph(1, 1) != '⣿' {
		t.Fatalf("glyph %q", c.Glyph(1, 1))
	}
	c.Clear()
	c.DrawSprite(s, 1.5, 1, red)
	l, r := c.Glyph(1, 1), c.Glyph(2, 1)
	if l != 0x2800|0x08|0x10|0x20|0x80 || r != 0x2800|0x01|0x02|0x04|0x40 {
		t.Fatalf("spans: %q %q", l, r)
	}
	s2 := ParseSprite("#.", " #")
	if w, h := s2.Size(); w != 2 || h != 2 {
		t.Fatalf("size %d,%d", w, h)
	}
}

func TestGlyphs_SingleWidth(t *testing.T) {
	var c Canvas
	c.Reset(1, 1)
	for m := 1; m < 256; m++ {
		c.Clear()
		for i, p := range [][2]float64{{0, 0}, {0, .25}, {0, .5}, {.5, 0}, {.5, .25}, {.5, .5}, {0, .75}, {.5, .75}} {
			if m&(1<<i) != 0 {
				c.Dot(p[0], p[1], red)
			}
		}
		g := c.Glyph(0, 0)
		if g == 0 || ansi.StringWidth(string(g)) != 1 {
			t.Fatalf("mask %d glyph %q width %d", m, g, ansi.StringWidth(string(g)))
		}
	}
}
