package view

import (
	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
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
