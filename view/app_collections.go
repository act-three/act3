package view

import (
	"fmt"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
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
								TextNode(Size2)(
									//fmt.Sprintf("%d Movies", len(col.Movies())),
									LiveText(col.MovieCountField()),
								),
							),
							DialogButton("/-/dialog/collection-movie-add/"+col.ID(), ButtonGhost)(
								Text("Add Movie"),
							),
						),
						turbo.StreamTarget("collection-"+col.ID()+"-movies")(
							html.Range(col.Movies(), collectionMovieItem),
						),
					),

					SettingsGroup()(
						SettingsGroupHead()(
							SettingsItemLabel()(
								SettingsItemLabelTitle(fmt.Sprintf("%d Series", len(col.Series()))),
							),
							Button(ButtonGhost)(Text("Add Series")),
						),
						turbo.StreamTarget("collection-"+col.ID()+"-series")(
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

func AppCollectionMovieAddDialog(colID string) html.Node {
	return DialogStream(
		FlexCol(Gap2, Class("v-media-dialog"))(
			html.Div(
				attr.Class("v-media-dialog-fixed"),
			)(
				html.Text("Add Movie to Collection"),
			),
			html.Form(
				attr.Action("/-/part/collection-movie-search"),
				attr.Attr("data-turbo-frame")("results"),
			)(
				Hidden("col-id", colID),
				InputText(
					attr.Attr("autofocus"),
					attr.Class("v-media-dialog-fixed"),
					attr.Name("q"),
				),
			),
			html.Div(
				attr.Class("v-media-dialog-results"),
			)(
				turbo.Frame("results")(Spinner(Class("v-media-dialog-spinner"))),
			),
		),
	)
}

type CollectionMovieSearchResult struct {
	Movie        *model.MovieWork
	InCollection bool
}

func AppCollectionMovieSearchResults(colID string, results []CollectionMovieSearchResult) html.Node {
	return turbo.Frame("results")(
		FlexCol(Gap4, Class("v-media-detail-body"))(
			html.Range(results, func(r CollectionMovieSearchResult) html.Node {
				mw := r.Movie
				return html.Form(
					attr.Method("POST"),
					attr.Action("/-/do/collection-movie-add"),
					stimulus.Action("turbo:submit-end->dialog#close"),
					Disabled(r.InCollection),
				)(
					Hidden("col-id", colID),
					Hidden("movie-id", mw.MovieHead.ID()),
					html.Button(
						attr.Style("all: unset; cursor: pointer; width: 100%"),
					)(
						Card(CardSurface, CardSize3, Class("v-media-search-card"))(
							FlexRow(Gap4, attr.Style("height: 100%"))(
								Inset(InsetSideLeft, Class("v-media-search-poster"))(
									PosterImg(attr.Style("height: 100%"), attr.Src(mw.PosterPath())),
								),
								FlexCol(Gap2)(
									html.If(r.InCollection, func() html.Node {
										return Label("solid/check-circle", "Already in Collection")
									}),
									TextNode(TextBold)(html.Text(mw.Title())),
									TextNode()(html.Text(mw.MovieEditionHead.Year())),
									TextNode()(html.Text(fmt.Sprintf("%d min", mw.MovieEditionHead.Runtime()))),
								),
							),
						),
					),
				)
			}),
		),
	)
}

func collectionMovieItem(mo *model.MovieHead) html.Node {
	return SettingsItem()(
		SettingsItemLabel()(
			SettingsItemLabelTitle(mo.Slug()),
		),
	)
}

func CollectionMovieAppend(col *model.Collection, mo *model.MovieHead) html.Node {
	return Group(
		turbo.Append("collection-"+col.ID()+"-movies",
			collectionMovieItem(mo),
		),
		LiveTextUpdate(col.MovieCountField()),
	)
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
