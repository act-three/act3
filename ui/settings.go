package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func SettingsPage(title string) html.Element {
	return func(nodes ...html.Node) html.Node {
		return html.Div(attr.Class("u-settings-page"))(
			FlexCol(Gap8)(
				SettingsContent()(Text(title, TextSize6)),
				Group(nodes...),
			),
		)
	}
}

func SettingsContent() html.Element {
	return FlexCol(Class("u-settings-content"))
}

func SettingsSection(title string) html.Node {
	return html.Div(attr.Class("u-settings-section"))(
		Text(title, TextSize3, FontBold),
	)
}

func SettingsSectionDescription(title, description string) html.Node {
	return html.Div(attr.Class("u-settings-section"))(
		Text(title, TextSize3, FontBold),
		Text(description, TextSize2, Class("u-settings-label-description")),
	)
}

func SettingsGroup() html.Element {
	return FlexCol(Class("u-settings-group"))
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

func SettingsControl() html.Element {
	return FlexRow()
}
