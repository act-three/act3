package hi

import "testing"

func TestButtonStateScope(t *testing.T) {
	var label, sibling environment
	Render(VStack(
		Button(struct{}{}, base{envProbe(&label)}).Selected(true).MenuOpen(true),
		base{envProbe(&sibling)},
	))
	for name, env := range map[string]environment{"label": label, "sibling": sibling} {
		if env.buttonSelected || env.buttonMenuOpen {
			t.Errorf("button state leaked into %s", name)
		}
	}
}
