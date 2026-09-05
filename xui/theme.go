package ui

import (
	"math"

	"ily.dev/act3/xui/internal/sheet"
	"ily.dev/domi"
)

// A theme holds the inputs from which theme colors are derived.
type theme struct {
	base     oklch // always opaque
	accent   oklch
	contrast float64
}

var defaultTheme = theme{
	base:     oklch{l: 0.982, c: 0.0013, h: 100, a: 1},
	accent:   oklch{l: 0.511, c: 0.23, h: 277, a: 1},
	contrast: 30,
}

// Theme sets the base colors used to derive the colors
// of individual views on a page.
//
// The background color determines the color of background elements
// such as menu surfaces, buttons, and the page itself.
// It also determines whether a given view is in light mode or dark mode.
// Foreground content on a light background
// is drawn in dark colors, and vice versa.
//
// The accent color is applied to prominent controls and links.
//
// Contrast influences how far derived colors differ from the background.
// It ranges from 15 (subtle) to 100 (stark).
// Values outside this range are clamped.
func Theme(background, accent Color, contrast float64) Option {
	b := background.color().colorCoords(defaultTheme)
	b.a = 1
	return optionTheme{theme: theme{
		base:     b,
		accent:   accent.color().colorCoords(defaultTheme),
		contrast: min(max(contrast, 15), 100),
	}}
}

// optionTheme is the Option returned by Theme.
// The embedded Option is never set; it only marks the type an Option.
type optionTheme struct {
	domi.Option
	theme theme
}

// styles returns the declarations that establish t on an element:
// the background color, the foreground color for text on it, and the matching color scheme.
func (t theme) styles() sheet.StyleSet {
	var ss sheet.StyleSet
	ss.Set("background-color", t.base.css())
	ss.Set("color", Primary.color().colorCoords(t).css())
	ss.Set("color-scheme", t.base.colorScheme())
	return ss
}

// A ThemeColor is derived from the app's theme.
// See [Theme].
//
// The given color scale includes a base color and scaling factor,
// which depend on the theme's colors and contrast.
// See [ColorScale].
//
// The given ΔL (change in lightness) is multiplied by the scaling factor.
// The given ΔC (change in chroma) is multiplied by the factor's magnitude,
// except under ForegroundScale, where it is added without scaling.
// The scaled values of ΔL and ΔC are then added to the base color
// to yield a resolved color.
// Finally, the lightness and chroma of the resolved color
// are clamped to valid ranges.
//
// Positive ΔL moves toward the contrasting extreme from the base.
// It moves toward white with a dark base and toward black with a light base.
// Positive ΔC moves away from gray.
func ThemeColor(ΔL, ΔC float64, s ColorScale) Color {
	return newColor(themeColor{from: s.base(), l: ΔL, c: ΔC, s: s})
}

// A ModeColor uses lightMode in light mode
// and darkMode in dark mode.
//
// The mode is determined by the theme's background color.
func ModeColor(lightMode, darkMode Color) Color {
	return newColor(modeColor{lightMode.color(), darkMode.color()})
}

// A ColorScale selects a base color from the theme
// (background, foreground, or accent)
// and determines how far a derived color differs from its base color
// as the theme's contrast level rises.
// Each scale is designed for a particular role in the interface.
type ColorScale int

const (
	// ForegroundScale is designed for foreground elements,
	// such as text.
	// Lightness deltas have less effect on backgrounds near middle gray
	// and at higher contrast levels. Chroma deltas are added without scaling.
	ForegroundScale ColorScale = iota

	// ControlScale is designed for the faces of controls,
	// such as buttons.
	ControlScale

	// BorderScale is designed for borders and dividers.
	BorderScale

	// BackgroundScale is designed for the backgrounds of regions and panels.
	BackgroundScale
)

// base returns the base color of a color with scale s.
func (s ColorScale) base() color {
	if s == ForegroundScale {
		return themeForeground{}
	}
	return themeBackground{}
}

// factor returns the factor scaling a theme color's delta under s,
// for a color derived from base on the background bg
// at the given contrast level.
// Its sign follows the lightness of base:
// positive with a dark base, negative with a light base.
// Its magnitude grows with the contrast level,
// except under ForegroundScale, where it declines:
// higher contrast keeps text nearer black or white.
func (s ColorScale) factor(base, bg oklch, contrast float64) float64 {
	k := contrast
	over := max(k-30, 0)
	// Contrast above 30 counts at reduced weight.
	eff := min(k, 30) + over/4
	light := base.isLight()
	var f float64
	switch s {
	case BackgroundScale:
		f = eff / 30
	case ControlScale:
		f = eff / 70
		if light {
			f *= 0.8
		}
	case BorderScale:
		f = (k + over*0.4) / 10
		if light {
			f *= 0.9
		} else {
			f *= 0.8
		}
	case ForegroundScale:
		// Text recedes less on backgrounds near middle gray.
		// Distance ranges from 0.5 at the midpoint to 1 at either extreme.
		distance := 0.5 + math.Abs(bg.l-0.5)
		f = (3 + (100-k)/70) / 4 * distance
	}
	if light {
		f = -f
	}
	return f
}

// A themeColor is a color derived from its base color
// by a delta scaled by a scale.
type themeColor struct {
	from color
	l, c float64
	s    ColorScale
}

func (tc themeColor) colorCoords(t theme) oklch {
	o := tc.from.colorCoords(t)
	f := tc.s.factor(o, t.base, t.contrast)
	// Only lightness has a direction that depends on the mode.
	cf := math.Abs(f)
	if tc.s == ForegroundScale {
		cf = 1
	}
	return newOKLCH(o.l+tc.l*f, o.c+tc.c*cf, o.h, o.a)
}

// A modeColor is one of two colors, chosen by the mode of the theme.
type modeColor struct{ light, dark color }

func (m modeColor) in(t theme) color {
	if t.base.isLight() {
		return m.light
	}
	return m.dark
}

func (m modeColor) colorCoords(t theme) oklch { return m.in(t).colorCoords(t) }

// themeBackground is the theme's background color.
type themeBackground struct{}

func (themeBackground) colorCoords(t theme) oklch { return t.base }

// themeForeground is the theme's foreground color.
type themeForeground struct{}

func (themeForeground) colorCoords(t theme) oklch { return t.base.text() }

// themeAccent is the theme's accent color.
type themeAccent struct{}

func (themeAccent) colorCoords(t theme) oklch { return t.accent }

// textOn returns the foreground color for text on the face of a control
// colored c: black or white, whichever reads better, faintly tinted
// with c's hue.
func textOn(c color) Color { return newColor(textOnColor{c}) }

type textOnColor struct{ on color }

func (tc textOnColor) colorCoords(t theme) oklch {
	const maxChroma = 0.017
	o := tc.on.colorCoords(t).text()
	o.c = min(o.c, maxChroma)
	return o
}

// hoverOf returns the face of a hovered control whose face is c.
func hoverOf(c color) Color {
	return ModeColor(
		newColor(themeColor{c, 0.052, -0.0067, BackgroundScale}),
		newColor(themeColor{c, 0.043, 0.0067, BackgroundScale}),
	)
}

// selectedColor is the background color tinted toward the accent.
// A strongly chromatic background takes a stronger tint.
type selectedColor struct{}

func (selectedColor) colorCoords(t theme) oklch {
	w := 0.05
	if t.base.isLight() {
		w = 0.18
	}
	return mix(t.base, t.accent, min(w*(1+t.base.c/0.09), 1))
}
