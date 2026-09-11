package hi_test

import (
	"strings"
	"testing"

	"ily.dev/act3/hi"
)

// TestThemeOption verifies that the Theme option sets the root's colors:
// the background, Primary derived from it for text,
// and the color scheme the background's lightness calls for.
// The background is always opaque, and the contrast level is clamped.
func TestThemeOption(t *testing.T) {
	tests := []struct {
		name     string
		bg       hi.Color
		contrast float64
		want     string
	}{
		{"light", hi.OKLCH(0.95, 0.02, 80), 30, "background-color:oklch(0.95 0.02 80);color:oklch(0.1634 0.0133 80);color-scheme:light"},
		{"theme color", hi.ThemeColor(0.1, 0, hi.BackgroundScale), 30, "background-color:oklch(0.882 0.0013 100);color:oklch(0.1517 0.00395 100);color-scheme:light"},
		{"dark", hi.OKLCH(0.2, 0.03, 215), 30, "background-color:oklch(0.2 0.03 215);color:oklch(0.9312 0.0183 215);color-scheme:dark"},
		{"mid light", hi.OKLCH(0.65, 0, 0), 30, "background-color:oklch(0.65 0 0);color:oklch(0.1118 0.0033 0);color-scheme:light"},
		{"mid dark", hi.OKLCH(0.5, 0.3, 0), 30, "background-color:oklch(0.5 0.3 0);color:oklch(0.957 0.1533 0);color-scheme:dark"},
		{"translucent background", hi.OKLCHA(0.2, 0.03, 215, 0.5), 30, "background-color:oklch(0.2 0.03 215);color:oklch(0.9312 0.0183 215);color-scheme:dark"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, hi.Image("/x.png"), hi.Theme(tt.bg, hi.OKLCH(0.5, 0.2, 280), tt.contrast))
			if got := classRule(t, html, `<hi-root class="(hi-\w+)"`); got != tt.want {
				t.Errorf("root rule = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestThemeBackground pins what a theme background establishes on its
// box: its color as the background, Primary derived from it as the
// text color, and a color scheme only when its lightness crosses the
// light-dark boundary of the enclosing theme.
func TestThemeBackground(t *testing.T) {
	tests := []struct {
		name string
		c    hi.Color
		want string
	}{
		{"dark on light", hi.OKLCH(0.2, 0.03, 215), "background-color:oklch(0.2 0.03 215);color:oklch(0.9312 0.0183 215);color-scheme:dark;display:block;overflow-wrap:break-word"},
		{"light on light", hi.OKLCH(0.9, 0, 0), "background-color:oklch(0.9 0 0);color:oklch(0.1548 0.0033 0);display:block;overflow-wrap:break-word"},
		{"opacity ignored", hi.OKLCHA(0.2, 0, 0, 0.5), "background-color:oklch(0.2 0 0);color:oklch(0.9312 0.0033 0);color-scheme:dark;display:block;overflow-wrap:break-word"},
		{"theme color", hi.ThemeColor(0.1, 0, hi.BackgroundScale), "background-color:oklch(0.882 0.0013 100);color:oklch(0.1517 0.00395 100);display:block;overflow-wrap:break-word"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, hi.Text("x").ThemeBackground(tt.c))
			if got := classRule(t, html, `<hi-text class="(hi-\w+)"`); got != tt.want {
				t.Errorf("theme background rule = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestThemeBackgroundRebases pins that theme colors inside a theme
// background derive from it rather than from the page, including when
// nested.
func TestThemeBackgroundRebases(t *testing.T) {
	// On the dark background, the background scale's factor is +1 at
	// contrast 30, so the delta applies as-is.
	shifted := hi.ThemeColor(0.1, 0, hi.BackgroundScale)
	html := render(t, hi.Text("x").Foreground(shifted).ThemeBackground(hi.OKLCH(0.2, 0, 0)))
	if got := classRule(t, html, `<hi-text class="(hi-\w+)"`); !strings.Contains(got, "color:oklch(0.3 0 0)") {
		t.Errorf("foreground inside a theme background = %q, want derived from it", got)
	}

	nested := render(t, hi.VStack(hi.Text("x").ThemeBackground(shifted)).ThemeBackground(hi.OKLCH(0.2, 0, 0)))
	if got := classRule(t, nested, `<hi-text class="(hi-\w+)"`); !strings.Contains(got, "background-color:oklch(0.3 0 0)") || strings.Contains(got, "color-scheme") {
		t.Errorf("nested theme background = %q, want derived from the enclosing one without a scheme change", got)
	}

	// Paint applied outside the theme background lands on a wrapper box
	// and resolves in the enclosing theme. On the page, the factor is
	// -1, so the delta moves toward black.
	outer := render(t, hi.Text("x").ThemeBackground(hi.OKLCH(0.2, 0, 0)).BorderStroke(1, shifted))
	if got := classRule(t, outer, `<hi-box class="(hi-\w+)"`); !strings.Contains(got, "box-shadow:inset 0 0 0 1px oklch(0.882 0.0013 100)") {
		t.Errorf("stroke outside a theme background = %q, want derived from the page", got)
	}
	if got := classRule(t, outer, `<hi-text class="(hi-\w+)"`); strings.Contains(got, "box-shadow") {
		t.Errorf("theme background box = %q, want no stroke of its own", got)
	}

	// The theme background's own text color is set inside a foreground
	// set outside it, so the theme background's wins.
	fg := render(t, hi.Text("x").ThemeBackground(hi.OKLCH(0.2, 0, 0)).Foreground(shifted))
	if got := classRule(t, fg, `<hi-text class="(hi-\w+)"`); !strings.Contains(got, "color:oklch(0.9312 0.0033 0)") {
		t.Errorf("foreground outside a theme background = %q, want its text color", got)
	}
}
