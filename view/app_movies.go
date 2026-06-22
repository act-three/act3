package view

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/expr"
	"ily.dev/act3/model"
	"ily.dev/act3/msg"
	"ily.dev/act3/service/tmdb"
	. "ily.dev/act3/ui"
	"ily.dev/act3/web/static"
)

func AppMovies(
	s []*model.MovieWork,
	selected *model.MovieEdition,
	editions []*model.MovieWork,
	dls []*model.DownloadHead,
	uploads []model.Upload,
	notFound bool,
) (title string, n domi.Node) {
	title = "All Movies"
	if selected != nil {
		title = selected.Title()
	}
	return title, FlexCol(Class("v-media-page"))(
		ToolbarPrimary()(
			FlexRow(Gap4)(
				Button(onClick(&msg.MovieAddOpen{}), ButtonSurface)(
					Text("Add Movie"),
				),
				Button(onClick(&msg.MovieCreate{}), ButtonSurface)(
					Text("New Movie"),
				),
			),
		),
		Split()(
			List()(
				ListItems(s, func(mo *model.MovieWork) bool {
					return selected != nil && mo.MovieHead.ID() == selected.MovieHead().ID()
				}, AppMoviesListItem),
			),
			appMovieSelection(selected, editions, dls, uploads, notFound),
		),
	)
}

func appMovieSelection(
	selected *model.MovieEdition,
	editions []*model.MovieWork,
	dls []*model.DownloadHead,
	uploads []model.Upload,
	notFound bool,
) domi.Node {
	switch {
	case selected != nil:
		return appMoviesDetail(selected, editions, dls, uploads)
	case notFound:
		return Center(Class("v-media-muted"))(
			domi.Text("Not Found"),
		)
	}
	return Center(Class("v-media-muted"))(
		domi.Text("No Movie Selected"),
	)
}

func AppMoviesListItem(
	mo *model.MovieWork, attrs ...domi.Attr,
) domi.Node {
	return CardLink(mo.EditorPath(), CardGhost,
		group(attrs...),
	)(
		CardMedia()(html.Img(imgAttrs(mo.Poster()))),
		CardContent()(
			CardTitle()(domi.Text(mo.Title())),
			CardDescription(LineClamp2)(
				domi.Text(mo.MovieEditionHead.ReleaseDate()),
			),
		),
	)
}

