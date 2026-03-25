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
	ButtonSolid   = attr.Attr("data-variant")("solid")
	ButtonSurface = attr.Attr("data-variant")("surface")
	ButtonGhost   = attr.Attr("data-variant")("ghost")
)

// Sizes
var (
	ButtonSize1 = attr.Attr("data-size")("1")
	ButtonSize2 = attr.Attr("data-size")("2")
	ButtonSize3 = attr.Attr("data-size")("3")
	ButtonSize4 = attr.Attr("data-size")("4")
)

// Shapes
var ButtonCircle = attr.Attr("data-radius")("circle")
