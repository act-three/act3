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

const EditMediaMoviesListItems = "movie-list-items"

func EditMediaMovies(
	title string,
	s []*model.MovieHead,
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
				turbo.Sink(EditMediaMoviesListItems)(
					ListItems(s, EditMediaMoviesListItem),
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

func EditMediaMoviesListItem(
	mo *model.MovieHead, attrs ...attr.Node,
) html.Node {
	return Card(CardGhost,
		attr.Group(attrs...),
		ListID(mo.ID()),
		ListURL(mo.EditURL()),
	)(
		CardMedia()(html.Img(attr.Src(mo.ImageURL()))),
		CardContent()(
			CardTitle()(html.Text(mo.Title())),
			CardDescription(LineClamp2)(
				html.Text(mo.YearDisplay()),
			),
		),
	)
}

func EditMediaMoviesDetail(
	mo *model.Movie,
	med *model.MovieEdition,
	dls []*model.DownloadHead,
) html.Node {
	return FlexCol(Class("v-media-detail"))(
		ScrollY(
			Class("v-media-detail-body"),
		)(
			FlexCol(Gap4)(
				FlexRow(Gap2)(
					expr.IfElse(mo.ImageURL() != "",
						func() html.Node {
							return ImageFrame()(
								PosterImg(PosterFill, attr.Src(mo.ImageURL())),
							)
						},
						func() html.Node {
							return html.Group()
						},
					),
					FlexCol(Gap4, Class("v-media-detail-body"))(
						html.H1()(html.Text(mo.Title())),
						html.If(mo.YearDisplay() != "", func() html.Node {
							return html.P()(html.Text(mo.YearDisplay()))
						}),
						html.P()(html.Safe(mo.Summary())),
					),
				),
				editMediaMoviesDetailEdition(mo, med, dls),
			),
		),
	)
}

func EditMovieAddDialog(frameID string) html.Node {
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
				turbo.Frame("results"),
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

// EditMovieSearchResults renders the search results for
// adding a movie.
func EditMovieSearchResults(results []MovieSearchResult) html.Node {
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
		return MovieResultLink(t.Local)
	}
	return turbo.Frame(frameID)(
		html.Form(
			attr.Method("post"),
			attr.Action("/-/do/add-movie-tmdb"),
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

func MovieResultLink(mo *model.MovieHead) html.Node {
	return FlexRow(Gap2)(
		Label("line/check-circle", "In Library"),
		Button(
			Href(mo.EditURL()),
			Attr("data-turbo-frame")("detail"),
			Attr("data-action")("click->dialog#close"),
		)(
			Text("Edit"),
		),
	)
}

func editMediaMoviesDetailEdition(
	mo *model.Movie,
	med *model.MovieEdition,
	dls []*model.DownloadHead,
) html.Node {
	if med == nil {
		return html.Div()(html.Text("Unknown Edition"))
	}
	return html.Div()(
		editMediaMoviesEditionSelector(mo),
		AddTorrentButton("med-id", med.ID()),
		html.Div(
			attr.Class("v-media-download-list"),
		)(
			turbo.Sink("edition-torrents-"+med.ID())(
				html.Range(dls, DownloadListItem),
			),
		),
		editMediaMoviesDetailVideos(med),
	)
}

func editMediaMoviesEditionSelector(mo *model.Movie) html.Node {
	return html.Div(
		attr.Name("edition"),
	)(
		html.Div(
			attr.Class("v-media-selector-label"),
		)(
			html.Text("edition"),
		),
		html.Div()(
			html.RangeSeq(mo.MovieEditionSeq(), func(med *model.MovieEdition) html.Node {
				return html.Div(
					attr.Value(med.Title()),
				)(
					html.Label()(html.Text(med.Title())),
				)
			}),
		),
	)
}

func editMediaMoviesDetailVideos(med *model.MovieEdition) html.Node {
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
						attr.Action("/-/do/reimport-video/"+v.ID()),
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
									"/-/do/reencode-video/"+v.ID(),
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
