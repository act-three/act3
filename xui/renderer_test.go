package ui_test

import (
	"strings"
	"testing"

	ui "ily.dev/act3/xui"
	"ily.dev/domi"
)

func renderWith(t *testing.T, r *ui.Renderer, v ui.View) string {
	t.Helper()
	var sb strings.Builder
	if err := domi.RenderTo(&sb, r.Render(v)); err != nil {
		t.Fatalf("render: %v", err)
	}
	return sb.String()
}

// TestRendererAccumulatesRules verifies that rules remain in the stylesheet after their views disappear.
// It also verifies that those rules are reused when the views return.
func TestRendererAccumulatesRules(t *testing.T) {
	var r ui.Renderer
	first := renderWith(t, &r, ui.Text("a").Padding(ui.Edges(16)))
	if !strings.Contains(first, "padding:16px") {
		t.Fatalf("first render missing its rule:\n%s", first)
	}
	gone := renderWith(t, &r, ui.Text("a"))
	if !strings.Contains(gone, "padding:16px") {
		t.Errorf("rule dropped when its view went away:\n%s", gone)
	}
	back := renderWith(t, &r, ui.Text("a").Padding(ui.Edges(16)))
	if back != first {
		t.Errorf("revisit is not byte-identical to the first render:\n%s\nvs:\n%s", back, first)
	}
}

// TestRendererStyleElement verifies that the style element is always present.
// It must be the first child of ui-root
// and include the configured nonce.
func TestRendererStyleElement(t *testing.T) {
	plain := renderWith(t, new(ui.Renderer), ui.Text("x"))
	if !strings.Contains(plain, "<ui-root><style></style>") {
		t.Errorf("empty style element not first in ui-root:\n%s", plain)
	}
	nonced := renderWith(t, &ui.Renderer{Nonce: "abc123"}, ui.Text("x"))
	if !strings.Contains(nonced, `<ui-root><style nonce="abc123">`) {
		t.Errorf("style element does not carry the nonce:\n%s", nonced)
	}
}
