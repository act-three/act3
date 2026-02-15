package list

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
)

const (
	controller = "list"
)

var (
	ID  = attr.Attr("data-list-id-param")
	URL = attr.Attr("data-list-url-param")
)

func List(prefix, target string, attrs ...attr.Node) html.Element {
	return ScrollArea(
		attr.Class("p-2"),
		attr.Group(attrs...),
		stimulus.Controller(controller),
		stimulus.Value(controller, "prefix")(prefix),
		stimulus.Value(controller, "target")(target),
		stimulus.Action("turbo:render@document->list#render"),
	)
}

func Items[T any](items []T, f func(T, ...attr.Node) html.Node) html.Node {
	return html.Range(items, func(v T) html.Node {
		return f(v,
			stimulus.Target("list", "item"),
			stimulus.Action("click->list#select"),
		)
	})
}
