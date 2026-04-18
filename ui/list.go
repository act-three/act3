package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
)

const listController = "list"

var (
	ListID  = Attr("data-list-id-param")
	ListURL = Attr("data-list-url-param")
)

func List(prefix, target string, attrs ...attr.Node) html.Element {
	return ScrollY(
		Class("u-list"),
		group(attrs...),
		stimulus.Controller(listController),
		stimulus.Value(listController, "prefix")(prefix),
		stimulus.Value(listController, "target")(target),
		stimulus.Action("turbo:render@document->list#render"),
	)
}

func ListItems[T any](items []T, f func(T, ...attr.Node) html.Node) html.Node {
	return html.Range(items, func(v T) html.Node {
		return f(v,
			stimulus.Target("list", "item"),
			stimulus.Action("click->list#select"),
		)
	})
}
