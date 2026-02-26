package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Grid12(attrs ...attr.Node) html.Element {
	return Box(
		Class("a$grid-12"),
		group(attrs...),
	)
}

func Grid9(attrs ...attr.Node) html.Element {
	return Box(
		Class("a$grid-9"),
		group(attrs...),
	)
}

func Grid8(attrs ...attr.Node) html.Element {
	return Box(
		Class("a$grid-8"),
		group(attrs...),
	)
}
