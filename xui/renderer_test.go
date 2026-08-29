package ui_test

import (
	"strings"
	"testing"

	"ily.dev/domi"

	ui "ily.dev/act3/xui"
)

func renderWith(t *testing.T, r *ui.Renderer, v ui.View) string {
	t.Helper()
	var sb strings.Builder
	_, page := r.Render(v)
	if err := domi.RenderTo(&sb, page); err != nil {
		t.Fatalf("render: %v", err)
	}
	return sb.String()
}

// TestRendererAccumulatesRules verifies that rules remain in the stylesheet after their views disappear.
// It also verifies that those rules are reused when the views return.
func TestRendererAccumulatesRules(t *testing.T) {
	var r ui.Renderer
	first := renderWith(t, &r, ui.Text("a").Padding(ui.Edges(16)))
	if !strings.Contains(first, "padding-block-start:16px") {
		t.Fatalf("first render missing its rule:\n%s", first)
	}
	gone := renderWith(t, &r, ui.Text("a"))
	if !strings.Contains(gone, "padding-block-start:16px") {
		t.Errorf("rule dropped when its view went away:\n%s", gone)
	}
	back := renderWith(t, &r, ui.Text("a").Padding(ui.Edges(16)))
	if back != first {
		t.Errorf("revisit is not byte-identical to the first render:\n%s\nvs:\n%s", back, first)
	}
}

// TestRendererTitle verifies how a page title flows out of a view tree:
// the innermost title wins, then the first of several siblings,
// a title applied to a group lands on each member,
// and wrappers and layers pass a title through.
func TestRendererTitle(t *testing.T) {
	for _, tt := range []struct {
		name string
		view ui.View
		want string
	}{
		{"none", ui.Text("a"), ""},
		{"leaf", ui.Text("a").Title("x"), "x"},
		{"inner wins", ui.VStack(ui.Text("a").Title("x")).Title("y"), "x"},
		{"outer fills empty", ui.VStack(ui.Text("a")).Title("y"), "y"},
		{"first sibling wins", ui.VStack(ui.Text("a").Title("x"), ui.Text("b").Title("z")), "x"},
		{"empty sibling skipped", ui.VStack(ui.Text("a"), ui.Text("b").Title("x")), "x"},
		{"group", ui.Group(ui.Text("a").Title("x"), ui.Text("b")), "x"},
		{"group outer yields to first member", ui.Group(ui.Text("a").Title("x"), ui.Text("b")).Title("d"), "x"},
		{"group outer lands on first member", ui.Group(ui.Text("a"), ui.Text("b").Title("x")).Title("d"), "d"},
		{"group outer fills empty", ui.Group(ui.Text("a"), ui.Text("b")).Title("d"), "d"},
		{"for", ui.For([]string{"a", "b"}, nil, func(s string) ui.View { return ui.Text(s).Title(s) }).Title("d"), "a"},
		{"keyed for", ui.For([]string{"a", "b"}, func(s string) string { return s }, func(s string) ui.View { return ui.Text(s).Title(s) }).Title("d"), "a"},
		{"through padding", ui.Text("a").Title("x").Padding(ui.Edges(8)), "x"},
		{"through frame", ui.Text("a").Title("x").Frame(ui.Width(40)), "x"},
		{"through paint", ui.Text("a").Title("x").Background(ui.Accent).Opacity(0.5), "x"},
		{"through scroll", ui.ScrollView(ui.Vertical, ui.Text("a").Title("x")), "x"},
		{"from overlay", ui.Text("a").Overlay(ui.Center, ui.Text("b").Title("x")), "x"},
		{"base wins over overlay", ui.Text("a").Title("x").Overlay(ui.Center, ui.Text("b").Title("z")), "x"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := new(ui.Renderer).Render(tt.view)
			if got != tt.want {
				t.Errorf("title = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRendererStyleElement verifies that the style element is always present.
// It must be the first child of ui-root
// and include the configured nonce.
func TestRendererStyleElement(t *testing.T) {
	// A native image is the one view with no declarations of its own.
	plain := renderWith(t, new(ui.Renderer), ui.Image("/x.png"))
	if !strings.Contains(plain, "<ui-root><style>@layer xui{}</style>") {
		t.Errorf("empty style element not first in ui-root:\n%s", plain)
	}
	nonced := renderWith(t, &ui.Renderer{Nonce: "abc123"}, ui.Image("/x.png"))
	if !strings.Contains(nonced, `<ui-root><style nonce="abc123">`) {
		t.Errorf("style element does not carry the nonce:\n%s", nonced)
	}
}
