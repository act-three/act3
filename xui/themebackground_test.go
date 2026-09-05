package ui_test

import (
	"strings"
	"testing"

	ui "ily.dev/act3/xui"
)

// TestThemeBackground pins what a theme background establishes on its
// box: its color as the background, the foreground color for text on
// it, and a color scheme only when its lightness crosses the light-dark
// boundary of the enclosing theme.
func TestThemeBackground(t *testing.T) {
	tests := []struct {
		name string
		c    ui.Color
		want string
	}{
		{"dark on light", ui.OKLCH(0.2, 0.03, 215), "background-color:oklch(0.2 0.03 215);color:oklch(1 0.015 215);color-scheme:dark;display:block;overflow-wrap:break-word"},
		{"light on light", ui.OKLCH(0.9, 0, 0), "background-color:oklch(0.9 0 0);color:oklch(0 0 0);display:block;overflow-wrap:break-word"},
		{"opacity ignored", ui.OKLCHA(0.2, 0, 0, 0.5), "background-color:oklch(0.2 0 0);color:oklch(1 0 0);color-scheme:dark;display:block;overflow-wrap:break-word"},
		{"theme color", ui.ThemeColor(0.1, 0, ui.BackgroundScale), "background-color:oklch(0.9 0 0);color:oklch(0 0 0);display:block;overflow-wrap:break-word"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, ui.Text("x").ThemeBackground(tt.c))
			if got := classRule(t, html, `<ui-text class="(ui-\w+)"`); got != tt.want {
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
	shifted := ui.ThemeColor(0.1, 0, ui.BackgroundScale)
	html := render(t, ui.Text("x").Foreground(shifted).ThemeBackground(ui.OKLCH(0.2, 0, 0)))
	if got := classRule(t, html, `<ui-text class="(ui-\w+)"`); !strings.Contains(got, "color:oklch(0.3 0 0)") {
		t.Errorf("foreground inside a theme background = %q, want derived from it", got)
	}

	nested := render(t, ui.VStack(ui.Text("x").ThemeBackground(shifted)).ThemeBackground(ui.OKLCH(0.2, 0, 0)))
	if got := classRule(t, nested, `<ui-text class="(ui-\w+)"`); !strings.Contains(got, "background-color:oklch(0.3 0 0)") || strings.Contains(got, "color-scheme") {
		t.Errorf("nested theme background = %q, want derived from the enclosing one without a scheme change", got)
	}

	// Paint applied outside the theme background lands on a wrapper box
	// and resolves in the enclosing theme. On the page, the factor is
	// -1, so the delta moves toward black.
	outer := render(t, ui.Text("x").ThemeBackground(ui.OKLCH(0.2, 0, 0)).BorderStroke(1, shifted))
	if got := classRule(t, outer, `<ui-box class="(ui-\w+)"`); !strings.Contains(got, "box-shadow:inset 0 0 0 1px oklch(0.9 0 0)") {
		t.Errorf("stroke outside a theme background = %q, want derived from the page", got)
	}
	if got := classRule(t, outer, `<ui-text class="(ui-\w+)"`); strings.Contains(got, "box-shadow") {
		t.Errorf("theme background box = %q, want no stroke of its own", got)
	}

	// The theme background's own text color is set inside a foreground
	// set outside it, so the theme background's wins.
	fg := render(t, ui.Text("x").ThemeBackground(ui.OKLCH(0.2, 0, 0)).Foreground(shifted))
	if got := classRule(t, fg, `<ui-text class="(ui-\w+)"`); !strings.Contains(got, "color:oklch(1 0 0)") {
		t.Errorf("foreground outside a theme background = %q, want its text color", got)
	}
}

func TestThemeBackgroundCSSColorPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("ThemeBackground with a CSS color did not panic")
		}
	}()
	render(t, ui.Text("x").ThemeBackground(ui.CSSColor("red")))
}