func appMoviesDetail(
	med *model.MovieEdition,
	editions []*model.MovieWork,
	dls []*model.DownloadHead,
	uploads []model.Upload,
) domi.Node {
	mo := med.MovieHead()
	return FlexCol(Class("v-media-detail"))(
		ScrollY(
			Class("v-media-detail-body"),
		)(
			SettingsPage()(
				iff(len(editions) > 1,
					func() domi.Node {
						return appMoviesEditionList(editions, med)
					}),

				FlexCol(Gap6)(
					SettingsContent()(
						TextNode(Size6)(domi.Text(med.Title())),
						Box()(
							Link(
								med.TheaterPath(),
							)(Text("View in Theater", Size3,
								// TODO(april): maybe make this the default for Text
								Style("display: inline-block"),
							)),
						),
					),

					SettingsGroup()(
						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Title"),
							),
							SettingsTextField(med.Title(), func(v string) msg.Msg {
								return &msg.MovieEditionSetTitle{ID: med.ID(), Title: v}
							}),
						),

						iff(len(editions) > 1, func() domi.Node {
							return SettingsItem()(
								SettingsItemLabel()(
									SettingsItemLabelTitle("Edition"),
								),

								SettingsTextField(med.Label(), func(v string) msg.Msg {
									return &msg.MovieEditionSetLabel{ID: med.ID(), Label: v}
								}),
							)
						}),

						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Release Date"),
							),

							SettingsTextField(med.ReleaseDate(), func(v string) msg.Msg {
								return &msg.MovieEditionSetReleaseDate{ID: med.ID(), ReleaseDate: v}
							}),
						),

						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Poster"),
							),

							buttonImageEdit(
								&msg.ImageDialogOpen{ID: med.ID()},
								med.Poster(),
								AspectPoster,
							),
						),

						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Runtime"),
							),

							SettingsTextField(med.RuntimeString(), func(v string) msg.Msg {
								return &msg.MovieEditionSetRuntime{ID: med.ID(), Runtime: v}
							}, SettingsTextFieldSuffix(" min")),
						),
					),

					FlexCol(Gap2)(
						SettingsContent()(Text("Summary", Size2)),
						SettingsTextArea(med.Summary(), func(v string) msg.Msg {
							return &msg.MovieEditionSetSummary{ID: med.ID(), Summary: v}
						}),
					),
				),

				SettingsGroup()(
					SettingsGroupHead()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Downloads"),
						),
						addTorrentButton("med-id", med.ID()),
					),
					rangeNodes(dls, downloadListItem),
				),

				SettingsGroup()(
					SettingsGroupHead()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Upload"),
						),
						uploadVideoControl("med-id", med.ID(), uploads),
					),
				),

				SettingsGroup()(
					appMoviesDetailVideos(med),
				),

				SettingsGroup()(
					SettingsItem()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Create Edition"),
							SettingsItemLabelDescription("Create a new edition by duplicating this one"),
						),

						Button(onClick(&msg.MovieEditionAdd{EditionID: med.ID()}),
							ButtonGhost, ButtonSize2,
						)(Text("Duplicate")),
					),
				),

				SettingsGroup()(
					SettingsItem()(
						SettingsItemLabel()(
							expr.IfElse(len(editions) > 1,
								func() domi.Node {
									return SettingsItemLabelTitle("Delete Edition")
								},
								func() domi.Node {
									return SettingsItemLabelTitle("Delete Movie")
								},
							),
							SettingsItemLabelDescription("Deleted items remain in Trash for 30 days"),
						),

						expr.IfElse(len(editions) > 1,
							func() domi.Node {
								return trashForm(med.ID())
							},
							func() domi.Node {
								return trashForm(mo.ID())
							},
						),
					),
				),
			),
		),
	)
}

