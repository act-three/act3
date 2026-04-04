package view

import (
	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
)

type CollectionStub struct {
	Slug  string
	Title string
}

var placeholderCollections = []CollectionStub{
	{"the-dark-knight-trilogy", "The Dark Knight Trilogy"},
	{"before-trilogy", "Before Trilogy"},
	{"cornetto-trilogy", "Cornetto Trilogy"},
	{"lord-of-the-rings", "Lord of the Rings"},
	{"star-wars-original", "Star Wars: Original Trilogy"},
}

const AppCollectionsListItems = "collection-list-items"

func AppCollections(
	title string,
	detail ...html.Node,
) (string, html.Node) {
	return title, FlexCol(attr.Class("v-media-page"))(
		ToolbarPrimary()(
			Button(ButtonSurface, Disabled(true))(
				Text("Add Collection"),
			),
		),
		Split()(
			List("/app/collections/", "detail")(
				turbo.StreamTarget(AppCollectionsListItems)(
					ListItems(placeholderCollections, AppCollectionsListItem),
				),
			),
			turbo.Frame("detail", turbo.Advance())(
				expr.IfElse(detail != nil,
					func() html.Node {
						return Group(detail...)
					},
					func() html.Node {
						return Center(Class("v-media-muted"))(
							html.Text("No Collection Selected"),
						)
					},
				),
			),
		),
	)
}

func AppCollectionsListItem(
	c CollectionStub, attrs ...attr.Node,
) html.Node {
	return Card(CardGhost,
		attr.Group(attrs...),
		ListID(c.Slug),
		ListURL("/app/collections/"+c.Slug),
	)(
		CardContent()(
			CardTitle()(html.Text(c.Title)),
		),
	)
}

func AppCollectionDetail(slug string) html.Node {
	var col CollectionStub
	for _, c := range placeholderCollections {
		if c.Slug == slug {
			col = c
			break
		}
	}
	if col.Slug == "" {
		return Center(Class("v-media-muted"))(
			html.Text("Collection not found"),
		)
	}

	return FlexCol(Class("v-media-detail"))(
		ScrollY(Class("v-media-detail-body"))(
			SettingsPage()(
				FlexCol(Gap6)(
					SettingsContent()(
						TextNode(Size6)(html.Text(col.Title)),
					),

					SettingsGroup()(
						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Title"),
							),
							TextNode()(html.Text(col.Title)),
						),

						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Slug"),
							),
							TextNode()(html.Text(col.Slug)),
						),
					),
				),
			),
		),
	)
}
