package ui_test

import (
	"testing"

	ui "ily.dev/act3/xui"
)

func TestOKLCH(t *testing.T) {
	tests := []struct {
		name  string
		color ui.Color
		want  string
	}{
		{"opaque", ui.OKLCH(0.7, 0.15, 250), "oklch(0.7 0.15 250)"},
		{"gray", ui.OKLCH(0.5, 0, 0), "oklch(0.5 0 0)"},
		{"opaque alpha", ui.OKLCHA(0.7, 0.15, 250, 1), "oklch(0.7 0.15 250)"},
		{"clamped alpha", ui.OKLCHA(0.7, 0.15, 250, 1.5), "oklch(0.7 0.15 250)"},
		{"clamped negative alpha", ui.OKLCHA(0.7, 0.15, 250, -0.5), "oklch(0.7 0.15 250 / 0)"},
		{"clamped lightness and chroma", ui.OKLCH(1.5, -0.1, 250), "oklch(1 0 250)"},
		{"clamped negative lightness", ui.OKLCH(-0.5, 0.15, 250), "oklch(0 0.15 250)"},
		{"unclamped hue", ui.OKLCH(0.7, 0.15, 400), "oklch(0.7 0.15 400)"},
		{"translucent", ui.OKLCHA(0.7, 0.15, 250, 0.5), "oklch(0.7 0.15 250 / 0.5)"},
		{"transparent", ui.OKLCHA(0.7, 0.15, 250, 0), "oklch(0.7 0.15 250 / 0)"},
		{"fractional hue", ui.OKLCH(0.62, 0.21, 27.5), "oklch(0.62 0.21 27.5)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.color)
			if got := classRule(t, html, `<ui-color class="(ui-\w+)"`); got != "align-self:stretch;background-color:"+tt.want+";justify-self:stretch" {
				t.Errorf("got %q, want background-color:%s:\n%s", got, tt.want, html)
			}
		})
	}
}
