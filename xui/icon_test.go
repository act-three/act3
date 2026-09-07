package ui_test

import (
	"strings"
	"testing"

	"ily.dev/domi"

	ui "ily.dev/act3/xui"
)

// TestIconSource pins Icon's lookup: a source's svg is the icon's
// content, and a nil result falls back to the placeholder — as does
// having no source at all.
func TestIconSource(t *testing.T) {
	source := ui.IconSource(func(name string) domi.Node {
		if name != "film" {
			return nil
		}
		return domi.Tag("svg", domi.Name("data-icon", name))()
	})
	for _, tt := range []struct {
		name string
		view ui.View
		opts []ui.Option
		want string
	}{
		{"found", ui.Icon("film"), []ui.Option{source}, `<svg data-icon="film">`},
		{"missing", ui.Icon("nope"), []ui.Option{source}, `stroke-linecap="round"`},
		{"no source", ui.Icon("film"), nil, `stroke-linecap="round"`},
	} {
		_, page := ui.Render(tt.view, tt.opts...)
		var sb strings.Builder
		if err := domi.RenderTo(&sb, page); err != nil {
			t.Fatalf("%s: render: %v", tt.name, err)
		}
		if !strings.Contains(sb.String(), "<ui-icon") || !strings.Contains(sb.String(), tt.want) {
			t.Errorf("%s: page lacks %q:\n%s", tt.name, tt.want, sb.String())
		}
	}
}
