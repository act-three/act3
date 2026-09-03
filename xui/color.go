package ui

import "cmp"

// Theme color tokens.
var (
	Muted  Color = CSSColor("var(--ui-color-muted)")
	Accent Color = CSSColor("var(--ui-color-accent)")
	Danger Color = CSSColor("var(--ui-color-danger)")

	borderColor  Color = CSSColor("var(--ui-color-border)")
	surfaceColor Color = CSSColor("var(--ui-color-surface)")
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

// CSSColor returns the color given by expr.
// It can be any valid CSS color expression,
// such as "#fff" or "var(--my-color)".
func CSSColor(expr string) Color {
	c := cssColor(expr)
	return colorView{base{nodeColor(c)}, c}
}

// A color is the internal representation of a color.
// Unlike Color, it is not a View.
type color interface {
	// colorCSS returns a representation of the receiver
	// as a CSS color expression.
	colorCSS() string
}

type cssColor string

func (c cssColor) colorCSS() string {
	return string(c)
}

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
