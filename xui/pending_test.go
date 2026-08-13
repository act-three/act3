package ui

import "testing"

// TestConsumedColorYieldsToBox pins the consumption discipline for
// inherited values: a consumed Foreground lands only where the box
// carries no value of its own — the box is the innermost writer.
func TestConsumedColorYieldsToBox(t *testing.T) {
	var env environment
	env.fg.Set("red")
	p := env.takePending()

	blue := Color("blue")
	opinion := box{pres: presentation{color: &blue}}
	p.applyTo(&opinion)
	if got := *opinion.pres.color; got != "blue" {
		t.Errorf("consumed color overwrote the box's own: %s, want blue", got)
	}

	var plain box
	env.fg.Set("red")
	env.takePending().applyTo(&plain)
	if c := plain.pres.color; c == nil || *c != "red" {
		t.Errorf("consumed color missing from an unopinionated box: %v", c)
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