// AppMovieAddDialog renders the add-movie dialog: a TMDB search box
// and its results. While a search is in flight, searching shows a
// spinner in place of the previous results.
func AppMovieAddDialog(query string, searching bool, results []model.MovieSearchResult) domi.Node {
	return dialog(&msg.DialogClose{})(
		FlexCol(Gap2, Class("v-media-dialog"))(
			html.Div(
				Class("v-media-dialog-fixed"),
			)(
				domi.Text("Add Movie"),
			),
			html.Form(
				onSubmit("q", func(v string) msg.Msg {
					return &msg.MovieSearch{Query: v}
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
				expr.IfElse(searching,
					func() domi.Node {
						return Spinner(Class("v-media-dialog-spinner"))
					},
					func() domi.Node {
						return AppMovieSearchResults(results)
					},
				),
			),
		),
	)
}

func AppMoviePosterDialog(med *model.MovieEdition) domi.Node {
	return imageDialog(&msg.DialogClose{}, AspectPoster)(
		buttonUpload()(
			Hidden("med-id", med.ID()),
			PosterImg(AspectPoster, PosterFill, imgAttrs(med.Poster())),
		),
	)
}

func posterURL(p *string) string {
	if p != nil {
		return tmdb.PosterURL(*p)
	}
	return static.Path("/static/poster-fallback.png")
}

// AppMovieSearchResults renders the search results for
// adding a movie.
func AppMovieSearchResults(results []model.MovieSearchResult) domi.Node {
	return FlexCol(Gap4, Class("v-media-detail-body"))(
		rangeNodes(results, func(t model.MovieSearchResult) domi.Node {
			return Card(CardSurface, CardSize3,
				Class("v-media-search-card"),
			)(
				FlexRow(Gap4, Style("height: 100%"))(
					Inset(InsetSideLeft, Class("v-media-search-poster"))(
						PosterImg(AspectPoster, Style("height: 100%"), attr.Src(posterURL(t.Movie.PosterPath))),
					),
					FlexCol(Gap2)(
						movieSearchTitle(t.Movie),
						movieSearchAction(t),
						TextNode(LineClamp3)(
							domi.Text(t.Movie.Overview),
						),
					),
				),
			)
		}),
	)
}

func movieSearchTitle(m tmdb.SearchResult) domi.Node {
	title := m.Title
	if len(m.ReleaseDate) >= 4 {
		title += " (" + m.ReleaseDate[:4] + ")"
	}
	return domi.Text(title)
}

func movieSearchAction(t model.MovieSearchResult) domi.Node {
	if t.Local != nil {
		return movieResultLink(t.Local.EditorPath())
	}
	return Button(
		onClick(&msg.MovieAdd{TMDBID: t.Movie.ID}),
		ButtonSurface,
	)(domi.Text("Add"))
}

func movieResultLink(editorURL string) domi.Node {
	return FlexRow(Gap2)(
		Label("line/check-circle", "In Library"),
		ButtonLink(editorURL)(
			Text("Edit"),
		),
	)
}

func appMoviesEditionList(editions []*model.MovieWork, current *model.MovieEdition) domi.Node {
	return FlexCol(Gap2)(
		rangeNodes(editions, func(ed *model.MovieWork) domi.Node {
			var card domi.Element
			if ed.MovieEditionHead.ID() == current.ID() {
				card = Card(CardSurface, CardSize1, CardSelected)
			} else {
				card = CardLink(ed.EditorPath(), CardSurface, CardSize1)
			}
			return card(
				FlexRow()(
					CardContent()(
						CardTitle()(
							domi.Text(ed.MovieEditionHead.Label()),
						),
						CardDescription()(
							movieTheaterPathText(&ed.MovieHead, &ed.MovieEditionHead),
						),
					),

					iff(ed.MovieEditionHead.ID() == current.ID() && ed.MovieEditionHead.Slug() != "", func() domi.Node {
						return Button(onClick(&msg.MovieEditionSetDefault{ID: ed.MovieEditionHead.ID()}),
							ButtonGhost, ButtonSize2,
						)(Text("Make Default"))
					}),
				),
			)
		}),
	)
}

// movieTheaterPathText renders "/slug" or "/slug/edition-slug"
// with each slug segment in a targetable span.
func movieTheaterPathText(mo *model.MovieHead, med *model.MovieEditionHead) domi.Node {
	if med.Slug() == "" {
		return Group(domi.Text("/"), domi.Text(mo.Slug()))
	}
	return Group(
		domi.Text("/"), domi.Text(mo.Slug()),
		domi.Text("/"), domi.Text(med.Slug()),
	)
}

func appMoviesDetailVideos(med *model.MovieEdition) domi.Node {
	vids := med.Videos()
	if len(vids) == 0 {
		return html.Div(
			Class("v-media-muted"),
		)(domi.Text("No videos"))
	}
	return FlexCol(Gap2)(
		TextNode(TextBold)(domi.Text("Videos")),
		rangeNodes(vids, func(v *model.Video) domi.Node {
			return html.Div(Class("v-media-indent"))(
				html.Div()(
					domi.Text("ID: "),
					domi.Text(v.ID()),
				),
				html.Div()(
					domi.Text("Name: "),
					domi.Text(v.Name()),
				),
				FlexRow(Gap2, Style("margin-top: 0.5rem"))(
					activeVideoControl(v, &msg.MovieVideoSetActive{MovieEditionID: med.ID(), VideoID: v.ID()}),
					Button(onClick(&msg.VideoReimport{ID: v.ID()}), Destructive)(
						domi.Text("Re-import"),
					),
					expr.IfElse(v.OriginalKey() != "",
						func() domi.Node {
							return Button(onClick(&msg.VideoReencode{ID: v.ID()}), Destructive)(
								domi.Text("Re-encode"),
							)
						},
						func() domi.Node { return Group() },
					),
					trashForm(v.ID()),
				),
			)
		}),
	)
}
