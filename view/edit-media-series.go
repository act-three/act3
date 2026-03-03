package view

import (
	"fmt"
	"time"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/ui/turbo"
)

const EditMediaSeriesListItems = "series-list-items"

func EditMediaSeries(
	title string,
	s []*model.SeriesHead,
	detail ...html.Node,
) html.Node {
	return app(title, FlexCol(attr.Class("place-self-stretch"))(
		ToolbarPrimary()(
			DialogButton("/dialog/series-add", ButtonRadiusMedium, ButtonSurface)(
				Icon("plus"),
				html.Text("Add Series"),
			),
			html.Div(attr.Class("relative w-md"))(
				editMediaSeriesSearchbar(),
			),
			html.Div(),
		),
		Split()(
			List("/edit/series/", "detail")(
				turbo.Sink(EditMediaSeriesListItems)(
					ListItems(s, EditMediaSeriesListItem),
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

func EditMediaSeriesListItem(ss *model.SeriesHead, attrs ...attr.Node) html.Node {
	return Card(CardGhost,
		attr.Group(attrs...),
		ListID(ss.ID()),
		ListURL(ss.EditURL()),
	)(
		CardMedia()(html.Img(attr.Src(ss.TVmazeImageURL()))),
		CardContent()(
			CardTitle()(html.Text(ss.Title())),
			CardDescription(attr.Class("line-clamp-2"))(
				html.If(ss.PremieredOn() != nil,
					func() html.Node { return html.Text(*ss.PremieredOn()) },
				),
				html.Text(ss.Status()),
			),
		),
	)
}

func EditMediaSeriesDetail(
	sr *model.Series,
	sed *model.SeriesEdition,
	dls []*model.DownloadHead,
) html.Node {
	return html.Div(attr.Class("place-self-stretch h-full w-full flex flex-col"))(
		ScrollY(
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
						return editMediaSeriesDetailEdition(sed, dls)
					},
				),
			),
		),
	)
}

func editMediaSeriesSearchbar() html.Node {
	return html.Text("editMediaSeriesSearchbar")
}

func editMediaSeriesDetailEdition(
	sed *model.SeriesEdition,
	dls []*model.DownloadHead,
) html.Node {
	return html.Div()(
		editMediaSeriesAddTorrentButton(sed.ID()),
		html.Div(
			attr.Class("border"),
		)(
			turbo.Sink("edition-torrents-"+sed.ID())(
				editMediaSeriesListDownloadDetail(dls),
			),
		),
		editMediaSeriesDetailEpisodeList(sed),
	)
}

func editMediaSeriesAddTorrentButton(sedID string) html.Node {
	return html.Form(
		attr.Class("flex flex-row gap-1 group"),
		attr.Method("POST"),
		attr.Enctype("multipart/form-data"),
		attr.Action("/do/add-torrent"),
		stimulus.Controller("add-torrent"),
		stimulus.Action("turbo:submit-end->add-torrent#reset"),
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
			stimulus.Target("add-torrent", "picker"),
			stimulus.Action("change->add-torrent#upload"),
		),
		Button(
			stimulus.Target("add-torrent", "button"),
			stimulus.Action("click->add-torrent#open:prevent"),
		)(
			html.Text("Add Torrent…"),
		),
	)
}

func editMediaSeriesListDownloadDetail(dls []*model.DownloadHead) html.Node {
	return html.Range(dls, editMediaSeriesListDownloadDetailItem)
}

func editMediaSeriesListDownloadDetailItem(dl *model.DownloadHead) html.Node {
	return html.Div(
		attr.Class("p-1"),
	)(
		html.A(
			attr.Href(dl.URL()),
			turbo.DataFrame("main"),
		)(
			html.Text(dl.Title()),
		),
	)
}

func editMediaSeriesDetailEpisodeList(sed *model.SeriesEdition) html.Node {
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
							html.RangeSeq(sn.Episodes(model.Significant), editMediaSeriesDetailEpisodeListItem),
						),
					)
				})
			},
		),
	)
}

func editMediaSeriesDetailEpisodeListItem(ep *model.Episode) html.Node {
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
				DialogButton(ep.EditDialogURL(), ButtonGhost)(Icon("info")),
			),
		),
		ProgressContainer(ep.ID(), ep.Progress()),
	)
}

func truncate(s string, max int) string {
	if len(s) < max {
		return s
	}
	return s[:max] + "…"
}

// formatDuration formats d as a short human-readable string
// for UI display (e.g. "3m 24s", "45s", "1h 12m").
func formatDuration(d time.Duration) string {
	d = d.Truncate(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", h, m)
}
