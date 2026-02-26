package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Inset(attrs ...attr.Node) html.Element {
	side := getOption[sideOption](attrs, 0)
	return html.Div(
		Class("a$inset"),
		Class(insetSideClasses[side]),
		group(attrs...),
	)
}

var insetSideClasses = map[sideOption]string{
	sideAll:    "a$inset+all",
	sideX:      "a$inset+x",
	sideY:      "a$inset+y",
	sideTop:    "a$inset+top",
	sideBottom: "a$inset+bottom",
	sideLeft:   "a$inset+left",
	sideRight:  "a$inset+right",
}
