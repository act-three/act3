package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Button(attrs ...attr.Node) html.Element {
	a := group(attrs...)
	tag := "button"
	if a.Has("href") {
		tag = "a"
	}
	return func(nodes ...html.Node) html.Node {
		return html.Tag(tag)(
			Class("u-button"),
			a,
		)(nodes...)
	}
}

// Variants — set data-button on the button or any ancestor to inherit.
var (
	ButtonSolid   = Attr("data-button")("solid")
	ButtonSurface = Attr("data-button")("surface")
	ButtonGhost   = Attr("data-button")("ghost")
)

// Sizes — set data-button-size on the button or any ancestor to inherit.
var (
	ButtonSize1 = Attr("data-button-size")("1")
	ButtonSize2 = Attr("data-button-size")("2")
	ButtonSize3 = Attr("data-button-size")("3")
	ButtonSize4 = Attr("data-button-size")("4")
)

// Shapes
var ButtonCircle = Attr("data-radius")("circle")

// ButtonSelected highlights a surface button as selected.
var ButtonSelected = Attr("data-selected")
