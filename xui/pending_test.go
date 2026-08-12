package ui

import (
	"testing"

	"ily.dev/act3/xui/internal/sheet"
)

// classOf returns the canonical generated class for a style set.
// Equal declaration sets get equal class names in every sheet,
// so class equality is declaration-set equality.
func classOf(s sheet.StyleSet) string {
	return new(sheet.Sheet).ClassFor(s)
}

// TestConsumedStylesYieldToBox pins the consumption discipline for
// inherited declarations: applyTo lands consumed values under the
// box's own declarations, so a component opinion — the box being the
// innermost writer — wins by ordinary map-set order.
func TestConsumedStylesYieldToBox(t *testing.T) {
	var env environment
	env.fg.Set("red")
	p := env.takePending()

	var opinion box
	opinion.setStyle("color", "blue")
	p.applyTo(&opinion)
	if got, want := classOf(opinion.styles), classOf(sheet.Style("color", "blue")); got != want {
		t.Errorf("consumed color overwrote the box's own: class %s, want %s", got, want)
	}

	var plain box
	env.fg.Set("red")
	env.takePending().applyTo(&plain)
	if got, want := classOf(plain.styles), classOf(sheet.Style("color", "red")); got != want {
		t.Errorf("consumed color missing from an unopinionated box: class %s, want %s", got, want)
	}
}

// TestTakePendingStripsEnvironment pins consume-exactly-once: the
// first boundary to take the pending effects empties the environment,
// so nothing is left for a second boundary in the same subtree.
func TestTakePendingStripsEnvironment(t *testing.T) {
	var env environment
	env.fg.Set("red")
	env.font.Set(Title)
	env.takePending()
	if _, ok := env.fg.Take(); ok {
		t.Error("foreground color survived takePending")
	}
	if _, ok := env.font.Take(); ok {
		t.Error("font size survived takePending")
	}
}
