package view

import (
	"fmt"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/model"
	"ily.dev/act3/model/kind"
	"ily.dev/act3/msg"
	. "ily.dev/act3/ui"
)

func AppCollections(
	s []*model.CollectionHead,
	selected *model.Collection,
	notFound bool,
) (title string, n domi.Node) {
	return "Collections", FlexCol(Class("v-media-page"))(
		ToolbarPrimary()(
			Button(onClick(&msg.CollectionAdd{}), ButtonSurface)(Text("Add Collection")),
		),
		Split()(
			List()(
				ListItems(s, func(c *model.CollectionHead) bool {
					return selected != nil && c.ID() == selected.ID()
				}, AppCollectionsListItem),
			),
			appCollectionSelection(selected, notFound),
		),
	)
}

func appCollectionSelection(selected *model.Collection, notFound bool) domi.Node {
	switch {
	case selected != nil:
		return appCollectionDetail(selected)
	case notFound:
		return Center(Class("v-media-muted"))(
			domi.Text("Not Found"),
		)
	}
	return Center(Class("v-media-muted"))(
		domi.Text("No Collection Selected"),
	)
}

func AppCollectionsListItem(
	c *model.CollectionHead, attrs ...domi.Attr,
) domi.Node {
	return CardLink(c.EditorPath(), CardGhost,
		group(attrs...),
	)(
		CardContent()(
			CardTitle()(domi.Text(c.Title())),
		),
	)
}

func appCollectionDetail(col *model.Collection) domi.Node {
	return FlexCol(Class("v-media-detail"))(
		ScrollY(Class("v-media-detail-body"))(
			SettingsPage()(
				FlexCol(Gap6)(
					SettingsContent()(
						TextNode(Size6)(domi.Text(col.Title())),
						Box()(
							Link(
								col.TheaterPath(),
							)(Text("View in Theater", Size3,
								Style("display: inline-block"),
							)),
						),
					),

					SettingsGroup()(
						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Title"),
							),
							SettingsTextField(col.Title(), func(v string) msg.Msg {
								return &msg.CollectionSetTitle{ID: col.ID(), Title: v}
							}),
						),

						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Banner"),
							),
							buttonImageEdit(
								&msg.ImageDialogOpen{Kind: kind.Collection{}, ID: col.ID()},
								col.Banner(),
								AspectBanner,
							),
						),
					),

					SettingsGroup()(
						SettingsGroupHead()(
							SettingsItemLabel()(
								TextNode(Size2)(
									//fmt.Sprintf("%d Movies", len(col.Movies())),
									domi.Textf("%d Movies", len(col.Movies())),
								),
							),
							Button(onClick(&msg.CollectionMovieAddOpen{CollectionID: col.ID()}), ButtonGhost)(
								Text("Add Movie"),
							),
						),
						collectionMovieItems(col),
					),

					SettingsGroup()(
						SettingsGroupHead()(
							SettingsItemLabel()(
								TextNode(Size2)(
									domi.Textf("%d Series", len(col.Series())),
								),
							),
							Button(onClick(&msg.CollectionSeriesAddOpen{CollectionID: col.ID()}), ButtonGhost)(
								Text("Add Series"),
							),
						),
						collectionSeriesItems(col),
					),

					SettingsGroup()(
						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Delete Collection"),
								SettingsItemLabelDescription("Deleted items remain in Trash for 30 days"),
							),
							trashForm(kind.Collection{}, col.ID()),
						),
					),
				),
			),
		),
	)
}

// AppCollectionMovieAddDialog renders the add-movie-to-collection
// picker: a library search box and its results.
func AppCollectionMovieAddDialog(colID, query string, results []model.CollectionMovieSearchResult) domi.Node {
	return dialog(&msg.DialogClose{})(
		FlexCol(Gap2, Class("v-media-dialog"))(
			html.Div(
				Class("v-media-dialog-fixed"),
			)(
				domi.Text("Add Movie to Collection"),
			),
			html.Form(
				onSubmit("q", func(v string) msg.Msg {
					return &msg.CollectionPickerSearch{Query: v}
				}),
			)(
				InputText(
					Attr("autofocus")(""),
					Class("v-media-dialog-fixed"),
					attr.Name("q"),
					attr.Value(query),
				),
			),
			html.Div(
				Class("v-media-dialog-results"),
			)(
				AppCollectionMovieSearchResults(colID, results),
			),
		),
	)
}

