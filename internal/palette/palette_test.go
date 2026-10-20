package palette

import (
	"math"
	"strings"
	"testing"

	"github.com/v4run/tcal/internal/scene"
)

func TestParse(t *testing.T) {
	m, auto, err := Parse("auto")
	if err != nil || !auto || (m != TrueColor && m != ANSI) {
		t.Errorf("auto: got (%v, %v, %v)", m, auto, err)
	}
	if m, auto, err := Parse("TrueColor"); m != TrueColor || auto || err != nil {
		t.Errorf("TrueColor: got (%v, %v, %v)", m, auto, err)
	}
	if m, auto, err := Parse("ansi"); m != ANSI || auto || err != nil {
		t.Errorf("ansi: got (%v, %v, %v)", m, auto, err)
	}
	if _, _, err := Parse("256"); err == nil || !strings.Contains(err.Error(), "256") {
		t.Errorf("256: want error naming value, got %v", err)
	}
}

func TestDetect(t *testing.T) {
	env := func(v string) func(string) string {
		return func(k string) string {
			if k == "COLORTERM" {
				return v
			}
			return ""
		}
	}
	for v, want := range map[string]Mode{"truecolor": TrueColor, "24bit": TrueColor, "": ANSI, "yes": ANSI} {
		if got := Detect(env(v)); got != want {
			t.Errorf("COLORTERM=%q: got %v want %v", v, got, want)
		}
	}
}

func TestContrast(t *testing.T) {
	if got := Contrast(Dark, Light); math.Abs(got-21) > 0.01 {
		t.Errorf("Dark/Light = %v", got)
	}
	x := scene.RGB{R: 90, G: 120, B: 200}
	if got := Contrast(x, x); math.Abs(got-1) > 1e-9 {
		t.Errorf("same = %v", got)
	}
	y := scene.RGB{R: 10, G: 20, B: 30}
	if Contrast(x, y) != Contrast(y, x) {
		t.Error("not symmetric")
	}
}

func TestBlend(t *testing.T) {
	c := scene.RGB{R: 200, G: 0, B: 0}
	if got := Blend(c, Dark, 1); got != c {
		t.Errorf("keep 1: %v", got)
	}
	bg := scene.RGB{R: 5, G: 6, B: 7}
	if got := Blend(c, bg, 0); got != bg {
		t.Errorf("keep 0: %v", got)
	}
	if got := Blend(c, Dark, 0.5); got != (scene.RGB{R: 100}) {
		t.Errorf("keep .5: %v", got)
	}
}

func TestNearest(t *testing.T) {
	cases := []struct {
		c    scene.RGB
		want scene.Color
	}{
		{scene.RGB{R: 200, G: 40, B: 40}, scene.Red},
		{scene.RGB{R: 40, G: 180, B: 60}, scene.Green},
		{scene.RGB{R: 60, G: 90, B: 220}, scene.Blue},
		{scene.RGB{R: 230, G: 200, B: 60}, scene.Yellow},
		{scene.RGB{R: 60, G: 200, B: 210}, scene.Cyan},
		{scene.RGB{R: 190, G: 60, B: 190}, scene.Magenta},
		{scene.RGB{R: 235, G: 235, B: 235}, scene.White},
		{scene.RGB{R: 128, G: 128, B: 128}, scene.Default},
	}
	for _, tc := range cases {
		if got := Nearest(tc.c); got != tc.want {
			t.Errorf("Nearest(%v) = %v want %v", tc.c, got, tc.want)
		}
	}
}
