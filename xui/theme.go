package ui

import (
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
	base:     oklch{l: 1, a: 1},
	accent:   oklch{l: 0.511, c: 0.23, h: 277, a: 1},
	contrast: 30,
}

// Theme sets the colors from which a [Handler]'s pages derive their theme.
//
// Base is the color of the page background.
// Content on a light base is drawn in dark colors, and vice versa.
// Any opacity of base is ignored.
//
// Accent is the color of prominent controls and links.
//
// Contrast ranges from 15 (subtle) to 100 (stark).
// It sets how far derived colors depart from the base.
// Values outside the range are clamped.
func Theme(base, accent OKLCHColor, contrast float64) Option {
	b := base.coords()
	b.a = 1
	return optionTheme{theme: theme{
		base:     b,
		accent:   accent.coords(),
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
// the base color, the text color read on it, and the matching color scheme.
func (t theme) styles() sheet.StyleSet {
	var ss sheet.StyleSet
	ss.Set("background-color", t.base.colorCSS(t))
	ss.Set("color", t.base.text().colorCSS(t))
	ss.Set("color-scheme", t.base.colorScheme())
	return ss
}
