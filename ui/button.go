package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Button(attrs ...attr.Node) html.Element {
	a := attr.Group(attrs...)
	tag := "button"
	if a.Has("href") {
		tag = "a"
	}
	return func(nodes ...html.Node) html.Node {
		return html.Tag(tag)(
			attr.Class("u-button"),
			a,
		)(nodes...)
	}
}

// Variants
var (
	ButtonSolid       = attr.Class("u-button+solid")
	ButtonSoft        = attr.Class("u-button+soft")
	ButtonSurface     = attr.Class("u-button+surface")
	ButtonOutline     = attr.Class("u-button+outline")
	ButtonGhost       = attr.Class("u-button+ghost")
	ButtonDestructive = attr.Class("u-button+destructive")
)

// Sizes
var (
	ButtonSize1 = attr.Class("u-button+size-1")
	ButtonSize2 = attr.Class("u-button+size-2")
	ButtonSize3 = attr.Class("u-button+size-3")
	ButtonSize4 = attr.Class("u-button+size-4")
)

// Shapes / radius
var (
	ButtonRadiusNone   = attr.Class("u-button+radius-none")
	ButtonRadiusSmall  = attr.Class("u-button+radius-small")
	ButtonRadiusMedium = attr.Class("u-button+radius-medium")
	ButtonRadiusLarge  = attr.Class("u-button+radius-large")
	ButtonRadiusFull   = attr.Class("u-button+radius-full")
	ButtonCircle       = attr.Class("u-button+radius-circle")
)
