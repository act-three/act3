package ui

import "ily.dev/domi"

func PickerGroup(attrs ...domi.Attr) domi.Element {
	return FlexCol(Class("u-picker-group"), group(attrs...))
}

func PickerGroupHead(attrs ...domi.Attr) domi.Element {
	return FlexRow(Class("u-picker-group-head"), group(attrs...))
}

func PickerItem(attrs ...domi.Attr) domi.Element {
	return FlexRow(Class("u-picker-item"), group(attrs...))
}

func PickerItemLabel(attrs ...domi.Attr) domi.Element {
	return FlexCol(Style("gap:3px"), group(attrs...))
}
