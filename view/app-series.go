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
	"ily.dev/act3/ui/turbo"
)

const AppSeriesListItems = "series-list-items"

func AppSeries(
	title string,
	s []*model.SeriesWork,
	detail ...html.Node,
) html.Node {
	return app(title, FlexCol(attr.Class("v-media-page"))(
		ToolbarPrimary()(
			DialogButton("/-/dialog/series-add", ButtonRadiusMedium, ButtonSurface)(
				Icon("line/plus"),
				html.Text("Add Series"),
			),
			html.Div(attr.Class("v-media-searchbar"))(
				appSeriesSearchbar(),
			),
			html.Div(),
		),
		Split()(
			List("/app/series/", "detail")(
				turbo.Sink(AppSeriesListItems)(
					ListItems(s, AppSeriesListItem),
				),
			),
			expr.IfElse(detail != nil,
				func() html.Node {
					return Group(detail...)
				},
				func() html.Node {
					return Center(Class("v-media-muted"))(
						html.Text("No Series Selected"),
					)
				},
			),
		),
	))
}

func AppSeriesListItem(ss *model.SeriesWork, attrs ...attr.Node) html.Node {
	return Card(CardGhost,
		attr.Group(attrs...),
		ListID(ss.SeriesHead.ID()),
		ListURL(ss.EditorURL()),
	)(
		CardMedia()(html.Img(attr.Src(ss.TVmazeImageURL()))),
		CardContent()(
			CardTitle()(html.Text(ss.Title())),
			CardDescription(LineClamp2)(
				html.If(ss.PremieredOn() != nil,
					func() html.Node { return html.Text(*ss.PremieredOn()) },
				),
				html.Text(ss.Status()),
			),
		),
	)
}

func AppSeriesDetail(
	sed *model.SeriesEdition,
	editions []*model.SeriesWork,
	dls []*model.DownloadHead,
) html.Node {
	sr := sed.SeriesHead()
	return FlexCol(Class("v-media-detail"))(
		ScrollY(
			Class("v-media-detail-body"),
		)(
			FlexCol(Gap4)(
				appSeriesEditionList(editions, sed),
				FlexRow(Gap2)(
					html.Img(),
					FlexCol(Gap4, Class("v-media-detail-body"))(
						html.H1()(html.Text(sr.Title())),
						html.P()(html.Safe(sed.Summary())),
					),
				),
				appSeriesDetailEdition(sed, dls),
			),
		),
	)
}

func appSeriesSearchbar() html.Node {
	return html.Text("appSeriesSearchbar")
}

func appSeriesEditionList(editions []*model.SeriesWork, current *model.SeriesEdition) html.Node {
	return FlexCol(Gap4)(
		html.Range(editions, func(ed *model.SeriesWork) html.Node {
			selected := attr.Group()
			if ed.SeriesEditionHead.ID() == current.ID() {
				selected = CardSelected
			}
			return Card(
				CardSurface,
				CardSize3,
				attr.Href(ed.EditorURL()),
				selected,
			)(
				CardContent()(
					CardTitle()(
						Text(ed.SeriesEditionHead.Title()),
					),
				),
			)
		}),
		html.Form(
			attr.Method("POST"),
			attr.Action("/-/do/add-series-edition"),
		)(
			html.Input(attr.Type("hidden"), attr.Name("edition-id"), attr.Value(current.ID())),
			Button(ButtonSurface, ButtonSize2)(Text("Duplicate Edition")),
		),
	)
}

func appSeriesDetailEdition(
	sed *model.SeriesEdition,
	dls []*model.DownloadHead,
) html.Node {
	if sed == nil {
		return html.Div()(html.Text("Unknown Edition"))
	}
	return html.Div()(
		addTorrentButton("sed-id", sed.ID()),
		html.Div(
			attr.Class("v-media-download-list"),
		)(
			turbo.Sink("edition-torrents-"+sed.ID())(
				html.Range(dls, downloadListItem),
			),
		),
		appSeriesDetailEpisodeList(sed),
	)
}

func appSeriesDetailEpisodeList(sed *model.SeriesEdition) html.Node {
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
							html.RangeSeq(sn.Episodes(model.Significant), appSeriesDetailEpisodeListItem),
						),
					)
				})
			},
		),
	)
}

func appSeriesDetailEpisodeListItem(ep *model.Episode) html.Node {
	return FlexCol(Gap1)(
		FlexRow()(
			html.Div()(
				html.Text(ep.Label()),
			),
			html.Div()(
				DialogButton(ep.EditDialogURL(), ButtonGhost)(Icon("line/info-circle")),
			),
		),
		progressContainer(ep.ID(), ep.Progress()),
	)
}

func AppSeriesAddDialog(frameID string) html.Node {
	return Dialog(frameID,
		FlexCol(Gap2, Class("v-media-dialog"))(
			html.Div(
				attr.Class("v-media-dialog-fixed"),
			)(
				html.Text("Add Series"),
			),
			html.Form(
				attr.Action("/-/part/series-search"),
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

// AppEpisodeDialog renders the dialog for inspecting an
// episode's videos, renditions, and metadata.
func AppEpisodeDialog(
	frameID string,
	ep *model.Episode,
	videos []schema.Video,
	renditions []schema.RenditionForStreaming,
) html.Node {
	return Dialog(frameID,
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

			TextNode(FontBold, attr.Style("margin-top: 1rem"))(html.Text("Videos")),
			expr.IfElse(len(videos) == 0,
				func() html.Node {
					return html.Div(
						attr.Class("v-media-muted"),
					)(html.Text("No videos found"))
				},
				func() html.Node { return html.Group() },
			),
			html.Range(videos, func(v schema.Video) html.Node {
				return appEpisodeDialogVideo(v)
			}),

			TextNode(FontBold, attr.Style("margin-top: 1rem"))(html.Text("Renditions for Streaming")),
			expr.IfElse(len(renditions) == 0,
				func() html.Node {
					return html.Div(
						attr.Class("v-media-muted"),
					)(html.Text("No renditions found"))
				},
				func() html.Node { return html.Group() },
			),
			html.Range(renditions, func(r schema.RenditionForStreaming) html.Node {
				return appEpisodeDialogRendition(r)
			}),

			TextNode(FontBold, attr.Style("margin-top: 1rem"))(html.Text("Metadata")),
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

func appEpisodeDialogVideo(v schema.Video) html.Node {
	return html.Div(
		attr.Class("v-media-indent"),
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
		FlexRow(Gap2, attr.Style("margin-top: 0.5rem"))(
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

func appEpisodeDialogRendition(r schema.RenditionForStreaming) html.Node {
	return html.Div(
		attr.Class("v-media-indent"),
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

// AppSeriesSearchResults renders the search results for
// adding a series.
func AppSeriesSearchResults(results []SeriesSearchResult) html.Node {
	return turbo.Frame("results")(
		FlexCol(Gap4, Class("v-media-detail-body"))(
			html.Range(results, func(t SeriesSearchResult) html.Node {
				frameID := "tvmaze-" + strconv.Itoa(t.TVmaze.ID)
				return Card(CardSurface, CardSize3, Class("v-media-search-card"))(
					FlexRow(Gap4, attr.Style("height: 100%"))(
						Inset(InsetSideLeft, Class("v-media-search-poster"))(
							PosterImg(attr.Style("height: 100%"), attr.Src(t.TVmaze.Image.Medium())),
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
									return SeriesResultLink(t.Local.EditorURL())
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

func SeriesResultLink(editorURL string) html.Node {
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
