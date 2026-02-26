package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	CardSurface = Class("u-card+surface")
	CardGhost   = Class("u-card+ghost")
	CardClassic = Class("u-card+classic")
)

var (
	CardSelected = attr.Attr("data-selected")
)

var (
	CardSize1 = Class("u-card+size-1")
	CardSize2 = Class("u-card+size-2")
	CardSize3 = Class("u-card+size-3")
	CardSize4 = Class("u-card+size-4")
	CardSize5 = Class("u-card+size-5")
)

func Card(attrs ...attr.Node) html.Element {
	a := attr.Group(attrs...)
	tag := "div"
	if a.Has("href") {
		tag = "a"
	}
	return html.Tag(tag)(
		attr.Class("u-card"),
		a,
	)
}

func CardMedia(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("u-card-media"),
		attr.Group(attrs...),
	)
}

func CardContent(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("u-card-content"),
		attr.Group(attrs...),
	)
}

func CardTitle(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("u-card-title"),
		attr.Group(attrs...),
	)
}

func CardDescription(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("u-card-description"),
		attr.Group(attrs...),
	)
}
