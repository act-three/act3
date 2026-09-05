package ui

import (
	"cmp"
	"fmt"
	"math"
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
	c := newOKLCH(L, C, h, α)
	return viewOKLCH{newColor(c), c}
}

// CSSColor returns the color given by expr.
// It can be any valid CSS color expression,
// such as "#fff" or "var(--my-color)".
func CSSColor(expr string) Color {
	return newColor(cssColor(expr))
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

func newOKLCH(l, c, h, a float64) oklch {
	const maxChroma = 0.5 // Beyond the reach of any display.
	return oklch{
		l: min(max(l, 0), 1),
		c: min(max(c, 0), maxChroma),
		h: h,
		a: min(max(a, 0), 1),
	}
}

// mix returns the color a fraction w of the way from a to b.
func mix(a, b oklch, w float64) oklch {
	lerp := func(x, y float64) float64 { return x + (y-x)*w }
	la, aa, ba := a.lab()
	lb, ab, bb := b.lab()
	return fromLab(lerp(la, lb), lerp(aa, ab), lerp(ba, bb), lerp(a.a, b.a))
}

// fromLab returns the color with the given OKLab coordinates and opacity.
func fromLab(l, a, b, alpha float64) oklch {
	h := math.Atan2(b, a) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return oklch{l: l, c: math.Hypot(a, b), h: h, a: alpha}
}

func (c oklch) colorCSS(theme) string {
	if c.a < 1 {
		return fmt.Sprintf("oklch(%.4g %.4g %.4g / %.4g)", c.l, c.c, c.h, c.a)
	}
	return fmt.Sprintf("oklch(%.4g %.4g %.4g)", c.l, c.c, c.h)
}

func (c oklch) colorCoords(theme) oklch { return c }

// isLight reports whether c is a light color.
// Foreground content on a light color background
// is drawn in dark colors, and vice versa.
func (c oklch) isLight() bool {
	const threshold = 0.57
	return c.l > threshold
}

// colorScheme returns a suitable CSS color-scheme value
// for a background color of c.
func (c oklch) colorScheme() string {
	if c.isLight() {
		return "light"
	}
	return "dark"
}

// text returns a suitable foreground color for a background color of c.
// Black when c is light or white when c is dark,
// tinted with c's hue at half its chroma.
func (c oklch) text() oklch {
	t := oklch{l: 1, c: c.c / 2, h: c.h, a: 1}
	if c.isLight() {
		t.l = 0
	}
	return t
}

// lab returns c's coordinates in the OKLab color space.
func (c oklch) lab() (l, a, b float64) {
	h := c.h * math.Pi / 180
	return c.l, c.c * math.Cos(h), c.c * math.Sin(h)
}

// A compositeColor takes its lightness, chroma, hue, and opacity
// from four other colors.
type compositeColor struct{ l, c, h, a color }

func (cc compositeColor) colorCSS(t theme) string { return cc.colorCoords(t).colorCSS(t) }

func (cc compositeColor) colorCoords(t theme) oklch {
	return oklch{
		l: cc.l.colorCoords(t).l,
		c: cc.c.colorCoords(t).c,
		h: cc.h.colorCoords(t).h,
		a: cc.a.colorCoords(t).a,
	}
}

type cssColor string

func (c cssColor) colorCSS(theme) string { return string(c) }

func (c cssColor) colorCoords(theme) oklch {
	panic(fmt.Sprintf("ui: coordinates of CSS color %q are unknown", string(c)))
}

type colorView struct {
	base
	c color
}

func newColor(c color) colorView {
	return colorView{base{nodeColor(c)}, c}
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
