package view

import (
	"path"
	"strconv"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	"ily.dev/act3/service/tmdb"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/web/static"
)

const AppMoviesListItems = "movie-list-items"

func AppMovies(
	title string,
	s []*model.MovieWork,
	detail ...html.Node,
) (string, html.Node) {
	return title, FlexCol(Class("v-media-page"))(
		ToolbarPrimary()(
			DialogButton("/-/dialog/movie-add", ButtonSurface)(
				Text("Add Movie"),
			),
		),
		Split()(
			List("/app/movies/", "detail")(
				turbo.StreamTarget(AppMoviesListItems)(
					ListItems(s, AppMoviesListItem),
				),
			),
			turbo.Frame("detail", turbo.Advance())(
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
		),
	)
}

func AppMoviesListItem(
	mo *model.MovieWork, attrs ...attr.Node,
) html.Node {
	return Card(CardGhost,
		group(attrs...),
		ListID(mo.MovieHead.ID()),
		ListURL(mo.EditorPath()),
	)(
		CardMedia()(html.Img(imgAttrs(mo.PosterField()))),
		CardContent()(
			CardTitle()(LiveText(mo.TitleField())),
			CardDescription(LineClamp2)(
				LiveText(mo.MovieEditionHead.ReleaseDateField()),
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

				FlexCol(Gap6)(
					SettingsContent()(
						TextNode(Size6)(LiveText(med.TitleField())),
						Box()(
							Link(
								med.TheaterPath(),
								turbo.DataFrame("_top"),
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
							SettingsTextField("/-/do/movie-edition-set-title", "title", med.Title(), LiveAddr(med.TitleAddr()))(
								Hidden("id", med.ID()),
							),
						),

						html.If(len(editions) > 1, func() html.Node {
							return SettingsItem()(
								SettingsItemLabel()(
									SettingsItemLabelTitle("Edition"),
								),

								SettingsTextField("/-/do/movie-edition-set-label", "label", med.Label(), LiveAddr(med.LabelAddr()))(
									Hidden("id", med.ID()),
								),
							)
						}),

						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Release Date"),
							),

							SettingsTextField("/-/do/movie-edition-set-release-date", "release-date", med.ReleaseDate(), LiveAddr(med.ReleaseDateAddr()))(
								Hidden("id", med.ID()),
							),
						),

						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Poster"),
							),

							buttonPosterEdit(
								"/-/dialog/movie-poster/"+med.ID(),
								med.Poster(), med.PosterAddr(),
							),
						),

						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Runtime"),
							),

							SettingsTextField("/-/do/movie-edition-set-runtime", "runtime", med.RuntimeString(), LiveAddr(med.RuntimeAddr()), SettingsTextFieldSuffix(" min"))(
								Hidden("id", med.ID()),
							),
						),
					),

					FlexCol(Gap2)(
						SettingsContent()(Text("Summary", Size2)),
						SettingsTextArea("/-/do/movie-edition-set-summary", "summary", med.Summary(), LiveAddr(med.SummaryAddr()))(
							Hidden("id", med.ID()),
						),
					),
				),

				SettingsGroup()(
					SettingsGroupHead()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Downloads"),
						),
						addTorrentButton("med-id", med.ID()),
					),
					turbo.StreamTarget("edition-torrents-"+med.ID())(
						html.Range(dls, downloadListItem),
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

						html.Form(
							attr.Method("POST"),
							attr.Action("/-/do/movie-edition-add"),
						)(
							Hidden("edition-id", med.ID()),
							Button(ButtonGhost, ButtonSize2)(Text("Duplicate")),
						),
					),
				),

				SettingsGroup()(
					SettingsItem()(
						SettingsItemLabel()(
							expr.IfElse(len(editions) > 1,
								func() html.Node {
									return SettingsItemLabelTitle("Delete Edition")
								},
								func() html.Node {
									return SettingsItemLabelTitle("Delete Movie")
								},
							),
							SettingsItemLabelDescription("Deleted items remain in Trash for 30 days"),
						),

						expr.IfElse(len(editions) > 1,
							func() html.Node {
								return trashForm(med.ID())
							},
							func() html.Node {
								return trashForm(mo.ID())
							},
						),
					),
				),
			),
		),
	)
}

func AppMovieAddDialog() html.Node {
	return DialogStream(
		FlexCol(Gap2, Class("v-media-dialog"))(
			html.Div(
				Class("v-media-dialog-fixed"),
			)(
				html.Text("Add Movie"),
			),
			html.Form(
				attr.Action("/-/part/movie-search"),
				Attr("data-turbo-frame")("results"),
			)(
				InputText(
					Attr("autofocus"),
					Class("v-media-dialog-fixed"),
					attr.Name("q"),
				),
			),
			html.Div(
				Class("v-media-dialog-results"),
			)(
				turbo.Frame("results")(Spinner(Class("v-media-dialog-spinner"))),
			),
		),
	)
}

func AppMoviePosterDialog(med *model.MovieEdition) html.Node {
	return ImageDialogStream(2, 3)(
		buttonUpload()(
			Hidden("med-id", med.ID()),
			PosterImg(PosterFill, imgAttrs(med.PosterField())),
		),
	)
}

// MovieSearchResult pairs a TMDB search result with an
// optional local movie entry.
type MovieSearchResult struct {
	TMDB  tmdb.SearchResult
	Local *model.MovieHead
}

