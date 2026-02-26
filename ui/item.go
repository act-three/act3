package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	ItemSelected attr.Node = attr.Attr("data-selected")
)

func Item(attrs ...attr.Node) html.Element {
	a := attr.Group(attrs...)
	tag := "div"
	if a.Has("href") {
		tag = "a"
	}
	return html.Tag(tag)(
		attr.Class("a$item"),
		a,
		attr.Attr("data-slot")("item"),
		attr.Attr("data-variant")("default"),
		attr.Attr("data-size")("default"),
	)
}

func ItemMedia(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("a$item-media"),
		attr.Group(attrs...),
		attr.Attr("data-slot")("item-media"),
		attr.Attr("data-variant")("poster"),
	)
}

func ItemContent(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("a$item-content"),
		attr.Group(attrs...),
		attr.Attr("data-slot")("item-content"),
	)
}

func ItemTitle(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("a$item-title"),
		attr.Group(attrs...),
		attr.Attr("data-slot")("item-title"),
	)
}

func ItemDescription(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("a$item-description"),
		attr.Group(attrs...),
		attr.Attr("data-slot")("item-description"),
	)
}
