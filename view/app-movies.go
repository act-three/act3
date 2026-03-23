package view

import (
	"strconv"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	"ily.dev/act3/service/tmdb"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
)

const AppMoviesListItems = "movie-list-items"

func AppMovies(
	title string,
	s []*model.MovieWork,
	detail ...html.Node,
) html.Node {
	return app(title, FlexCol(attr.Class("v-media-page"))(
		ToolbarPrimary()(
			DialogButton("/-/dialog/movie-add")(
				Icon("line/plus"),
				html.Text("Add Movie"),
			),
			html.Div(),
		),
		Split()(
			List("/app/movies/", "detail")(
				turbo.Sink(AppMoviesListItems)(
					ListItems(s, AppMoviesListItem),
				),
			),
			expr.IfElse(detail != nil,
				func() html.Node {
					return Group(detail...)
				},
				func() html.Node {
					return Center(Class("v-media-muted"))(
						html.Text("No Movie Selected"),
					)
				},
			),
		),
	))
}

func AppMoviesListItem(
	mo *model.MovieWork, attrs ...attr.Node,
) html.Node {
	return Card(CardGhost,
		attr.Group(attrs...),
		ListID(mo.MovieHead.ID()),
		ListURL(mo.EditorURL()),
	)(
		CardMedia()(html.Img(attr.Src(mo.ImageURL()))),
		CardContent()(
			CardTitle()(html.Text(mo.Title())),
			CardDescription(LineClamp2)(
				html.Text(mo.Year()),
			),
		),
	)
}

func AppMoviesDetail(
	med *model.MovieEdition,
	editions []*model.MovieWork,
	dls []*model.DownloadHead,
) html.Node {
	mo := med.MovieHead()
	return FlexCol(Class("v-media-detail"))(
		ScrollY(
			Class("v-media-detail-body"),
		)(
			SettingsPage()(
				html.If(len(editions) > 1,
					func() html.Node {
						return appMoviesEditionList(editions, med)
					},
				),

				SettingsContent()(
					Text(med.Title(), TextSize6),
					// TODO(april): use med.TheaterURL to link to this edition
					Link(
						med.MovieHead().TheaterURL(),
						turbo.DataFrame("_top"),
					)(Text("View in Theater", TextSize3)),
				),

				SettingsGroup()(
					SettingsItem()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Title"),
						),
						SettingsTextField("/-/do/movie-edition-set-title", "title", med.Title())(
							html.Input(attr.Type("hidden"), attr.Name("id"), attr.Value(med.ID())),
						),
					),

					SettingsItem()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Year Released"),
						),

						SettingsTextField("/-/do/movie-edition-set-year", "year", med.Year())(
							html.Input(attr.Type("hidden"), attr.Name("id"), attr.Value(med.ID())),
						),
					),

					SettingsItem()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Poster"),
						),

						ImageFrame(attr.Style("width:30px"))(
							PosterImg(PosterFill, attr.Src(med.ImageURL())),
						),
					),

					SettingsItem()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Runtime"),
						),

						SettingsTextField("/-/do/movie-edition-set-runtime", "runtime", med.RuntimeDisplay(), SettingsTextFieldSuffix(" min"))(
							html.Input(attr.Type("hidden"), attr.Name("id"), attr.Value(med.ID())),
						),
					),
				),

				SettingsContent()(Text("Summary", TextSize2)),
				// TODO(april): create a text area component that looks good
				html.Textarea(
					attr.Style("border: 1px solid var(--border-strong)"),
					attr.Style("padding: 16px"),
					attr.Style("border-radius: 8px"),
					attr.Style("height: 10rem"),
					attr.Disabled,
				)(html.Text(med.Summary())),

				html.If(len(editions) > 1,
					func() html.Node {
						return SettingsGroup()(
							SettingsItem()(
								SettingsItemLabel()(
									SettingsItemLabelTitle("Edition"),
								),

								SettingsTextField("/-/do/movie-edition-set-label", "label", med.Label())(
									html.Input(attr.Type("hidden"), attr.Name("id"), attr.Value(med.ID())),
								),
							),

							html.If(med.Slug() != "",
								func() html.Node {
									return Group(
										SettingsItem()(
											SettingsItemLabel()(
												SettingsItemLabelTitle("URL"),
											),

											SettingsTextField("/-/do/movie-edition-set-slug", "slug", med.Slug(), SettingsTextFieldPrefix(mo.TheaterURL()+"/"))(
												html.Input(attr.Type("hidden"), attr.Name("id"), attr.Value(med.ID())),
											),
										),

										SettingsItem()(
											SettingsItemLabel()(
												SettingsItemLabelTitle("Default"),
												SettingsItemLabelDescription("Shown first when opening a movie"),
											),

											Button(ButtonGhost, ButtonSize2)(Text("Set Default")),
										),
									)
								},
							),
						)
					},
				),

				SettingsGroup()(
					SettingsGroupHead()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Downloads"),
						),
						addTorrentButton("med-id", med.ID()),
					),
					turbo.Sink("edition-torrents-"+med.ID(), SettingsGroupItems)(
						html.Range(dls, downloadListItem),
					),
				),

				SettingsGroup()(
					appMoviesDetailVideos(med),
				),

				SettingsGroup()(
					SettingsItem()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Edition"),
							SettingsItemLabelDescription("Create a new edition by duplicating this one"),
						),

						html.Form(
							attr.Method("POST"),
							attr.Action("/-/do/movie-edition-add"),
						)(
							html.Input(attr.Type("hidden"), attr.Name("edition-id"), attr.Value(med.ID())),
							Button(ButtonGhost, ButtonSize2)(Text("Duplicate")),
						),
					),
				),

				html.If(med.Slug() == "" && len(editions) > 1,
					func() html.Node {
						return SettingsContent()(
							Label(
								"line/x-circle",
								"The default edition can't be deleted. To delete this edition, first make another edition the default.",
							),
						)
					},
				),
				SettingsGroup()(
					SettingsItem()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Delete"),
							SettingsItemLabelDescription("Deleted items remain in Trash for 30 days"),
						),

						html.Form(
							attr.Method("POST"),
							attr.Action("/-/do/movie-edition-delete"),
						)(
							html.Input(attr.Type("hidden"), attr.Name("edition-id"), attr.Value(med.ID())),
							Button(ButtonDestructive, ButtonGhost, ButtonSize2)(Text("Delete")),
						),
					),
				),
			),
		),
	)
}

