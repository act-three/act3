package view

import (
	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
)

const AppCollectionsListItems = "collection-list-items"

func AppCollections(
	title string,
	s []*model.CollectionHead,
	detail ...html.Node,
) (string, html.Node) {
	return title, FlexCol(attr.Class("v-media-page"))(
		ToolbarPrimary()(
			html.Form(
				attr.Method("POST"),
				attr.Action("/-/do/collection-add"),
			)(
				Button(ButtonSurface)(Text("Add Collection")),
			),
		),
		Split()(
			List("/app/collections/", "detail")(
				turbo.StreamTarget(AppCollectionsListItems)(
					ListItems(s, AppCollectionsListItem),
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
	c *model.CollectionHead, attrs ...attr.Node,
) html.Node {
	return Card(CardGhost,
		attr.Group(attrs...),
		ListID(c.ID()),
		ListURL(c.EditorPath()),
	)(
		CardContent()(
			CardTitle()(LiveText(c.TitleField())),
		),
	)
}

func AppCollectionDetail(col *model.Collection) html.Node {
	return FlexCol(Class("v-media-detail"))(
		ScrollY(Class("v-media-detail-body"))(
			SettingsPage()(
				FlexCol(Gap6)(
					SettingsContent()(
						TextNode(Size6)(LiveText(col.TitleField())),
					),

					SettingsGroup()(
						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Title"),
							),
							SettingsTextField("/-/do/collection-set-title", "title", col.Title(), LiveAddr(col.TitleAddr()))(
								Hidden("id", col.ID()),
							),
						),

						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Banner"),
							),
							buttonBannerEdit(
								"/-/dialog/collection-banner/"+col.ID(),
								col.BannerPath(),
							),
						),
					),

					SettingsGroup()(
						SettingsGroupHead()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Movies"),
							),
						),
						html.Div(SettingsGroupItems)(
							html.Range(col.Movies(), func(mo *model.MovieHead) html.Node {
								return SettingsItem()(
									SettingsItemLabel()(
										SettingsItemLabelTitle(mo.Slug()),
									),
								)
							}),
						),
					),

					SettingsGroup()(
						SettingsGroupHead()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Series"),
							),
						),
						html.Div(SettingsGroupItems)(
							html.Range(col.Series(), func(sr *model.SeriesHead) html.Node {
								return SettingsItem()(
									SettingsItemLabel()(
										SettingsItemLabelTitle(sr.Title()),
									),
								)
							}),
						),
					),
				),
			),
		),
	)
}

func AppCollectionBannerDialog(col *model.CollectionHead) html.Node {
	return DialogStream(
		ImageFrame()(
			buttonUpload()(
				Hidden("col-id", col.ID()),
				html.Img(
					attr.Src(col.BannerPath()),
					attr.Style("width: 100%; aspect-ratio: 1000/185; object-fit: cover"),
				),
			),
		),
	)
}

func CollectionChangeBanner(col *model.CollectionHead, oldBannerID string) html.Node {
	oldURL := model.BannerPath(oldBannerID)
	return turbo.SetTargets(`img[src="`+oldURL+`"]`, html.Div(attr.Src(col.BannerPath()))())
}

func CollectionSetSlug(col *model.CollectionHead, oldSlug string) html.Node {
	oldEditorPath := "/app/collections/" + oldSlug
	return Group(
		LiveTextUpdate(col.SlugField()),
		turbo.SetTargets(`[data-list-id-param="`+col.ID()+`"]`,
			html.Div(ListURL(col.EditorPath()))(),
		),
		turbo.URLReplace(oldEditorPath, col.EditorPath()),
	)
}
