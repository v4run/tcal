package render

import (
	"regexp"
	"strings"
	"testing"
)

// stripANSI removes ANSI escape sequences from s, returning the visible text.
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func TestHighlight_None(t *testing.T) {
	got := Highlight("22", HighlightNone)
	if got != "22" {
		t.Errorf("got %q, want %q", got, "22")
	}
}

func TestHighlight_Bracket(t *testing.T) {
	got := Highlight("22", HighlightBracket)
	if got != "[22]" {
		t.Errorf("got %q, want %q", got, "[22]")
	}
}

func TestHighlight_BracketWithSingleDigit(t *testing.T) {
	got := Highlight("3", HighlightBracket)
	if got != "[3]" {
		t.Errorf("got %q, want %q", got, "[3]")
	}
}

func TestHighlight_ReverseProducesAnsi(t *testing.T) {
	got := Highlight("22", HighlightReverse)
	// We don't assert exact escape codes (lipgloss owns those), only that
	// the visible text is preserved and ANSI escapes wrap it.
	if !strings.Contains(got, "22") {
		t.Errorf("reverse should preserve text, got %q", got)
	}
	if got == "22" {
		t.Errorf("reverse should add styling, got identical plain string")
	}
}

func TestHighlight_Combined_HasNoBrackets(t *testing.T) {
	got := Highlight("22", HighlightReverse|HighlightColor)
	visible := stripANSI(got)
	if strings.Contains(visible, "[") || strings.Contains(visible, "]") {
		t.Errorf("combined (reverse+color) must NOT contain brackets, got %q", got)
	}
	if !strings.Contains(got, "22") {
		t.Errorf("combined must preserve the day digits, got %q", got)
	}
}

func TestHighlight_All_EqualsCombined(t *testing.T) {
	got := Highlight("22", HighlightAll)
	want := Highlight("22", HighlightReverse|HighlightColor)
	if got != want {
		t.Errorf("HighlightAll should equal Reverse|Color, got %q want %q", got, want)
	}
}