func AppMovieAddDialog(frameID string) html.Node {
	return Dialog(frameID,
		FlexCol(Gap2, Class("v-media-dialog"))(
			html.Div(
				attr.Class("v-media-dialog-fixed"),
			)(
				html.Text("Add Movie"),
			),
			html.Form(
				attr.Action("/-/part/movie-search"),
				attr.Attr("data-turbo-frame")("results"),
			)(
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

// MovieSearchResult pairs a TMDB search result with an
// optional local movie entry.
type MovieSearchResult struct {
	TMDB  tmdb.SearchResult
	Local *model.MovieHead
}

// AppMovieSearchResults renders the search results for
// adding a movie.
func AppMovieSearchResults(results []MovieSearchResult) html.Node {
	return turbo.Frame("results")(
		FlexCol(Gap4, Class("v-media-detail-body"))(
			html.Range(results, func(t MovieSearchResult) html.Node {
				frameID := "tmdb-" + strconv.Itoa(t.TMDB.ID)
				return Card(CardSurface, CardSize3,
					Class("v-media-search-card"),
				)(
					FlexRow(Gap4, attr.Style("height: 100%"))(
						Inset(InsetSideLeft, Class("v-media-search-poster"))(
							PosterImg(attr.Style("height: 100%"), attr.Src(tmdb.ImageURL(t.TMDB.PosterPath))),
						),
						FlexCol(Gap2)(
							movieSearchTitle(t.TMDB),
							movieSearchAction(frameID, t),
							TextNode(LineClamp3)(
								html.Text(t.TMDB.Overview),
							),
						),
					),
				)
			}),
		),
	)
}

func movieSearchTitle(m tmdb.SearchResult) html.Node {
	title := m.Title
	if len(m.ReleaseDate) >= 4 {
		title += " (" + m.ReleaseDate[:4] + ")"
	}
	return html.Text(title)
}

func movieSearchAction(frameID string, t MovieSearchResult) html.Node {
	if t.Local != nil {
		return MovieResultLink(t.Local.EditorURL())
	}
	return turbo.Frame(frameID)(
		html.Form(
			attr.Method("post"),
			attr.Action("/-/do/movie-add-by-tmdb"),
			turbo.DataFrame(frameID),
		)(
			html.Input(
				attr.Type("hidden"),
				attr.Name("id"),
				attr.Value(strconv.Itoa(t.TMDB.ID)),
			),
			Button(ButtonSurface)(html.Text("Add")),
		),
	)
}

func MovieResultLink(editorURL string) html.Node {
	return FlexRow(Gap2)(
		Label("line/check-circle", "In Library"),
		Button(
			Href(editorURL),
			Attr("data-turbo-frame")("detail"),
			Attr("data-action")("click->dialog#close"),
		)(
			Text("Edit"),
		),
	)
}

func appMoviesEditionList(editions []*model.MovieWork, current *model.MovieEdition) html.Node {
	return FlexCol(Gap2)(
		html.Range(editions, func(ed *model.MovieWork) html.Node {
			selected := attr.Group()
			if ed.MovieEditionHead.ID() == current.ID() {
				selected = CardSelected
			}
			return Card(
				CardSurface,
				CardSize1,
				attr.Href(ed.EditorURL()),
				selected,
			)(
				CardContent()(
					CardTitle()(
						Text(ed.MovieEditionHead.Label()),
					),
					CardDescription()(
						Text(ed.TheaterURL()),
					),
				),
			)
		}),
	)
}

func appMoviesDetailVideos(med *model.MovieEdition) html.Node {
	vids := med.Videos()
	if len(vids) == 0 {
		return html.Div(
			attr.Class("v-media-muted"),
		)(html.Text("No videos"))
	}
	return FlexCol(Gap2)(
		TextNode(FontBold)(html.Text("Videos")),
		html.Range(vids, func(v *model.Video) html.Node {
			return html.Div(Class("v-media-indent"))(
				html.Div()(
					html.Text("ID: "),
					html.Text(v.ID()),
				),
				html.Div()(
					html.Text("Path: "),
					html.Text(v.ReleasePath()),
				),
				FlexRow(Gap2, attr.Style("margin-top: 0.5rem"))(
					html.Form(
						attr.Action("/-/do/video-reimport/"+v.ID()),
						attr.Method("POST"),
					)(
						Button(ButtonDestructive)(
							html.Text("Re-import"),
						),
					),
					expr.IfElse(v.OriginalHash() != "",
						func() html.Node {
							return html.Form(
								attr.Action(
									"/-/do/video-reencode/"+v.ID(),
								),
								attr.Method("POST"),
							)(
								Button(ButtonDestructive)(
									html.Text("Re-encode"),
								),
							)
						},
						func() html.Node { return html.Group() },
					),
				),
			)
		}),
	)
}
