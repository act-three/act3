package view

import (
	"fmt"
	"strconv"
	"time"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	"ily.dev/act3/service/tvmaze"
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
			DialogButton("/-/dialog/series-add", ButtonRadiusMedium, ButtonSurface)(
				Icon("line/plus"),
				html.Text("Add Series"),
			),
			html.Div(attr.Class("relative w-md"))(
				editMediaSeriesSearchbar(),
			),
			html.Div(),
		),
		Split()(
			List("/app/series/", "detail")(
				turbo.Sink(EditMediaSeriesListItems)(
					ListItems(s, EditMediaSeriesListItem),
				),
			),
			expr.IfElse(detail != nil,
				func() html.Node {
					return Group(detail...)
				},
				func() html.Node {
					return Center(Class("text-gray-11/50"))(
						html.Text("No Series Selected"),
					)
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
	return FlexCol(Class("place-self-stretch h-full w-full"))(
		ScrollY(
			Class("p-4"),
		)(
			FlexCol(Gap4)(
				FlexRow(Gap2)(
					html.Img(),
					FlexCol(Gap4, Class("p-4"))(
						html.H1()(html.Text(sr.Title())),
						html.P()(html.Safe(sr.Summary())),
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
		attr.Action("/-/do/add-torrent"),
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
	return FlexCol(Gap2)(
		expr.IfElse(sed == nil,
			func() html.Node {
				return html.Div()(html.Text("Unknown Order"))
			},
			func() html.Node {
				return html.RangeSeq(sed.Seasons(), func(sn *model.Season) html.Node {
					return FlexRow(Gap2)(
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
	return FlexCol(Gap1)(
		FlexRow()(
			html.Div()(
				html.Text(ep.Label()),
			),
			html.Div()(
				DialogButton(ep.EditDialogURL(), ButtonGhost)(Icon("line/info-circle")),
			),
		),
		ProgressContainer(ep.ID(), ep.Progress()),
	)
}

func EditSeriesAddDialog() html.Node {
	return Dialog(
		FlexCol(
			attr.Attr("data-controller")("add-series"),
			Gap2,
			Class("w-2xl h-full"),
		)(
			html.Div(
				attr.Class("flex-none"),
			)(
				html.Text("Add Series"),
			),
			html.Form(
				attr.Action("/-/part/series-search"),
				attr.Attr("data-turbo-frame")("results"),
			)(
				InputText(
					attr.Attr("autofocus"),
					attr.Attr("data-action")("add-series#search"),
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

// EditEpisodeDialog renders the dialog for inspecting an
// episode's videos, renditions, and metadata.
func EditEpisodeDialog(
	ep *model.Episode,
	videos []schema.Video,
	renditions []schema.RenditionForStreaming,
) html.Node {
	return Dialog(
		ScrollY()(
			html.Div()(
				html.Text(ep.SeriesHead().Title()),
			),
			html.Div()(
				html.Text(ep.SeasonHead().Name()),
			),
			html.Div()(
				html.Text(ep.Label()),
			),

			TextNode(FontBold, Class("mt-4"))(html.Text("Videos")),
			expr.IfElse(len(videos) == 0,
				func() html.Node {
					return html.Div(
						attr.Class("text-gray-500"),
					)(html.Text("No videos found"))
				},
				func() html.Node { return html.Group() },
			),
			html.Range(videos, func(v schema.Video) html.Node {
				return editEpisodeDialogVideo(v)
			}),

			TextNode(FontBold, Class("mt-4"))(html.Text("Renditions for Streaming")),
			expr.IfElse(len(renditions) == 0,
				func() html.Node {
					return html.Div(
						attr.Class("text-gray-500"),
					)(html.Text("No renditions found"))
				},
				func() html.Node { return html.Group() },
			),
			html.Range(renditions, func(r schema.RenditionForStreaming) html.Node {
				return editEpisodeDialogRendition(r)
			}),

			TextNode(FontBold, Class("mt-4"))(html.Text("Metadata")),
			html.Div()(html.Text("Title")),
			html.Div()(html.Text("Sort Title")),
			html.Div()(html.Text("Season Number")),
			html.Div()(html.Text("Episode Number")),
			html.Div()(html.Text("Overview (plot summary)")),
			html.Div()(html.Text("Release Date")),
			html.Div()(html.Text("Special Episode Info")),
			html.Div()(html.Text("Path")),
			html.Div()(html.Text("Filesize")),
			html.Div()(html.Text("Video Details (codec, framerate, etc)")),
			html.Div()(html.Text("Audio Details (codec, etc)")),
			html.Div()(html.Text("Subtitle Details (format, etc)")),
		),
	)
}

func editEpisodeDialogVideo(v schema.Video) html.Node {
	return html.Div(
		attr.Class("ml-4 mt-2"),
	)(
		html.Div()(
			html.Text("ID: "),
			html.Text(v.ID),
		),
		html.Div()(
			html.Text("Release Path: "),
			html.Text(v.ReleasePath),
		),
		html.Div()(
			html.Text("Original Hash: "),
			html.Text(v.OriginalHash),
		),
		expr.IfElse(v.MVPlaylist != "",
			func() html.Node {
				return html.Div()(
					html.Text("Playlist: "),
					Code()(
						html.Text(v.MVPlaylist),
					),
				)
			},
			func() html.Node { return html.Group() },
		),
		FlexRow(Gap2, Class("mt-2"))(
			html.Form(
				attr.Action("/-/do/reimport-video/"+v.ID),
				attr.Method("POST"),
			)(
				Button(ButtonDestructive)(
					html.Text("Re-import"),
				),
			),
			expr.IfElse(v.OriginalHash != "",
				func() html.Node {
					return html.Form(
						attr.Action("/-/do/reencode-video/"+v.ID),
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
}

func editEpisodeDialogRendition(r schema.RenditionForStreaming) html.Node {
	return html.Div(
		attr.Class("ml-4 mt-2"),
	)(
		html.Div()(
			html.Text("ID: "),
			html.Text(r.ID),
		),
		html.Div()(
			html.Text("Video ID: "),
			html.Text(r.VideoID),
		),
		html.Div()(
			html.Text("Codec: "),
			html.Text(r.Codec),
		),
		html.Div()(
			html.Textf("Target Bitrate: %d kbit/s", r.TargetBitrate),
		),
		html.Div()(
			html.Textf("Remux: %v", r.Remux != 0),
		),
		html.Div()(
			html.Textf("Copy Audio: %v", r.CopyAudio != 0),
		),
		expr.IfElse(r.MaxHeight != 0,
			func() html.Node {
				return html.Div()(
					html.Textf("Max Height: %d", r.MaxHeight),
				)
			},
			func() html.Node { return html.Group() },
		),
		expr.IfElse(r.MaxFPS != 0,
			func() html.Node {
				return html.Div()(
					html.Textf("Max FPS: %d", r.MaxFPS),
				)
			},
			func() html.Node { return html.Group() },
		),
		expr.IfElse(r.Hash != "",
			func() html.Node {
				return html.Div()(
					html.Text("Hash: "),
					html.Text(r.Hash),
				)
			},
			func() html.Node { return html.Group() },
		),
	)
}

// SeriesSearchResult pairs a TVmaze show with an optional
// local series entry.
type SeriesSearchResult struct {
	TVmaze tvmaze.Show
	Local  *model.SeriesHead
}

// EditSeriesSearchResults renders the search results for
// adding a series.
func EditSeriesSearchResults(results []SeriesSearchResult) html.Node {
	return turbo.Frame("results")(
		FlexCol(Gap4, Class("p-4"))(
			html.Range(results, func(t SeriesSearchResult) html.Node {
				frameID := "tvmaze-" + strconv.Itoa(t.TVmaze.ID)
				return Card(CardSurface, CardSize3, Class("h-[200px]"))(
					FlexRow(Gap4, Class("h-full"))(
						Inset(InsetSideLeft, Class("flex-none"))(
							PosterImg(Class("h-full"), attr.Src(t.TVmaze.Image.Medium())),
						),
						FlexCol(Gap2)(
							html.Text(t.TVmaze.Name),
							expr.IfElse(t.Local == nil,
								func() html.Node {
									return turbo.Frame(frameID)(
										html.Form(
											attr.Method("post"),
											attr.Action("/-/do/add-series"),
											turbo.DataFrame(frameID),
										)(
											html.Input(
												attr.Type("hidden"),
												attr.Name("id"),
												attr.Value(strconv.Itoa(t.TVmaze.ID)),
											),
											Button(ButtonSurface)(html.Text("Add")),
										),
									)
								},
								func() html.Node {
									return SeriesResultLink(t.Local)
								},
							),
							TextNode(LineClamp3)(html.Safe(t.TVmaze.Summary)),
						),
					),
				)
			}),
		),
	)
}

func SeriesResultLink(ss *model.SeriesHead) html.Node {
	return FlexRow(Gap2)(
		Label("line/check-circle", "In Library"),
		Button(
			Href(ss.EditURL()),
			Attr("data-turbo-frame")("detail"),
			Attr("data-action")("click->dialog#dismiss"),
		)(
			Text("Edit"),
		),
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
