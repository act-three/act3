package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

type cardVariant int

const (
	cardSurface cardVariant = iota // default
	cardGhost
	cardClassic
)

var (
	Ghost        = option(cardGhost)
	Classic      = option(cardClassic)
	CardSelected = attr.Attr("data-selected")
)

func Card(attrs ...attr.Node) html.Element {
	a := attr.Group(attrs...)
	tag := "div"
	if a.Has("href") {
		tag = "a"
	}
	return html.Tag(tag)(
		attr.Class("a$card"),
		attr.Class(cardVariantClasses[getOption(attrs, cardSurface)]),
		attr.Class(cardSizeClasses[getOption(attrs, size1)]),
		a,
		attr.Attr("data-slot")("card"),
	)
}

var cardVariantClasses = map[cardVariant]string{
	cardSurface: "a$card+surface",
	cardGhost:   "a$card+ghost",
	cardClassic: "a$card+classic",
}

var cardSizeClasses = map[sizeOption]string{
	size1: "a$card+size-1",
	size2: "a$card+size-2",
	size3: "a$card+size-3",
	size4: "a$card+size-4",
	size5: "a$card+size-5",
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
