package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Grid12(attrs ...attr.Node) html.Element {
	return Box(
		Class("mx-auto w-282 grid grid-cols-12 gap-6"),
		group(attrs...),
	)
}

func Grid9(attrs ...attr.Node) html.Element {
	return Box(
		Class("w-full grid grid-cols-9 gap-6"),
		group(attrs...),
	)
}

func Grid8(attrs ...attr.Node) html.Element {
	return Box(
		Class("w-full grid grid-cols-8 gap-6"),
		group(attrs...),
	)
}
