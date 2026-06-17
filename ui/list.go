package ui

import (
	"ily.dev/domi"
)

// List renders the scrollable list pane of a list-detail layout.
// Build its children with [ListItems].
func List(attrs ...domi.Attr) domi.Element {
	return ScrollY(
		Class("u-list"),
		group(attrs...),
	)
}

// ListItems renders the items of a list-detail list. Each item
// should render as a link to its detail page (see [CardLink]);
// navigation and selection need no client state. selected reports
// whether an item is the one currently shown in the detail pane; it
// receives [CardSelected].
func ListItems[T any](items []T, selected func(T) bool, f func(T, ...domi.Attr) domi.Node) domi.Node {
	return rangeNodes(items, func(v T) domi.Node {
		if selected(v) {
			return f(v, CardSelected)
		}
		return f(v)
	})
}
