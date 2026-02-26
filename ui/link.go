package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Link(url string, attrs ...attr.Node) html.Element {
	return html.A(
		attr.Href(url),
		attr.Class("u-link"),
		attr.EnvAttr("class", linkUnderlineKey, "u-link+underline-auto"),
		group(attrs...),
	)
}

var (
	LinkUnderlineAuto   = html.WithValue(linkUnderlineKey, "u-link+underline-auto")
	LinkUnderlineAlways = html.WithValue(linkUnderlineKey, "u-link+underline-always")
	LinkUnderlineHover  = html.WithValue(linkUnderlineKey, "u-link+underline-hover")
)
