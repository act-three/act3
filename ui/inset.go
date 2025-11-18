package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Inset(attrs ...attr.Node) html.Element {
	class := "rounded-(--inset-rounded) p-(--inset-padding)"
	switch getOption[sideOption](attrs, 0) {
	case sideX:
		class = "mx-(--inset-padding)"
	case sideY:
		class = "my-(--inset-padding)"
	case sideTop:
		class = "rounded-t-(--inset-rounded) mt-(--inset-padding) mx-(--inset-padding)"
	case sideBottom:
		class = "rounded-b-(--inset-rounded) mb-(--inset-padding) mx-(--inset-padding)"
	case sideLeft:
		class = "rounded-l-(--inset-rounded) ml-(--inset-padding) my-(--inset-padding)"
	case sideRight:
		class = "rounded-r-(--inset-rounded) mr-(--inset-padding) my-(--inset-padding)"
	}
	return html.Div(
		Class(class),
		Class("overflow-hidden"),
		group(attrs...),
	)
}
