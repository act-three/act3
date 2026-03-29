package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func SettingsPage() html.Element {
	return func(nodes ...html.Node) html.Node {
		return html.Div(attr.Class("u-settings-page"))(
			FlexCol()(
				Group(nodes...),
			),
		)
	}
}

func SettingsContent() html.Element {
	return FlexCol(Class("u-settings-content"))
}

func SettingsGroup(attrs ...attr.Node) html.Element {
	return FlexCol(Class("u-settings-group"), group(attrs...))
}

func SettingsGroupHead(attrs ...attr.Node) html.Element {
	return FlexRow(Class("u-settings-group-head"), group(attrs...))
}

func SettingsItem(attrs ...attr.Node) html.Element {
	return FlexRow(Class("u-settings-item"), group(attrs...))
}

func SettingsItemLabel(attrs ...attr.Node) html.Element {
	return FlexCol(attr.Style("gap:3px"), group(attrs...))
}

func SettingsItemLabelTitle(s string) html.Node {
	return Text(s, Size2)
}

func SettingsItemLabelDescription(s string) html.Node {
	return Text(s, Size1, Class("u-settings-label-description"))
}

func SettingsItemLabelIcon() html.Element {
	return FlexRow(Class("u-settings-label-icon"))
}

var SettingsGroupItems = Class("u-settings-group-items")
