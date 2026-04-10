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
								col.Banner(), col.BannerAddr(),
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
							collectionMovieItems(col),
						),
					),

					SettingsGroup()(
						SettingsGroupHead()(
							SettingsItemLabel()(
								TextNode(Size2)(
									LiveText(col.SeriesCountField()),
								),
							),
							DialogButton("/-/dialog/collection-series-add/"+col.ID(), ButtonGhost)(
								Text("Add Series"),
							),
						),
						turbo.StreamTarget("collection-"+col.ID()+"-series")(
							collectionSeriesItems(col),
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
					imgLargestAttrs(col.BannerField()),
					attr.Style("width: 100%; aspect-ratio: 1000/185; object-fit: cover"),
				),
			),
		),
	)
}

func CollectionChangeBanner(col *model.CollectionHead) html.Node {
	return liveImgUpdate(col.BannerField())
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
									PosterImg(attr.Style("height: 100%"), imgAttrs(mw.PosterField())),
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

func collectionMovieItemID(colID, movieID string) string {
	return "col-" + colID + "-mo-" + movieID
}

func collectionMovieItems(col *model.Collection) html.Node {
	return html.Range(col.Movies(), func(mo *model.MovieWork) html.Node {
		return SettingsItem(attr.ID(collectionMovieItemID(col.ID(), mo.MovieHead.ID())))(
			FlexRow(attr.Style("align-items:center"), Gap4)(
				SettingsItemLabelIcon()(Icon("line/film-01")),
				SettingsItemLabelTitle(mo.Title()+" ("+mo.Year()+")"),
			),
			html.Form(
				attr.Method("POST"),
				attr.Action("/-/do/collection-movie-remove"),
			)(
				Hidden("col-id", col.ID()),
				Hidden("movie-id", mo.MovieHead.ID()),
				Button(SettingsHover, ButtonGhost)(Text("Remove")),
			),
		)
	})
}

func CollectionMovieAppend(col *model.Collection) html.Node {
	return Group(
		turbo.Update("collection-"+col.ID()+"-movies")(
			collectionMovieItems(col),
		),
		LiveTextUpdate(col.MovieCountField()),
	)
}

func CollectionMovieRemove(col *model.Collection, movieID string) html.Node {
	return Group(
		turbo.Remove(collectionMovieItemID(col.ID(), movieID)),
		LiveTextUpdate(col.MovieCountField()),
	)
}

func AppCollectionSeriesAddDialog(colID string) html.Node {
	return DialogStream(
		FlexCol(Gap2, Class("v-media-dialog"))(
			html.Div(
				attr.Class("v-media-dialog-fixed"),
			)(
				html.Text("Add Series to Collection"),
			),
			html.Form(
				attr.Action("/-/part/collection-series-search"),
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

type CollectionSeriesSearchResult struct {
	Series       *model.SeriesWork
	InCollection bool
}

func AppCollectionSeriesSearchResults(colID string, results []CollectionSeriesSearchResult) html.Node {
	return turbo.Frame("results")(
		FlexCol(Gap4, Class("v-media-detail-body"))(
			html.Range(results, func(r CollectionSeriesSearchResult) html.Node {
				sw := r.Series
				return html.Form(
					attr.Method("POST"),
					attr.Action("/-/do/collection-series-add"),
					stimulus.Action("turbo:submit-end->dialog#close"),
					Disabled(r.InCollection),
				)(
					Hidden("col-id", colID),
					Hidden("series-id", sw.SeriesHead.ID()),
					html.Button(
						attr.Style("all: unset; cursor: pointer; width: 100%"),
					)(
						Card(CardSurface, CardSize3, Class("v-media-search-card"))(
							FlexRow(Gap4, attr.Style("height: 100%"))(
								Inset(InsetSideLeft, Class("v-media-search-poster"))(
									PosterImg(attr.Style("height: 100%"), imgAttrs(sw.PosterField())),
								),
								FlexCol(Gap2)(
									html.If(r.InCollection, func() html.Node {
										return Label("solid/check-circle", "Already in Collection")
									}),
									TextNode(TextBold)(html.Text(sw.Title())),
								),
							),
						),
					),
				)
			}),
		),
	)
}

func collectionSeriesItemID(colID, seriesID string) string {
	return "col-" + colID + "-sr-" + seriesID
}

func collectionSeriesItems(col *model.Collection) html.Node {
	return html.Range(col.Series(), func(sw *model.SeriesWork) html.Node {
		return SettingsItem(attr.ID(collectionSeriesItemID(col.ID(), sw.SeriesHead.ID())))(
			FlexRow(attr.Style("align-items:center"), Gap4)(
				SettingsItemLabelIcon()(Icon("line/tv-03")),
				SettingsItemLabelTitle(sw.Title()),
			),
			html.Form(
				attr.Method("POST"),
				attr.Action("/-/do/collection-series-remove"),
			)(
				Hidden("col-id", col.ID()),
				Hidden("series-id", sw.SeriesHead.ID()),
				Button(SettingsHover, ButtonGhost)(Text("Remove")),
			),
		)
	})
}

func CollectionSeriesAppend(col *model.Collection) html.Node {
	return Group(
		turbo.Update("collection-"+col.ID()+"-series")(
			collectionSeriesItems(col),
		),
		LiveTextUpdate(col.SeriesCountField()),
	)
}

func CollectionSeriesRemove(col *model.Collection, seriesID string) html.Node {
	return Group(
		turbo.Remove(collectionSeriesItemID(col.ID(), seriesID)),
		LiveTextUpdate(col.SeriesCountField()),
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
