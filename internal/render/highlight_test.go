package render

import (
	"strings"
	"testing"
)

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

func TestHighlight_Combined_IncludesBrackets(t *testing.T) {
	got := Highlight("22", HighlightReverse|HighlightBracket|HighlightColor)
	if !strings.Contains(got, "[22]") {
		t.Errorf("combined must contain [22], got %q", got)
	}
}
