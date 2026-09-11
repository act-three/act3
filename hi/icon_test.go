package hi_test

import (
	"strings"
	"testing"

	"ily.dev/domi"

	"ily.dev/act3/hi"
)

// TestIconSource pins Icon's lookup: a source's svg is the icon's
// content, and a nil result falls back to the placeholder.
func TestIconSource(t *testing.T) {
	source := hi.IconSource(func(name string) domi.Node {
		if name != "film" {
			return nil
		}
		return domi.Tag("svg", domi.Name("data-icon", name))()
	})
	for _, tt := range []struct {
		name string
		view hi.View
		opts []hi.Option
		want string
	}{
		{"found", hi.Icon("film"), []hi.Option{source}, `<svg data-icon="film">`},
		{"missing", hi.Icon("nope"), []hi.Option{source}, `<svg`},
		{"default", hi.Icon("film"), nil, `<svg`},
		{"default missing", hi.Icon("nope"), nil, `<svg`},
	} {
		_, page := hi.Render(tt.view, tt.opts...)
		var sb strings.Builder
		if err := domi.RenderTo(&sb, page); err != nil {
			t.Fatalf("%s: render: %v", tt.name, err)
		}
		if !strings.Contains(sb.String(), "<hi-icon") || !strings.Contains(sb.String(), tt.want) {
			t.Errorf("%s: page lacks %q:\n%s", tt.name, tt.want, sb.String())
		}
	}
}
