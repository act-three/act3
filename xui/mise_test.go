package ui

import (
	"strings"
	"testing"

	"ily.dev/domi"
)

// TestBoxIsInnermostWriter pins the write-order rule for contended
// values: a box contributes its own values through the same
// environment slots as the modifiers wrapping it, writing last,
// so its value wins and build applies unconditionally.
func TestBoxIsInnermostWriter(t *testing.T) {
	var env environment
	env.tag.Set("picture") // an outer Tag modifier
	b := imageNode{src: "x.png"}.render(env)
	var sb strings.Builder
	if err := domi.RenderTo(&sb, b.node); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := sb.String(); !strings.Contains(got, "<img ") || strings.Contains(got, "picture") {
		t.Errorf("image should keep its img tag:\n%s", got)
	}
}

// TestTakeMiseStripsEnvironment pins consume-exactly-once: the
// first boundary to take the mise empties the environment,
// so nothing is left for a second boundary in the same subtree.
func TestTakeMiseStripsEnvironment(t *testing.T) {
	var env environment
	env.fg.Set("red")
	env.font.Set(Title)
	env.takeMise()
	if _, ok := env.fg.Take(); ok {
		t.Error("foreground color survived takeMise")
	}
	if _, ok := env.font.Take(); ok {
		t.Error("font size survived takeMise")
	}
}
