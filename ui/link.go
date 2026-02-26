package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Link(url string, attrs ...attr.Node) html.Element {
	return html.A(
		attr.Href(url),
		attr.Class("a$link"),
		attr.EnvAttr("class", linkUnderlineKey, "a$link+underline-auto"),
		group(attrs...),
	)
}

var (
	LinkUnderlineAuto   = html.WithValue(linkUnderlineKey, "a$link+underline-auto")
	LinkUnderlineAlways = html.WithValue(linkUnderlineKey, "a$link+underline-always")
	LinkUnderlineHover  = html.WithValue(linkUnderlineKey, "a$link+underline-hover")
)
