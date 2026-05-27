package render

import (
	"strings"
	"testing"
)

func TestFit_CentersHorizontallyAndVertically(t *testing.T) {
	block := "abc\nde"
	got := Fit(block, 7, 5)

	// width=7, content=3 → 2 spaces left, 2 right.
	// height=5, content=2 → 1 blank line above, 2 below (extra line goes below on odd diff).
	want := strings.Join([]string{
		"       ",
		"  abc  ",
		"  de   ",
		"       ",
		"       ",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestFit_NoVerticalCenteringWhenHeightZero(t *testing.T) {
	block := "abc"
	got := Fit(block, 7, 0)
	want := "  abc  "
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFit_TooNarrow_FallsBackToLeftAlign(t *testing.T) {
	block := "abcdefgh" // 8 wide
	got := Fit(block, 5, 0)
	if got != "abcdefgh" {
		t.Errorf("expected unchanged content on overflow, got %q", got)
	}
}

func TestFit_TooTall_TopAlignsWithinViewport(t *testing.T) {
	block := "a\nb\nc\nd\ne"
	got := Fit(block, 1, 3)
	// 5 lines into height=3 → top-align, no padding above.
	want := "a\nb\nc\nd\ne"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFit_PadsRightToFullWidth(t *testing.T) {
	block := "ab\ncde"
	got := Fit(block, 7, 0)
	// Each line padded to width=7 with content centered (block max width = 3).
	// (7-3)/2 = 2 spaces left, 2 right. Shorter lines still padded to 7.
	lines := strings.Split(got, "\n")
	for i, ln := range lines {
		if len(ln) != 7 {
			t.Errorf("line %d width=%d, want 7 (%q)", i, len(ln), ln)
		}
	}
}
