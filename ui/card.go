package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	CardSurface = Class("a$card+surface")
	CardGhost   = Class("a$card+ghost")
	CardClassic = Class("a$card+classic")
)

var (
	CardSelected = attr.Attr("data-selected")
)

var (
	CardSize1 = Class("a$card+size-1")
	CardSize2 = Class("a$card+size-2")
	CardSize3 = Class("a$card+size-3")
	CardSize4 = Class("a$card+size-4")
	CardSize5 = Class("a$card+size-5")
)

func Card(attrs ...attr.Node) html.Element {
	a := attr.Group(attrs...)
	tag := "div"
	if a.Has("href") {
		tag = "a"
	}
	return html.Tag(tag)(
		attr.Class("a$card"),
		a,
		attr.Attr("data-slot")("card"),
	)
}

func CardMedia(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("a$card-media"),
		attr.Group(attrs...),
		attr.Attr("data-slot")("card-media"),
	)
}

func CardContent(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("a$card-content"),
		attr.Group(attrs...),
		attr.Attr("data-slot")("card-content"),
	)
}

func CardTitle(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("a$card-title"),
		attr.Group(attrs...),
		attr.Attr("data-slot")("card-title"),
	)
}

func CardDescription(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("a$card-description"),
		attr.Group(attrs...),
		attr.Attr("data-slot")("card-description"),
	)
}
