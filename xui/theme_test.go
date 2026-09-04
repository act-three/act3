package ui

import (
	"context"
	"math"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	"ily.dev/domi"
)

// TestThemeOption verifies that the Theme option sets the root's colors:
// the background, white or black for text on it,
// and the color scheme the background's lightness calls for.
// The background is always opaque, and the contrast level is clamped.
func TestThemeOption(t *testing.T) {
	tests := []struct {
		name     string
		bg       OKLCHColor
		contrast float64
		want     string
	}{
		{"light", OKLCH(0.95, 0.02, 80), 30, "background-color:oklch(0.95 0.02 80);color:oklch(0 0.01 80);color-scheme:light"},
		{"dark", OKLCH(0.2, 0.03, 215), 30, "background-color:oklch(0.2 0.03 215);color:oklch(1 0.015 215);color-scheme:dark"},
		{"mid light", OKLCH(0.65, 0, 0), 30, "background-color:oklch(0.65 0 0);color:oklch(0 0 0);color-scheme:light"},
		{"mid dark", OKLCH(0.5, 0.3, 0), 30, "background-color:oklch(0.5 0.3 0);color:oklch(1 0.15 0);color-scheme:dark"},
		{"translucent background", OKLCHA(0.2, 0.03, 215, 0.5), 30, "background-color:oklch(0.2 0.03 215);color:oklch(1 0.015 215);color-scheme:dark"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Handler(
				func(context.Context, *url.URL) (*stubApp, domi.Cmd[struct{}]) {
					return &stubApp{view: Image("/x.png")}, nil
				},
				func(*url.URL) struct{} { return struct{}{} },
				func(*url.URL) struct{} { return struct{}{} },
				Theme(tt.bg, OKLCH(0.5, 0.2, 280), tt.contrast),
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

var (
	darkTheme  = theme{base: oklch{l: 0.2, c: 0.03, h: 215, a: 1}, accent: oklch{l: 0.6, c: 0.2, h: 10, a: 1}, contrast: 30}
	lightTheme = theme{base: oklch{l: 0.95, c: 0.03, h: 215, a: 1}, accent: oklch{l: 0.6, c: 0.2, h: 10, a: 1}, contrast: 30}
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// factor returns s's factor in th for a color derived from s's base color.
func factor(s ColorScale, th theme) float64 {
	return s.factor(s.base().colorCoords(th), th.base, th.contrast)
}

// TestScaleSign pins the sign convention: a scale's factor is positive
// with a dark base color and negative with a light one, so a positive
// delta always moves away from the base color's extreme. A foreground
// color's base color is the text color, which is light on a dark
// background.
func TestScaleSign(t *testing.T) {
	for _, s := range []ColorScale{BackgroundScale, ControlScale, BorderScale} {
		if f := factor(s, darkTheme); !(f > 0) {
			t.Errorf("dark factor(%d) = %v, want positive", s, f)
		}
		if f := factor(s, lightTheme); !(f < 0) {
			t.Errorf("light factor(%d) = %v, want negative", s, f)
		}
	}
	if f := factor(ForegroundScale, darkTheme); !(f < 0) {
		t.Errorf("dark foreground factor = %v, want negative (from white)", f)
	}
	if f := factor(ForegroundScale, lightTheme); !(f > 0) {
		t.Errorf("light foreground factor = %v, want positive (from black)", f)
	}
}

// TestScaleContrast pins that the surface, control, and border factors
// grow with contrast, and that the foreground factor shrinks: higher
// contrast pulls text less far from black or white.
func TestScaleContrast(t *testing.T) {
	low, high := darkTheme, darkTheme
	low.contrast, high.contrast = 15, 100
	for _, s := range []ColorScale{BackgroundScale, ControlScale, BorderScale} {
		if factor(s, low) >= factor(s, high) {
			t.Errorf("factor(%d): contrast 15 = %v, contrast 100 = %v, want growth", s, factor(s, low), factor(s, high))
		}
	}
	if math.Abs(factor(ForegroundScale, low)) <= math.Abs(factor(ForegroundScale, high)) {
		t.Errorf("foreground factor: contrast 15 = %v, contrast 100 = %v, want decline", factor(ForegroundScale, low), factor(ForegroundScale, high))
	}
}

// TestForegroundScaleMidpoint pins the symmetric lightness scaling
// around middle gray, including backgrounds just below the mode threshold.
func TestForegroundScaleMidpoint(t *testing.T) {
	for _, tt := range []struct{ bg, want float64 }{
		{0, 0.8}, {0.45, 0.89}, {0.5, 0.9}, {0.55, 0.89},
		{0.6, 0.12}, {0.95, 0.19}, {1, 0.2},
	} {
		th := darkTheme
		th.base.l = tt.bg
		got := ThemeColor(0.2, 0, ForegroundScale).color().colorCoords(th)
		if !near(got.l, tt.want) {
			t.Errorf("background %v: foreground lightness = %v, want %v", tt.bg, got.l, tt.want)
		}
	}
}

// TestForegroundChroma pins that foreground chroma offsets are independent
// of contrast, background lightness, and mode, and remain clamped.
func TestForegroundChroma(t *testing.T) {
	for _, tt := range []struct {
		name string
		c    Color
		want float64
	}{
		{"primary", Primary, 0.0183},
		{"secondary", Secondary, 0.0183},
		{"tertiary", Tertiary, 0.0183},
		{"positive delta", ThemeColor(0.2, 0.01, ForegroundScale), 0.025},
		{"negative delta", ThemeColor(0.2, -0.01, ForegroundScale), 0.005},
		{"clamped low", ThemeColor(0.2, -1, ForegroundScale), 0},
		{"clamped high", ThemeColor(0.2, 1, ForegroundScale), 0.5},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, l := range []float64{0, 0.5, 0.55, 0.95, 1} {
				for _, contrast := range []float64{15, 30, 100} {
					th := darkTheme
					th.base.l, th.contrast = l, contrast
					got := tt.c.color().colorCoords(th)
					if !near(got.c, tt.want) {
						t.Errorf("background %v, contrast %v: chroma = %v, want %v", l, contrast, got.c, tt.want)
					}
				}
			}
		})
	}
}

func TestThemeColor(t *testing.T) {
	tests := []struct {
		name string
		th   theme
		c    Color
		want oklch
	}{
		{
			"surface dark", darkTheme, ThemeColor(0.1, 0.01, BackgroundScale),
			oklch{l: 0.2 + 0.1*factor(BackgroundScale, darkTheme), c: 0.03 + 0.01*factor(BackgroundScale, darkTheme), h: 215, a: 1},
		},
		{
			"surface light", lightTheme, ThemeColor(0.1, 0.01, BackgroundScale),
			oklch{l: 0.95 + 0.1*factor(BackgroundScale, lightTheme), c: 0.03 - 0.01*factor(BackgroundScale, lightTheme), h: 215, a: 1},
		},
		{
			"foreground dark recedes from white", darkTheme, ThemeColor(0.1, 0, ForegroundScale),
			oklch{l: 1 + 0.1*factor(ForegroundScale, darkTheme), c: 0.015, h: 215, a: 1},
		},
		{
			"foreground light recedes from black", lightTheme, ThemeColor(0.1, 0, ForegroundScale),
			oklch{l: 0 + 0.1*factor(ForegroundScale, lightTheme), c: 0.015, h: 215, a: 1},
		},
		{"lightness clamped high", darkTheme, ThemeColor(2, 0, BackgroundScale), oklch{l: 1, c: 0.03, h: 215, a: 1}},
		{"lightness clamped low", darkTheme, ThemeColor(-2, 0, BackgroundScale), oklch{l: 0, c: 0.03, h: 215, a: 1}},
		{"chroma clamped", darkTheme, ThemeColor(0, -1, BackgroundScale), oklch{l: 0.2, c: 0, h: 215, a: 1}},
		{"mode picks light", lightTheme, ModeColor(OKLCH(0.1, 0, 0), OKLCH(0.9, 0, 0)), oklch{l: 0.1, a: 1}},
		{"mode picks dark", darkTheme, ModeColor(OKLCH(0.1, 0, 0), OKLCH(0.9, 0, 0)), oklch{l: 0.9, a: 1}},
		{
			"mode nested", darkTheme,
			ModeColor(ModeColor(OKLCH(0.1, 0, 0), OKLCH(0.2, 0, 0)), ModeColor(OKLCH(0.3, 0, 0), OKLCH(0.4, 0, 0))),
			oklch{l: 0.4, a: 1},
		},
		{
			"mode of theme color", darkTheme, ModeColor(OKLCH(0.1, 0, 0), ThemeColor(0.1, 0, BackgroundScale)),
			oklch{l: 0.2 + 0.1*factor(BackgroundScale, darkTheme), c: 0.03, h: 215, a: 1},
		},
		{"base", darkTheme, backgroundColor, darkTheme.base},
		{"accent", darkTheme, Accent, darkTheme.accent},
		{"accent text", darkTheme, accentTextColor, oklch{l: 0, c: 0.1, h: 10, a: 1}},
		{"headline is achromatic", darkTheme, Headline, oklch{l: 1, c: 0, h: 215, a: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.c.color().colorCoords(tt.th)
			if !near(got.l, tt.want.l) || !near(got.c, tt.want.c) || !near(got.h, tt.want.h) || !near(got.a, tt.want.a) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestLinkHue pins that a link takes the accent's hue at a fixed chroma
// and its lightness from the label scale.
func TestLinkHue(t *testing.T) {
	got := linkColor.color().colorCoords(darkTheme)
	if got.h != 10 || got.c != 0.233 {
		t.Errorf("link = %+v, want accent hue 10 at chroma 0.233", got)
	}
	if l := ThemeColor(0.388, 0, ForegroundScale).color().colorCoords(darkTheme).l; got.l != l {
		t.Errorf("link lightness = %v, want %v from the label scale", got.l, l)
	}
}

// TestSelected pins that the selected background lies between the
// background and the accent (nearer the background when it is mildly
// chromatic), that a chromatic background takes a stronger tint than a
// gray one, and that a strongly chromatic background never overshoots
// the accent.
func TestSelected(t *testing.T) {
	gray, vivid := darkTheme, lightTheme
	gray.base.c = 0
	vivid.base.c = 1
	sel := selectedBackground.color().colorCoords(darkTheme)
	graySel := selectedBackground.color().colorCoords(gray)
	vividSel := selectedBackground.color().colorCoords(vivid)
	if !(sel.l > darkTheme.base.l && sel.l < darkTheme.accent.l) {
		t.Errorf("selected lightness %v not between background %v and accent %v", sel.l, darkTheme.base.l, darkTheme.accent.l)
	}
	if sel.l-darkTheme.base.l > darkTheme.accent.l-sel.l {
		t.Errorf("selected lightness %v nearer the accent than the background", sel.l)
	}
	if graySel.l-gray.base.l >= sel.l-darkTheme.base.l {
		t.Errorf("gray background tinted %v, chromatic background %v, want stronger tint on the chromatic background", graySel.l-gray.base.l, sel.l-darkTheme.base.l)
	}
	if !near(vividSel.l, vivid.accent.l) || !near(vividSel.c, vivid.accent.c) {
		t.Errorf("vivid background selected = %+v, want the accent %+v", vividSel, vivid.accent)
	}
}

func TestMix(t *testing.T) {
	a := oklch{l: 0.2, c: 0.1, h: 90, a: 1}
	b := oklch{l: 0.8, c: 0.1, h: 90, a: 0.5}
	if got := mix(a, b, 0); got != a {
		t.Errorf("mix(a, b, 0) = %+v, want a", got)
	}
	if got := mix(a, b, 1); !near(got.l, b.l) || !near(got.c, b.c) || !near(got.h, b.h) || !near(got.a, b.a) {
		t.Errorf("mix(a, b, 1) = %+v, want b", got)
	}
	if got := mix(a, b, 0.5); !near(got.l, 0.5) || !near(got.c, 0.1) || !near(got.h, 90) || !near(got.a, 0.75) {
		t.Errorf("mix(a, b, 0.5) = %+v", got)
	}
}

func TestCSSColorCoordsPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("colorCoords of a CSS color did not panic")
		}
	}()
	cssColor("red").colorCoords(darkTheme)
}
