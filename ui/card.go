package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	CardSurface = Attr("data-card")("surface")
	CardGhost   = Attr("data-card")("ghost")
	CardClassic = Attr("data-card")("classic")
)

var CardSelected = Attr("data-selected")

var (
	CardSize1 = Attr("data-card-size")("1")
	CardSize2 = Attr("data-card-size")("2")
	CardSize3 = Attr("data-card-size")("3")
	CardSize4 = Attr("data-card-size")("4")
	CardSize5 = Attr("data-card-size")("5")
)

func Card(attrs ...attr.Node) html.Element {
	a := group(attrs...)
	tag := "div"
	if a.Has("href") {
		tag = "a"
	}
	return html.Tag(tag)(
		Class("u-card"),
		a,
	)
}

func CardMedia(attrs ...attr.Node) html.Element {
	return html.Div(
		Class("u-card-media"),
		group(attrs...),
	)
}

func CardContent(attrs ...attr.Node) html.Element {
	return html.Div(
		Class("u-card-content"),
		group(attrs...),
	)
}

func CardTitle(attrs ...attr.Node) html.Element {
	return html.Div(
		Class("u-card-title"),
		group(attrs...),
	)
}

func CardDescription(attrs ...attr.Node) html.Element {
	return html.Div(
		Class("u-card-description"),
		group(attrs...),
	)
}
