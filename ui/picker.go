package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func PickerGroup(attrs ...attr.Node) html.Element {
	return FlexCol(Class("u-picker-group"), group(attrs...))
}

func PickerGroupHead(attrs ...attr.Node) html.Element {
	return FlexRow(Class("u-picker-group-head"), group(attrs...))
}

func PickerItem(attrs ...attr.Node) html.Element {
	return FlexRow(Class("u-picker-item"), group(attrs...))
}

func PickerItemLabel(attrs ...attr.Node) html.Element {
	return FlexCol(attr.Style("gap:3px"), group(attrs...))
}
