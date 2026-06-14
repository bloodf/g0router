package mcp

import (
	"strings"
	"testing"
)

// TestSmartFilterTextDropsNoise: lines whose role is "generic" and empty "text"
// lines are dropped; meaningful role-prefixed lines pass through.
func TestSmartFilterTextDropsNoise(t *testing.T) {
	in := strings.Join([]string{
		`generic "wrapper"`,
		`button "Submit"`,
		`text ""`,
		`text "Hello"`,
		`generic`,
		`link "Home"`,
	}, "\n")
	got := smartFilterText(in)
	if strings.Contains(got, "generic") {
		t.Errorf("generic node not dropped: %q", got)
	}
	if strings.Contains(got, `text ""`) {
		t.Errorf("empty text line not dropped: %q", got)
	}
	if !strings.Contains(got, `button "Submit"`) {
		t.Errorf("meaningful line dropped: %q", got)
	}
	if !strings.Contains(got, `text "Hello"`) {
		t.Errorf("non-empty text dropped: %q", got)
	}
	if !strings.Contains(got, `link "Home"`) {
		t.Errorf("meaningful line dropped: %q", got)
	}
}

// TestSmartFilterTextCollapsesRepeatedSiblings: consecutive identical
// role-prefixed sibling lines collapse to a single occurrence.
func TestSmartFilterTextCollapsesRepeatedSiblings(t *testing.T) {
	in := strings.Join([]string{
		`listitem "Row"`,
		`listitem "Row"`,
		`listitem "Row"`,
		`listitem "Other"`,
	}, "\n")
	got := smartFilterText(in)
	if n := strings.Count(got, `listitem "Row"`); n != 1 {
		t.Errorf("repeated siblings not collapsed: count=%d in %q", n, got)
	}
	if !strings.Contains(got, `listitem "Other"`) {
		t.Errorf("distinct sibling dropped: %q", got)
	}
}

// TestSmartFilterTextTruncatesAt50K: an input longer than the hard cap is
// truncated to exactly maxToolResultChars.
func TestSmartFilterTextTruncatesAt50K(t *testing.T) {
	// Build a clean, non-collapsible, non-noise input well over the cap.
	var b strings.Builder
	for i := 0; b.Len() < maxToolResultChars+5000; i++ {
		b.WriteString("text \"line-")
		b.WriteString(strings.Repeat("x", 10))
		b.WriteByte(byte('0' + (i % 10)))
		b.WriteString("\"\n")
	}
	got := smartFilterText(b.String())
	if len(got) != maxToolResultChars {
		t.Fatalf("len(got) = %d, want exactly %d", len(got), maxToolResultChars)
	}
}

// TestSmartFilterTextCleanShortUnchanged: a clean short string with no noise and
// no repeats is returned unchanged.
func TestSmartFilterTextCleanShortUnchanged(t *testing.T) {
	in := strings.Join([]string{
		`heading "Title"`,
		`text "Body content"`,
	}, "\n")
	got := smartFilterText(in)
	if got != in {
		t.Fatalf("clean input changed:\n got=%q\nwant=%q", got, in)
	}
}

// TestMaxToolResultChars pins the 9router hard cap.
func TestMaxToolResultChars(t *testing.T) {
	if maxToolResultChars != 50000 {
		t.Fatalf("maxToolResultChars = %d, want 50000", maxToolResultChars)
	}
}
