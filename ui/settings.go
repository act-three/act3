package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

var SettingsHover = Class("u-settings-item-hover")

func SettingsPage() domi.Element {
	return func(nodes ...domi.Node) domi.Node {
		return html.Div(Class("u-settings-page"))(
			FlexCol()(
				Group(nodes...),
			),
		)
	}
}

func SettingsContent() domi.Element {
	return FlexCol(Class("u-settings-content"))
}

func SettingsGroup(attrs ...domi.Attr) domi.Element {
	return FlexCol(Class("u-settings-group"), group(attrs...))
}

func SettingsGroupHead(attrs ...domi.Attr) domi.Element {
	return FlexRow(Class("u-settings-group-head"), group(attrs...))
}

func SettingsItem(attrs ...domi.Attr) domi.Element {
	return FlexRow(Class("u-settings-item"), group(attrs...))
}

func SettingsItemLabel(attrs ...domi.Attr) domi.Element {
	return FlexCol(Style("gap:3px"), group(attrs...))
}

func SettingsItemLabelTitle(s string) domi.Node {
	return Text(s, Size2)
}

func SettingsItemLabelDescription(s string) domi.Node {
	return Text(s, Size1, Class("u-settings-label-description"))
}

func SettingsItemLabelIcon() domi.Element {
	return FlexRow(Class("u-settings-label-icon"))
}
