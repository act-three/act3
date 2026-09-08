package ui

import (
	"strings"
	"testing"

	"ily.dev/act3/xui/internal/sheet"
)

func TestIdealSizeDeclarations(t *testing.T) {
	for _, tt := range []struct {
		name      string
		ideal     rect
		unbounded AxisSet
		fills     AxisSet
		want      string
	}{
		{"zero", rect{}, Horizontal | Vertical, 0, ""},
		{"zero filling", rect{}, Horizontal | Vertical, Horizontal | Vertical, ""},
		{"width", rect{width: 10}, Horizontal | Vertical, 0, "width:10px"},
		{"height", rect{height: 10}, Horizontal | Vertical, 0, "height:10px"},
		{"filling width", rect{width: 10}, Horizontal, Horizontal, "min-width:10px"},
		{"filling height", rect{height: 10}, Vertical, Vertical, "min-height:10px"},
		{"bounded", rect{width: 10, height: 10}, 0, 0, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var sh sheet.Sheet
			env := environment{sheet: &sh}
			env.unbounded = tt.unbounded
			build(env, plan{ideal: tt.ideal, fills: tt.fills})
			css := sh.CSS()
			if strings.Contains(css, "width:0px") || strings.Contains(css, "height:0px") {
				t.Errorf("zero ideal emitted a dimension: %s", css)
			}
			if tt.want == "" {
				if strings.Contains(css, "width:") || strings.Contains(css, "height:") {
					t.Errorf("unexpected dimensions: %s", css)
				}
			} else if !strings.Contains(css, tt.want) {
				t.Errorf("missing %q: %s", tt.want, css)
			}
		})
	}
}
