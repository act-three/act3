package ui

import (
	"cmp"
	"fmt"
)

// Theme color tokens.
var (
	Muted  Color = OKLCH(0.544, 0.035, 265)
	Accent Color = colorView{base{nodeColor(accentColor{})}, accentColor{}}
	Danger Color = OKLCH(0.576, 0.209, 29.5)

	borderColor  Color = OKLCH(0.927, 0.007, 261)
	surfaceColor Color = colorView{base{nodeColor(baseColor{})}, baseColor{}}
)

// A Color represents a color.
//
// Some view modifiers take a Color as an argument.
// For instance, [View.Foreground] sets the color of foreground elements like text.
//
// A Color is also a View.
// When used as a View, a Color expands to fill the available space.
type Color interface {
	View

	// color always returns a non-nil color.
	color() color
}

// An OKLCHColor is a Color with known coordinates
// in the OKLCH color space.
type OKLCHColor interface {
	Color

	coords() oklch
}

// OKLCH returns the color with the given coordinates
// in the OKLCH color space.
//
//   - Lightness (L) ranges from 0 (black) to 1 (white).
//   - Chroma (C) is 0 for gray. Larger values are more colorful.
//     Displayable colors have chroma of at most about 0.4.
//   - Hue (h) is an angle in degrees.
//
// Lightness and chroma are clamped to their valid ranges.
func OKLCH(L, C, h float64) OKLCHColor {
	return OKLCHA(L, C, h, 1)
}

// OKLCHA returns the color with the given coordinates
// in the OKLCH color space with opacity channel α.
//
//   - Lightness (L) ranges from 0 (black) to 1 (white).
//   - Chroma (C) is 0 for gray. Larger values are more colorful.
//     Displayable colors have chroma of at most about 0.4.
//   - Hue (h) is an angle in degrees.
//   - Opacity (α) ranges from 0 (transparent) to 1 (opaque).
//
// Lightness, chroma, and opacity are clamped to their valid ranges.
func OKLCHA(L, C, h, α float64) OKLCHColor {
	col := oklch{
		l: min(max(L, 0), 1),
		c: max(C, 0),
		h: h,
		a: min(max(α, 0), 1),
	}
	return viewOKLCH{colorView{base{nodeColor(col)}, col}, col}
}

// CSSColor returns the color given by expr.
// It can be any valid CSS color expression,
// such as "#fff" or "var(--my-color)".
func CSSColor(expr string) Color {
	c := cssColor(expr)
	return colorView{base{nodeColor(c)}, c}
}

// A color is the internal representation of a color.
// Unlike Color, it is not a View.
// A color may depend on the theme it is used in,
// so it resolves only when a box is lowered.
type color interface {
	// colorCSS returns the receiver, resolved in t,
	// as a CSS color expression.
	colorCSS(t theme) string

	// colorCoords returns the receiver's OKLCH coordinates, resolved in t.
	// It panics if the receiver's coordinates are unknown.
	colorCoords(t theme) oklch
}

type oklch struct{ l, c, h, a float64 }

func (c oklch) colorCSS(theme) string {
	if c.a < 1 {
		return fmt.Sprintf("oklch(%g %g %g / %g)", c.l, c.c, c.h, c.a)
	}
	return fmt.Sprintf("oklch(%g %g %g)", c.l, c.c, c.h)
}

func (c oklch) colorCoords(theme) oklch { return c }

// lightThreshold is the lightness above which a color is light.
const lightThreshold = 0.57

// isLight reports whether c is a light color.
// Foreground content on a light color background
// is drawn in dark colors, and vice versa.
func (c oklch) isLight() bool { return c.l > lightThreshold }

// colorScheme returns a suitable CSS color-scheme value
// for a background color of c.
func (c oklch) colorScheme() string {
	if c.isLight() {
		return "light"
	}
	return "dark"
}

// text returns the base foreground color for a background color of c.
// Black when c is light or white when c is dark,
// tinted with c's hue at half its chroma.
func (c oklch) text() oklch {
	t := oklch{l: 1, c: c.c / 2, h: c.h, a: 1}
	if c.isLight() {
		t.l = 0
	}
	return t
}

type cssColor string

func (c cssColor) colorCSS(theme) string { return string(c) }

func (c cssColor) colorCoords(theme) oklch {
	panic(fmt.Sprintf("ui: coordinates of CSS color %q are unknown", string(c)))
}

// baseColor is the theme's base color.
type baseColor struct{}

func (baseColor) colorCSS(t theme) string   { return t.base.colorCSS(t) }
func (baseColor) colorCoords(t theme) oklch { return t.base }

// accentColor is the theme's accent color.
type accentColor struct{}

func (accentColor) colorCSS(t theme) string   { return t.accent.colorCSS(t) }
func (accentColor) colorCoords(t theme) oklch { return t.accent }

type colorView struct {
	base
	c color
}

func (c colorView) color() color { return c.c }

// Font has no effect because color contains no text.
// This overrides the embedded method Font
// to avoid emitting useless style declarations.
func (c colorView) Font(FontSize) View { return c }

// Foreground has no effect because color contains no foreground elements.
// This overrides the embedded method Foreground
// to avoid emitting useless style declarations.
func (c colorView) Foreground(Color) View { return c }

type viewOKLCH struct {
	colorView
	oklch oklch
}

func (c viewOKLCH) coords() oklch { return c.oklch }

// nodeColor paints a solid color.
func nodeColor(c color) node {
	return func(env environment) box {
		env.tag = cmp.Or(env.tag, "ui-color")
		env.bg = append(env.bg, term[color]{value: c})
		return build(env, plan{
			fills: Horizontal | Vertical,
			ideal: rect{width: newSize(10), height: newSize(10)},
		})
	}
}
