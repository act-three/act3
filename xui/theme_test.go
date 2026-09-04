package ui

import (
	"context"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	"ily.dev/domi"
)

// TestThemeOption verifies that the Theme option sets the root's colors:
// the base as the background, white or black text to read on it,
// and the color scheme the base's lightness calls for.
// The base is always opaque, and the contrast level is clamped.
func TestThemeOption(t *testing.T) {
	tests := []struct {
		name     string
		base     OKLCHColor
		contrast float64
		want     string
	}{
		{"light", OKLCH(0.95, 0.02, 80), 30, "background-color:oklch(0.95 0.02 80);color:oklch(0 0.01 80);color-scheme:light"},
		{"dark", OKLCH(0.2, 0.03, 215), 30, "background-color:oklch(0.2 0.03 215);color:oklch(1 0.015 215);color-scheme:dark"},
		{"mid light", OKLCH(0.65, 0, 0), 30, "background-color:oklch(0.65 0 0);color:oklch(0 0 0);color-scheme:light"},
		{"mid dark", OKLCH(0.5, 0.3, 0), 30, "background-color:oklch(0.5 0.3 0);color:oklch(1 0.15 0);color-scheme:dark"},
		{"translucent base", OKLCHA(0.2, 0.03, 215, 0.5), 30, "background-color:oklch(0.2 0.03 215);color:oklch(1 0.015 215);color-scheme:dark"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Handler(
				func(context.Context, *url.URL) (*stubApp, domi.Cmd[struct{}]) {
					return &stubApp{view: Image("/x.png")}, nil
				},
				func(*url.URL) struct{} { return struct{}{} },
				func(*url.URL) struct{} { return struct{}{} },
				Theme(tt.base, OKLCH(0.5, 0.2, 280), tt.contrast),
			)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
			if rec.Code != 200 {
				t.Fatalf("status = %d, body:\n%s", rec.Code, rec.Body)
			}
			body := rec.Body.String()
			m := regexp.MustCompile(`<ui-root class="(ui-\w+)">`).FindStringSubmatch(body)
			if m == nil {
				t.Fatalf("no root class:\n%s", body)
			}
			r := regexp.MustCompile(regexp.QuoteMeta("."+m[1]) + `\{([^}]*)\}`).FindStringSubmatch(body)
			if r == nil {
				t.Fatalf("no rule for root class %s:\n%s", m[1], body)
			}
			if r[1] != tt.want {
				t.Errorf("root rule = %q, want %q", r[1], tt.want)
			}
		})
	}
}

func TestThemeContrastClamped(t *testing.T) {
	for _, tt := range []struct{ in, want float64 }{
		{0, 15}, {15, 15}, {50, 50}, {100, 100}, {200, 100},
	} {
		o := Theme(OKLCH(1, 0, 0), OKLCH(0.5, 0.2, 280), tt.in).(optionTheme)
		if o.theme.contrast != tt.want {
			t.Errorf("Theme(contrast %v): contrast = %v, want %v", tt.in, o.theme.contrast, tt.want)
		}
	}
}
