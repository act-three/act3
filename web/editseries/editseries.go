package editseries

import (
	"net/http"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/web/app"
	"ily.dev/act3/web/item"
	"ily.dev/act3/web/list"
	"ily.dev/act3/web/toolbar"
	"ily.dev/act3/web/turbo"
	"ily.dev/act3/web/view"
)

const ListItems = "series-list-items"

func Editor(
	title string,
	s []*model.SeriesHead,
	detail ...html.Node,
) http.Handler {
	return app.Page(title, FlexCol(attr.Class("place-self-stretch"))(
		toolbar.Primary()(
			html.Div()(app.DialogButton("/dialog/series-add")(
				Icon("plus"),
				html.Text("Add Series"),
			)).
				With(ButtonRoundedRect).
				With(ButtonBordered),
			html.Div(attr.Class("relative w-md"))(
				seriesSearchbar(),
			),
			html.Div(),
		),
		view.Split()(
			list.List("/edit/series/", "detail")(
				turbo.Sink(ListItems)(
					list.Items(s, ListItem),
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
					)(html.Text("No Series Selected"))
				},
			),
		),
	))
}

func ListItem(ss *model.SeriesHead, attrs ...attr.Node) html.Node {
	return item.Item(
		attr.Group(attrs...),
		list.ID(ss.ID()),
		list.URL(ss.EditURL()),
	)(
		item.Media()(html.Img(attr.Src(ss.TVmazeImageURL()))),
		item.Content()(
			item.Title()(html.Text(ss.Title())),
			item.Description(attr.Class("line-clamp-2"))(
				html.If(ss.PremieredOn() != nil,
					func() html.Node { return html.Text(*ss.PremieredOn()) },
				),
				html.Text(ss.Status()),
			),
		),
	)
}

func Detail(
	sr *model.Series,
	sed *model.SeriesEdition,
	dls []*model.DownloadHead,
) html.Node {
	return html.Div(attr.Class("place-self-stretch h-full w-full flex flex-col"))(
		ScrollArea(
			attr.Class("p-4"),
		)(
			html.Div(
				attr.Class("flex flex-col gap-4"),
			)(
				html.Div(
					attr.Class("flex gap-2"),
				)(
					html.Img(),
					html.Div(
						attr.Class("flex flex-col gap-4 p-4"),
					)(
						html.H1()(html.Text(sr.Title())),
						html.P()(html.Text(sr.Summary())),
					),
				),
				html.Div(
					attr.Name("order"),
				)(
					html.Div(
						attr.Class("w-[180px]"),
					)(
						html.Text("order by"),
					),
					html.Div()(
						html.RangeSeq(sr.SeriesEditionSeq(), func(sed *model.SeriesEdition) html.Node {
							return html.Div(
								attr.Value(sed.Title()),
							)(
								html.Label()(html.Text(sed.Title())),
							)
						}),
					),
				),
				expr.IfElse(sed == nil,
					func() html.Node {
						return html.Div()(html.Text("Unknown Edition"))
					},
					func() html.Node {
						return detailEdition(sed, dls)
					},
				),
			),
		),
	)
}

func detailEdition(
	sed *model.SeriesEdition,
	dls []*model.DownloadHead,
) html.Node {
	return html.Div()(
		addTorrentButton(sed.ID()),
		turbo.Sink("add-torrent-errors"),
		html.Div(
			attr.Class("border"),
		)(
			turbo.Sink("edition-torrents-"+sed.ID())(
				ListDownloadDetail(dls),
			),
		),
		detailEpisodeList(sed),
	)
}

func detailEpisodeList(sed *model.SeriesEdition) html.Node {
	return html.Div(
		attr.Class("flex flex-col gap-2"),
	)(
		expr.IfElse(sed == nil,
			func() html.Node {
				return html.Div()(html.Text("Unknown Order"))
			},
			func() html.Node {
				return html.RangeSeq(sed.Seasons(), func(sn *model.Season) html.Node {
					return html.Div(
						attr.Class("flex gap-2"),
					)(
						html.Div()(html.Text(sn.Name())),
						html.Div()(html.Textf("%d", sn.NumEpisodes(model.Significant))),
						html.Div()(
							html.RangeSeq(sn.Episodes(model.Significant), detailEpisode).
								With(ButtonBorderless),
						),
					)
				})
			},
		),
	)
}

func detailEpisode(ep *model.Episode) html.Node {
	return html.Div(
		attr.Class("flex flex-col gap-1"),
	)(
		html.Div(
			attr.Class("flex flex-row"),
		)(
			html.Div()(
				html.Text(ep.Label()),
			),
			html.Div()(
				app.DialogButton(ep.EditDialogURL())(Icon("info")),
			),
		),
		html.Range(ep.Progress(), func(pi model.ProgressItem) html.Node {
			return FlexCol(Class("text-gray-11/80 text-sm"))(
				Text(pi.Desc),
				Progress(pi.Progress(), attr.Class("max-w-xs")).
					With(ProgressSM),
			)
		}),
	)
}

func ListDownloadDetail(dls []*model.DownloadHead) html.Node {
	return html.Range(dls, downloadDetail)
}

func downloadDetail(dl *model.DownloadHead) html.Node {
	return html.Div(
		attr.Class("p-1"),
	)(
		html.A(
			attr.Href(dl.URL()),
			turbo.TurboFrame("main"),
		)(
			html.Text(dl.Title()),
		),
	)
}

func addTorrentButton(sedID string) html.Node {
	return html.Form(
		attr.Class("flex flex-row gap-1 group"),
		attr.Method("POST"),
		attr.Enctype("multipart/form-data"),
		attr.Action("/do/add-torrent"),
		turbo.Controller("add-torrent"),
		turbo.Action("turbo:submit-end->add-torrent#reset"),
	)(
		html.Input(
			attr.Type("hidden"),
			attr.Name("sed-id"),
			attr.Value(sedID),
		),
		html.Input(
			attr.Class("hidden"),
			attr.Type("file"),
			attr.Name("torrent"),
			turbo.Target("add-torrent", "picker"),
			turbo.Action("change->add-torrent#upload"),
		),
		Button(
			turbo.Target("add-torrent", "button"),
			turbo.Action("click->add-torrent#open:prevent"),
		)(
			html.Text("Add Torrent…"),
		),
	)
}

func seriesSearchbar() html.Node {
	return html.Text("seriesSearchbar")
}
