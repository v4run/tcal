package dino

import (
	"math/rand/v2"
	"slices"
	"testing"
)

func TestDinoClearsEveryBushShape(t *testing.T) {
	for shape := range bushShapes {
		s := New(80, 20, rand.New(rand.NewPCG(1, 2)))
		s.nextX = 1 << 30 // no random spawns
		s.bushes = []bush{{x: 40, shape: shape}}
		cleared := false
		for i := 0; i < 80; i++ {
			s.Tick()
			if s.collides() {
				t.Fatalf("shape %d: dino hit bush at tick %d", shape, i)
			}
			if len(s.bushes) == 0 {
				cleared = true
				break
			}
		}
		if !cleared {
			t.Fatalf("shape %d: bush never scrolled off", shape)
		}
	}
}

func TestDinoNeverCollidesOverLongRun(t *testing.T) {
	for seed := uint64(0); seed < 20; seed++ {
		s := New(120, 30, rand.New(rand.NewPCG(seed, seed)))
		jumps := 0
		for i := 0; i < 5000; i++ {
			s.Tick()
			if s.jump == 0 {
				jumps++
			}
			if s.collides() {
				t.Fatalf("seed %d: collision at tick %d", seed, i)
			}
		}
		if jumps == 0 {
			t.Fatalf("seed %d: dino never jumped", seed)
		}
	}
}

func TestMinGapLeavesRoomToLand(t *testing.T) {
	if need := len(jumpArc) + jumpLead; minGap < need {
		t.Fatalf("minGap %d < jump length %d", minGap, need)
	}
}

func TestGlyph_OnlyInBottomStrip(t *testing.T) {
	const w, h = 60, 25
	s := New(w, h, rand.New(rand.NewPCG(3, 4)))
	for i := 0; i < 200; i++ {
		s.Tick()
		for y := 0; y < h-Height; y++ {
			for x := 0; x < w; x++ {
				if r := s.Glyph(x, y); r != 0 {
					t.Fatalf("glyph %q above strip at (%d,%d)", r, x, y)
				}
			}
		}
	}
}

func TestGlyph_GroundIsDrawnFullWidth(t *testing.T) {
	const w, h = 60, 25
	s := New(w, h, rand.New(rand.NewPCG(3, 4)))
	for x := 0; x < w; x++ {
		if s.Glyph(x, h-1) == 0 {
			t.Fatalf("no ground at column %d", x)
		}
	}
}

func TestNew_ZeroSizeIsSafe(t *testing.T) {
	s := New(0, 0, rand.New(rand.NewPCG(1, 1)))
	s.Tick()
	if s.Glyph(0, 0) != 0 {
		t.Error("zero-size scene drew a glyph")
	}
}

func TestSpriteDimensions(t *testing.T) {
	for f, frame := range sprites {
		if len(frame) != spriteH {
			t.Fatalf("frame %d has %d rows, want %d", f, len(frame), spriteH)
		}
		for r, row := range frame {
			if len(row) != spriteW {
				t.Fatalf("frame %d row %d has width %d, want %d", f, r, len(row), spriteW)
			}
		}
	}
	if m := slices.Max(jumpArc); m != maxJump {
		t.Fatalf("maxJump = %d, want max(jumpArc) = %d", maxJump, m)
	}
}
