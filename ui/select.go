package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
)

// SelectValue sets the currently selected item value.
var SelectValue = stimulus.Value("select", "current")

func Select(attrs ...attr.Node) html.Element {
	return func(nodes ...html.Node) html.Node {
		return html.Div(
			attr.Class("u-select"),
			stimulus.Controller("select"),
			stimulus.Action("keydown->select#keydown"),
			attr.Group(attrs...),
		)(nodes...)
	}
}

func SelectTrigger(attrs ...attr.Node) html.Element {
	return func(nodes ...html.Node) html.Node {
		return html.Button(
			attr.Class("u-select-trigger"),
			stimulus.Target("select", "trigger"),
			stimulus.Action("click->select#open"),
			attr.Group(attrs...),
		)(append(nodes, Icon("chevron-down"))...)
	}
}

func SelectContent(attrs ...attr.Node) html.Element {
	return func(nodes ...html.Node) html.Node {
		return html.Div(
			attr.Class("u-select-content"),
			stimulus.Target("select", "content"),
			attr.Group(attrs...),
		)(nodes...)
	}
}

func SelectItem(value string, attrs ...attr.Node) html.Element {
	return func(nodes ...html.Node) html.Node {
		return html.Div(
			attr.Class("u-select-item"),
			stimulus.Target("select", "item"),
			stimulus.Action("click->select#selectItem"),
			attr.Attr("data-select-value-param")(value),
			attr.Attr("tabindex")("-1"),
			attr.Group(attrs...),
		)(nodes...)
	}
}

// SelectLabel renders the text label inside a trigger.
// The initial text is for SSR; the JS controller keeps it
// in sync with the selected item.
func SelectLabel(label string) html.Node {
	return html.Span(
		attr.Class("u-select-label"),
		stimulus.Target("select", "label"),
	)(html.Text(label))
}

// SelectItemSelected marks an item as initially selected.
var SelectItemSelected = attr.Attr("data-selected")

// Variants
var (
	SelectSurface = attr.Class("u-select+surface")
)

// Sizes
var (
	SelectSize1 = attr.Class("u-select+size-1")
	SelectSize2 = attr.Class("u-select+size-2")
	SelectSize3 = attr.Class("u-select+size-3")
)
