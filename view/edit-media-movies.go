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
	return app(title, FlexCol(attr.Class("place-self-stretch"))(
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
					return html.Div(
						attr.Class(`
							grid
							h-full
							w-full
							place-items-center
							text-gray-11/50
						`),
					)(html.Text("No Movie Selected"))
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
			CardDescription(attr.Class("line-clamp-2"))(
				html.Text(mo.YearDisplay()),
			),
		),
	)
}

func EditMediaMoviesDetail(mo *model.Movie) html.Node {
	return html.Div(
		attr.Class("place-self-stretch h-full w-full flex flex-col"),
	)(
		ScrollY(
			attr.Class("p-4"),
		)(
			html.Div(
				attr.Class("flex flex-col gap-4"),
			)(
				html.Div(
					attr.Class("flex gap-2"),
				)(
					expr.IfElse(mo.ImageURL() != "",
						func() html.Node {
							return html.Img(
								attr.Src(mo.ImageURL()),
								attr.Class(
									"w-[180px] aspect-2/3 object-cover rounded-sm",
								),
							)
						},
						func() html.Node {
							return html.Group()
						},
					),
					html.Div(
						attr.Class("flex flex-col gap-4 p-4"),
					)(
						html.H1()(html.Text(mo.Title())),
						html.If(mo.YearDisplay() != "", func() html.Node {
							return html.P()(html.Text(mo.YearDisplay()))
						}),
						html.P()(html.Safe(mo.Summary())),
					),
				),
				editMediaMoviesDetailVideos(mo),
			),
		),
	)
}

func EditMovieAddDialog() html.Node {
	return Dialog(
		html.Div(
			attr.Class(`
				w-2xl
				h-full
				flex
				flex-col
				gap-2
			`),
		)(
			html.Div(
				attr.Class("flex-none"),
			)(
				html.Text("Add Movie"),
			),
			html.Form(
				attr.Action("/-/part/movie-search"),
				attr.Attr("data-turbo-frame")("results"),
			)(
				InputText(
					attr.Attr("autofocus"),
					attr.Class("flex-none"),
					attr.Name("q"),
				),
			),
			html.Div(
				attr.Class(`
					flex-initial
					overflow-auto
					overscroll-contain
					h-dvh
					max-h-full
					border
					rounded-sm
				`),
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
		FlexCol(Gap4, Class("p-4"))(
			html.Range(results, func(t MovieSearchResult) html.Node {
				frameID := "tmdb-" + strconv.Itoa(t.TMDB.ID)
				return Card(CardSurface, CardSize3,
					Class("h-[200px]"),
				)(
					FlexRow(Gap4, Class("h-full"))(
						Inset(InsetSideLeft,
							Class("flex-none"),
						)(
							html.Img(
								Class("block h-full aspect-2/3 object-cover"),
								attr.Src(tmdb.ImageURL(t.TMDB.PosterPath)),
							),
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
			Attr("data-action")("click->dialog#dismiss"),
		)(
			Text("Edit"),
		),
	)
}

func editMediaMoviesDetailVideos(mo *model.Movie) html.Node {
	vids := mo.Videos()
	if len(vids) == 0 {
		return html.Div(
			attr.Class("text-gray-11/50"),
		)(html.Text("No videos"))
	}
	return html.Div(
		attr.Class("flex flex-col gap-2"),
	)(
		html.Div(attr.Class("font-bold"))(
			html.Text("Videos"),
		),
		html.Range(vids, func(v *model.Video) html.Node {
			return html.Div(attr.Class("ml-4 mt-2"))(
				html.Div()(
					html.Text("ID: "),
					html.Text(v.ID()),
				),
				html.Div()(
					html.Text("Path: "),
					html.Text(v.ReleasePath()),
				),
				html.Div(
					attr.Class("mt-2 flex gap-2"),
				)(
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
