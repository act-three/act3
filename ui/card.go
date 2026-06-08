package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

var (
	CardSurface = Attr("data-card")("surface")
	CardGhost   = Attr("data-card")("ghost")
	CardClassic = Attr("data-card")("classic")
)

var CardSelected = Attr("data-selected")("")

var (
	CardSize1 = Attr("data-card-size")("1")
	CardSize2 = Attr("data-card-size")("2")
	CardSize3 = Attr("data-card-size")("3")
	CardSize4 = Attr("data-card-size")("4")
	CardSize5 = Attr("data-card-size")("5")
)

func Card(attrs ...domi.Attr) domi.Element {
	return html.Div(
		Class("u-card"),
		group(attrs...),
	)
}

// CardLink renders a card as an <a> with the given href.
func CardLink(href string, attrs ...domi.Attr) domi.Element {
	return html.A(
		Class("u-card"),
		Href(href),
		group(attrs...),
	)
}

func CardMedia(attrs ...domi.Attr) domi.Element {
	return html.Div(
		Class("u-card-media"),
		group(attrs...),
	)
}

func CardContent(attrs ...domi.Attr) domi.Element {
	return html.Div(
		Class("u-card-content"),
		group(attrs...),
	)
}

func CardTitle(attrs ...domi.Attr) domi.Element {
	return html.Div(
		Class("u-card-title"),
		group(attrs...),
	)
}

func CardDescription(attrs ...domi.Attr) domi.Element {
	return html.Div(
		Class("u-card-description"),
		group(attrs...),
	)
}
