package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func SettingsPage() html.Element {
	return func(nodes ...html.Node) html.Node {
		return html.Div(attr.Class("u-settings-page"))(
			FlexCol(Gap8)(
				Group(nodes...),
			),
		)
	}
}

func SettingsContent() html.Element {
	return FlexCol(Class("u-settings-content"))
}

func SettingsGroup() html.Element {
	return FlexCol(Class("u-settings-group"))
}

func SettingsGroupHead() html.Element {
	return FlexRow(Class("u-settings-group-head"))
}

func SettingsItem() html.Element {
	return FlexRow(Class("u-settings-item"))
}

func SettingsItemLabel() html.Element {
	return FlexCol(attr.Style("gap:3px"))
}

func SettingsItemLabelTitle(s string) html.Node {
	return Text(s, TextSize2)
}

func SettingsItemLabelDescription(s string) html.Node {
	return Text(s, TextSize1, Class("u-settings-label-description"))
}

var SettingsGroupItems = Class("u-settings-group-items")
