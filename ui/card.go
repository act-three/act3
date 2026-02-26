package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Card(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("a$card"),
		attr.Class(cardSizeClasses[getOption[sizeOption](attrs, 1)]),
		group(attrs...),
	)
}

var cardSizeClasses = map[sizeOption]string{
	size1: "a$card+size-1",
	size2: "a$card+size-2",
	size3: "a$card+size-3",
}