func posterURL(p *string) string {
	if p != nil {
		return tmdb.PosterURL(*p)
	}
	return static.Path("/static/poster-fallback.png")
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
					FlexRow(Gap4, Style("height: 100%"))(
						Inset(InsetSideLeft, Class("v-media-search-poster"))(
							PosterImg(Style("height: 100%"), attr.Src(posterURL(t.TMDB.PosterPath))),
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
		return MovieResultLink(t.Local.EditorPath())
	}
	return turbo.Frame(frameID)(
		html.Form(
			attr.Method("post"),
			attr.Action("/-/do/movie-add-by-tmdb"),
			turbo.DataFrame(frameID),
		)(
			Hidden("id", strconv.Itoa(t.TMDB.ID)),
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
		)(
			Text("Edit"),
		),
	)
}

func appMoviesEditionList(editions []*model.MovieWork, current *model.MovieEdition) html.Node {
	return FlexCol(Gap2)(
		html.Range(editions, func(ed *model.MovieWork) html.Node {
			selected := group()
			href := group()
			if ed.MovieEditionHead.ID() == current.ID() {
				selected = CardSelected
			} else {
				href = Href(ed.EditorPath())
			}
			return turbo.StreamTarget("edition-tab-" + ed.MovieEditionHead.ID())(
				Card(
					CardSurface,
					CardSize1,
					href,
					selected,
				)(
					FlexRow()(
						CardContent()(
							CardTitle()(
								LiveText(ed.MovieEditionHead.LabelField()),
							),
							CardDescription()(
								movieTheaterPathText(&ed.MovieHead, &ed.MovieEditionHead),
							),
						),

						html.If(ed.MovieEditionHead.ID() == current.ID() && ed.MovieEditionHead.Slug() != "", func() html.Node {
							return html.Form(
								attr.Method("POST"),
								attr.Action("/-/do/movie-edition-set-default"),
							)(
								Hidden("edition-id", ed.MovieEditionHead.ID()),
								Button(ButtonGhost, ButtonSize2)(Text("Make Default")),
							)
						}),
					),
				),
			)
		}),
	)
}

// movieTheaterPathText renders "/slug" or "/slug/edition-slug"
// with each slug segment in a targetable span.
func movieTheaterPathText(mo *model.MovieHead, med *model.MovieEditionHead) html.Node {
	if med.Slug() == "" {
		return Group(html.Text("/"), LiveText(mo.SlugField()))
	}
	return Group(
		html.Text("/"), LiveText(mo.SlugField()),
		html.Text("/"), LiveText(med.SlugField()),
	)
}

func MovieSetSlug(mo *model.MovieHead, oldSlug string, editions []*model.MovieWork) html.Node {
	nodes := []html.Node{
		LiveTextUpdate(mo.SlugField()),
		turbo.SetTargets("[data-list-id-param=\""+mo.ID()+"\"]",
			html.Div(ListURL(mo.EditorPath()))(),
		),
	}
	for _, ed := range editions {
		edSlug := ed.MovieEditionHead.Slug()
		oldTheaterPath := path.Join("/", oldSlug, edSlug)
		oldEditorPath := path.Join("/app/movies", oldSlug, edSlug)
		nodes = append(nodes,
			turbo.URLReplace(oldTheaterPath, ed.TheaterPath()),
			turbo.URLReplace(oldEditorPath, ed.EditorPath()),
		)
	}
	return Group(nodes...)
}

func MovieEditionSetSlug(ed *model.MovieWork, oldSlug string) html.Node {
	oldTheaterPath := path.Join(ed.MovieHead.TheaterPath(), oldSlug)
	oldEditorPath := path.Join(ed.MovieHead.EditorPath(), oldSlug)
	return Group(
		LiveTextUpdate(ed.MovieEditionHead.SlugField()),
		turbo.URLReplace(oldTheaterPath, ed.TheaterPath()),
		turbo.URLReplace(oldEditorPath, ed.EditorPath()),
	)
}

func MovieEditionChangePoster(med *model.MovieEditionHead) html.Node {
	return liveImgUpdate(med.PosterField())
}

func appMoviesDetailVideos(med *model.MovieEdition) html.Node {
	vids := med.Videos()
	if len(vids) == 0 {
		return html.Div(
			Class("v-media-muted"),
		)(html.Text("No videos"))
	}
	return FlexCol(Gap2)(
		TextNode(TextBold)(html.Text("Videos")),
		html.Range(vids, func(v *model.Video) html.Node {
			return html.Div(Class("v-media-indent"))(
				html.Div()(
					html.Text("ID: "),
					html.Text(v.ID()),
				),
				html.Div()(
					html.Text("Name: "),
					html.Text(v.Name()),
				),
				FlexRow(Gap2, Style("margin-top: 0.5rem"))(
					activeVideoControl(v, "/-/do/movie-video-set-active/"+med.ID()+"/"+v.ID()),
					ActionButton("/-/do/video-reimport/"+v.ID(), nil, Destructive)(
						html.Text("Re-import"),
					),
					expr.IfElse(v.OriginalKey() != "",
						func() html.Node {
							return ActionButton("/-/do/video-reencode/"+v.ID(), nil, Destructive)(
								html.Text("Re-encode"),
							)
						},
						func() html.Node { return Group() },
					),
					trashForm(v.ID()),
				),
			)
		}),
	)
}