func AppCollectionMovieSearchResults(colID string, results []model.CollectionMovieSearchResult) domi.Node {
	return FlexCol(Gap4, Class("v-media-detail-body"))(
		rangeNodes(results, func(r model.CollectionMovieSearchResult) domi.Node {
			mw := r.Movie
			return html.Button(
				onClick(&msg.CollectionMovieAdd{
					CollectionID: colID,
					MovieID:      mw.MovieHead.ID(),
				}),
				Inert(r.InCollection),
				Style("all: unset; cursor: pointer; width: 100%"),
			)(
				Card(CardSurface, CardSize3, Class("v-media-search-card"))(
					FlexRow(Gap4, Style("height: 100%"))(
						Inset(InsetSideLeft, Class("v-media-search-poster"))(
							PosterImg(AspectPoster, Style("height: 100%"), imgAttrs(mw.Poster())),
						),
						FlexCol(Gap2)(
							iff(r.InCollection, func() domi.Node {
								return Label("solid/check-circle", "Already in Collection")
							}),

							TextNode(TextBold)(domi.Text(mw.Title())),
							TextNode()(domi.Text(mw.MovieEditionHead.Year())),
							TextNode()(domi.Text(fmt.Sprintf("%d min", mw.MovieEditionHead.Runtime()))),
						),
					),
				),
			)
		}),
	)
}

func collectionMovieItemID(colID, movieID string) string {
	return "col-" + colID + "-mo-" + movieID
}

func collectionMovieItems(col *model.Collection) domi.Node {
	return rangeNodes(col.Movies(), func(mo *model.MovieWork) domi.Node {
		return SettingsItem(attr.ID(collectionMovieItemID(col.ID(), mo.MovieHead.ID())))(
			FlexRow(Style("align-items:center"), Gap4)(
				SettingsItemLabelIcon()(Icon("line/film-01")),
				SettingsItemLabelTitle(mo.Title()+" ("+mo.Year()+")"),
			),
			Button(onClick(&msg.CollectionMovieRemove{
				CollectionID: col.ID(),
				MovieID:      mo.MovieHead.ID(),
			}), SettingsHover, ButtonGhost)(Text("Remove")),
		)
	})
}

// AppCollectionSeriesAddDialog renders the add-series-to-collection
// picker: a library search box and its results.
func AppCollectionSeriesAddDialog(colID, query string, results []model.CollectionSeriesSearchResult) domi.Node {
	return dialog(&msg.DialogClose{})(
		FlexCol(Gap2, Class("v-media-dialog"))(
			html.Div(
				Class("v-media-dialog-fixed"),
			)(
				domi.Text("Add Series to Collection"),
			),
			html.Form(
				onSubmit("q", func(v string) msg.Msg {
					return &msg.CollectionPickerSearch{Query: v}
				}),
			)(
				InputText(
					Attr("autofocus")(""),
					Class("v-media-dialog-fixed"),
					attr.Name("q"),
					attr.Value(query),
				),
			),
			html.Div(
				Class("v-media-dialog-results"),
			)(
				AppCollectionSeriesSearchResults(colID, results),
			),
		),
	)
}

func AppCollectionSeriesSearchResults(colID string, results []model.CollectionSeriesSearchResult) domi.Node {
	return FlexCol(Gap4, Class("v-media-detail-body"))(
		rangeNodes(results, func(r model.CollectionSeriesSearchResult) domi.Node {
			sw := r.Series
			return html.Button(
				onClick(&msg.CollectionSeriesAdd{
					CollectionID: colID,
					SeriesID:     sw.SeriesHead.ID(),
				}),
				Inert(r.InCollection),
				Style("all: unset; cursor: pointer; width: 100%"),
			)(
				Card(CardSurface, CardSize3, Class("v-media-search-card"))(
					FlexRow(Gap4, Style("height: 100%"))(
						Inset(InsetSideLeft, Class("v-media-search-poster"))(
							PosterImg(AspectPoster, Style("height: 100%"), imgAttrs(sw.Poster())),
						),
						FlexCol(Gap2)(
							iff(r.InCollection, func() domi.Node {
								return Label("solid/check-circle", "Already in Collection")
							}),

							TextNode(TextBold)(domi.Text(sw.Title())),
						),
					),
				),
			)
		}),
	)
}

func collectionSeriesItemID(colID, seriesID string) string {
	return "col-" + colID + "-sr-" + seriesID
}

func collectionSeriesItems(col *model.Collection) domi.Node {
	return rangeNodes(col.Series(), func(sw *model.SeriesWork) domi.Node {
		return SettingsItem(attr.ID(collectionSeriesItemID(col.ID(), sw.SeriesHead.ID())))(
			FlexRow(Style("align-items:center"), Gap4)(
				SettingsItemLabelIcon()(Icon("line/tv-03")),
				SettingsItemLabelTitle(sw.Title()),
			),
			Button(onClick(&msg.CollectionSeriesRemove{
				CollectionID: col.ID(),
				SeriesID:     sw.SeriesHead.ID(),
			}), SettingsHover, ButtonGhost)(Text("Remove")),
		)
	})

}
