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
)

const AppMoviesListItems = "movie-list-items"

func AppMovies(
	title string,
	s []*model.MovieWork,
	detail ...html.Node,
) html.Node {
	return app(title, FlexCol(attr.Class("v-media-page"))(
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
	))
}

func AppMoviesListItem(
	mo *model.MovieWork, attrs ...attr.Node,
) html.Node {
	return Card(CardGhost,
		attr.Group(attrs...),
		ListID(mo.MovieHead.ID()),
		ListURL(mo.EditorPath()),
	)(
		CardMedia()(html.Img(attr.Src(mo.ImageURL()))),
		CardContent()(
			CardTitle()(movieEditionTitle(mo.MovieEditionHead.ID(), mo.Title())),
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
	_ = mo
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
						TextNode(Size6)(movieEditionTitle(med.ID(), med.Title())),
						Box()(
							Link(
								med.TheaterPath(),
								turbo.DataFrame("_top"),
								Class(movieEditionTheaterLinkClass(med.ID())),
							)(Text("View in Theater", Size3,
								// TODO(april): maybe make this the default for Text
								attr.Style("display: inline-block"),
							)),
						),
					),

					SettingsGroup()(
						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Title"),
							),
							SettingsTextField("/-/do/movie-edition-set-title", "title", med.Title(), movieEditionTitleAttrClass(med.ID()))(
								Hidden("id", med.ID()),
							),
						),

						html.If(len(editions) > 1, func() html.Node {
							return SettingsItem()(
								SettingsItemLabel()(
									SettingsItemLabelTitle("Edition"),
								),

								SettingsTextField("/-/do/movie-edition-set-label", "label", med.Label(), movieEditionLabelAttrClass(med.ID()))(
									Hidden("id", med.ID()),
								),
							)
						}),

						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Year Released"),
							),

							SettingsTextField("/-/do/movie-edition-set-year", "year", med.Year(), "")(
								Hidden("id", med.ID()),
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

							SettingsTextField("/-/do/movie-edition-set-runtime", "runtime", med.RuntimeString(), "", SettingsTextFieldSuffix(" min"))(
								Hidden("id", med.ID()),
							),
						),
					),

					FlexCol(Gap2)(
						SettingsContent()(Text("Summary", Size2)),
						SettingsTextArea("/-/do/movie-edition-set-summary", "summary", med.Summary())(
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
					turbo.StreamTarget("edition-torrents-"+med.ID(), SettingsGroupItems)(
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

				FlexCol(Gap2)(
					html.If(med.Slug() == "" && len(editions) > 1,
						func() html.Node {
							return SettingsContent()(
								Label(
									"line/x-circle",
									"The default edition can't be deleted. To delete this edition, first choose another default.",
									Size2,
								),
							)
						},
					),
					SettingsGroup(Disabled(true /* TODO(april): med.Slug() == "" && len(editions) > 1 */))(
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

							html.Form(
								attr.Method("POST"),
								attr.Action("/-/do/movie-edition-delete"),
							)(
								Hidden("edition-id", med.ID()),
								Button(Destructive, ButtonGhost, ButtonSize2)(Text("Delete")),
							),
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
			href := attr.Group()
			if ed.MovieEditionHead.ID() == current.ID() {
				selected = CardSelected
			} else {
				href = attr.Href(ed.EditorPath())
			}
			return Card(
				CardSurface,
				CardSize1,
				Class(movieEditionEditorLinkClass(ed.MovieEditionHead.ID())),
				href,
				selected,
			)(
				FlexRow()(
					CardContent()(
						CardTitle()(
							movieEditionLabel(ed.MovieEditionHead.ID(), ed.MovieEditionHead.Label()),
						),
						CardDescription()(
							movieTheaterPathText(ed.MovieHead.ID(), ed.MovieHead.Slug(), ed.MovieEditionHead.ID(), ed.MovieEditionHead.Slug()),
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
			)
		}),
	)
}

func movieEditionTitleTargetClass(id string) string {
	return "movie-edition-" + id + "-title"
}

func movieEditionTitleAttrClass(id string) string {
	return "movie-edition-" + id + "-title-attr"
}

func MovieEditionSetTitle(id, title string) html.Node {
	return html.Group(
		turbo.ReplaceTargets("."+movieEditionTitleTargetClass(id), turbo.Morph)(
			movieEditionTitle(id, title),
		),
		SettingsTextFieldSetValue("."+movieEditionTitleAttrClass(id), title),
	)
}

func movieEditionTitle(id, title string) html.Node {
	return html.Span(Class(movieEditionTitleTargetClass(id)))(html.Text(title))
}

func movieEditionLabelTargetClass(id string) string {
	return "movie-edition-" + id + "-label"
}

func movieEditionLabelAttrClass(id string) string {
	return "movie-edition-" + id + "-label-attr"
}

func MovieEditionSetLabel(id, label string) html.Node {
	return html.Group(
		turbo.ReplaceTargets("."+movieEditionLabelTargetClass(id), turbo.Morph)(
			movieEditionLabel(id, label),
		),
		SettingsTextFieldSetValue("."+movieEditionLabelAttrClass(id), label),
	)
}

func movieEditionLabel(id, label string) html.Node {
	return html.Span(Class(movieEditionLabelTargetClass(id)))(html.Text(label))
}

func movieSlugTargetClass(id string) string {
	return "movie-" + id + "-slug"
}

func movieSlug(id, slug string) html.Node {
	return html.Span(Class(movieSlugTargetClass(id)))(html.Text(slug))
}

func movieEditionSlugTargetClass(id string) string {
	return "movie-edition-" + id + "-slug"
}

func movieEditionSlug(id, slug string) html.Node {
	return html.Span(Class(movieEditionSlugTargetClass(id)))(html.Text(slug))
}

// movieTheaterPathText renders "/slug" or "/slug/edition-slug"
// with each slug segment in a targetable span.
func movieTheaterPathText(movieID, movieSlugVal, editionID, editionSlugVal string) html.Node {
	if editionSlugVal == "" {
		return Group(html.Text("/"), movieSlug(movieID, movieSlugVal))
	}
	return Group(
		html.Text("/"), movieSlug(movieID, movieSlugVal),
		html.Text("/"), movieEditionSlug(editionID, editionSlugVal),
	)
}

func movieEditionTheaterLinkClass(id string) string {
	return "movie-edition-" + id + "-theater-link"
}

func movieEditionEditorLinkClass(id string) string {
	return "movie-edition-" + id + "-editor-link"
}

func MovieSetSlug(id, oldSlug, newSlug string, editions []*model.MovieWork) html.Node {
	nodes := []html.Node{
		turbo.ReplaceTargets("."+movieSlugTargetClass(id), turbo.Morph)(
			movieSlug(id, newSlug),
		),
		turbo.SetTargets("[data-list-id-param=\""+id+"\"]",
			html.Div(ListURL("/app/movies/"+newSlug))(),
		),
	}
	for _, ed := range editions {
		edSlug := ed.MovieEditionHead.Slug()
		oldEditorPath := path.Join("/app/movies/"+oldSlug, edSlug)
		nodes = append(nodes,
			turbo.SetTargets("."+movieEditionTheaterLinkClass(ed.MovieEditionHead.ID()),
				html.Div(attr.Href(ed.TheaterPath()))(),
			),
			turbo.SetTargets("."+movieEditionEditorLinkClass(ed.MovieEditionHead.ID()),
				html.Div(attr.Href(ed.EditorPath()))(),
			),
			turbo.URLReplace(oldEditorPath, ed.EditorPath()),
		)
	}
	return Group(nodes...)
}

func MovieEditionSetSlug(ed *model.MovieWork, oldSlug string) html.Node {
	id := ed.MovieEditionHead.ID()
	oldEditorPath := path.Join(ed.MovieHead.EditorPath(), oldSlug)
	return Group(
		turbo.ReplaceTargets("."+movieEditionSlugTargetClass(id), turbo.Morph)(
			movieEditionSlug(id, ed.MovieEditionHead.Slug()),
		),
		turbo.SetTargets("."+movieEditionTheaterLinkClass(id),
			html.Div(attr.Href(ed.TheaterPath()))(),
		),
		turbo.SetTargets("."+movieEditionEditorLinkClass(id),
			html.Div(attr.Href(ed.EditorPath()))(),
		),
		turbo.URLReplace(oldEditorPath, ed.EditorPath()),
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
						Button(Destructive)(
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
								Button(Destructive)(
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
