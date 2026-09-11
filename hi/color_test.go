package hi_test

import (
	"testing"

	"ily.dev/act3/hi"
)

func TestOKLCH(t *testing.T) {
	tests := []struct {
		name  string
		color hi.Color
		want  string
	}{
		{"opaque", hi.OKLCH(0.7, 0.15, 250), "oklch(0.7 0.15 250)"},
		{"gray", hi.OKLCH(0.5, 0, 0), "oklch(0.5 0 0)"},
		{"opaque alpha", hi.OKLCHA(0.7, 0.15, 250, 1), "oklch(0.7 0.15 250)"},
		{"clamped alpha", hi.OKLCHA(0.7, 0.15, 250, 1.5), "oklch(0.7 0.15 250)"},
		{"clamped negative alpha", hi.OKLCHA(0.7, 0.15, 250, -0.5), "oklch(0.7 0.15 250 / 0)"},
		{"clamped lightness and chroma", hi.OKLCH(1.5, -0.1, 250), "oklch(1 0 250)"},
		{"clamped high chroma", hi.OKLCH(0.5, 1, 250), "oklch(0.5 0.5 250)"},
		{"clamped negative lightness", hi.OKLCH(-0.5, 0.15, 250), "oklch(0 0.15 250)"},
		{"unclamped hue", hi.OKLCH(0.7, 0.15, 400), "oklch(0.7 0.15 400)"},
		{"translucent", hi.OKLCHA(0.7, 0.15, 250, 0.5), "oklch(0.7 0.15 250 / 0.5)"},
		{"transparent", hi.OKLCHA(0.7, 0.15, 250, 0), "oklch(0.7 0.15 250 / 0)"},
		{"fractional hue", hi.OKLCH(0.62, 0.21, 27.5), "oklch(0.62 0.21 27.5)"},
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
