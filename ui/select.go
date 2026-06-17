package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/ui/stimulus"
)

// SelectValue sets the currently selected item value.
var SelectValue = stimulus.Value("select", "current")

func Select(attrs ...domi.Attr) domi.Element {
	return func(nodes ...domi.Node) domi.Node {
		return html.Output(
			Class("u-select"),
			attr.Role("presentation"),
			stimulus.Controller("select"),
			stimulus.Action("keydown->select#keydown"),
			group(attrs...),
		)(nodes...)
	}
}

func SelectTrigger(attrs ...domi.Attr) domi.Element {
	return func(nodes ...domi.Node) domi.Node {
		return html.Button(
			attr.Type("button"),
			Class("u-select-trigger"),
			stimulus.Target("select", "trigger"),
			group(attrs...),
		)(append(nodes, Icon("line/chevron-selector-vertical"))...)
	}
}

func SelectContent(attrs ...domi.Attr) domi.Element {
	return func(nodes ...domi.Node) domi.Node {
		return html.Div(
			Class("u-select-content"),
			stimulus.Target("select", "content"),
			stimulus.Action("toggle->select#toggled"),
			attr.Popover("auto"),
			group(attrs...),
		)(nodes...)
	}
}

func SelectItem(value string, attrs ...domi.Attr) domi.Element {
	return func(nodes ...domi.Node) domi.Node {
		return html.Div(
			Class("u-select-item"),
			stimulus.Target("select", "item"),
			stimulus.Action("click->select#selectItem"),
			Attr("data-select-value-param")(value),
			Attr("tabindex")("-1"),
			group(attrs...),
		)(nodes...)
	}
}

// SelectLabel renders the text label inside a trigger.
// The initial text is for SSR; the JS controller keeps it
// in sync with the selected item.
func SelectLabel(label string) domi.Node {
	return html.Span(
		Class("u-select-label"),
		stimulus.Target("select", "label"),
	)(domi.Text(label))
}

// SelectItemSelected marks an item as initially selected.
var SelectItemSelected = Attr("data-selected")("")

var (
	SelectSize1 = Attr("data-select-size")("1")
	SelectSize2 = Attr("data-select-size")("2")
	SelectSize3 = Attr("data-select-size")("3")
)
