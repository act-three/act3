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
			DialogButton("/-/dialog/series-add", ButtonSurface)(
				Text("Add Series"),
			),
		),
		Split()(
			List("/app/series/", "detail")(
				turbo.StreamTarget(AppSeriesListItems)(
					ListItems(s, AppSeriesListItem),
				),
			),
			turbo.Frame("detail", turbo.Advance())(
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
		),
	))
}

func AppSeriesListItem(ss *model.SeriesWork, attrs ...attr.Node) html.Node {
	return Card(CardGhost,
		attr.Group(attrs...),
		ListID(ss.SeriesHead.ID()),
		ListURL(ss.EditorPath()),
	)(
		CardMedia()(html.Img(attr.Src(ss.TVmazeImageURL()))),
		CardContent()(
			CardTitle()(seriesTitle(ss.SeriesHead.ID(), ss.Title())),
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
			SettingsPage()(
				expr.IfElse(len(editions) < 2,
					func() html.Node {
						return Group(
							FlexCol(Gap6)(
								SettingsContent()(
									TextNode(Size6)(seriesTitle(sr.ID(), sr.Title())),
									Box()(
										Link(
											sr.TheaterPath(),
											turbo.DataFrame("_top"),
										)(Text("View in Theater", Size3,
											attr.Style("display: inline-block"),
										)),
									),
								),
								SettingsGroup()(
									seriesTitleItem(sr),
									seriesPosterItem(sed),
								),
								seriesSummarySection(sed),
							),
						)

					},
					func() html.Node {
						return Group(
							FlexCol(Gap4)(
								SettingsContent()(
									TextNode(Size6)(seriesTitle(sr.ID(), sr.Title())),
								),
								SettingsGroup()(
									seriesTitleItem(sr),
								),
							),

							appSeriesEditionList(editions, sed),

							FlexCol(Gap6)(
								SettingsContent()(
									TextNode(Size4)(seriesEditionLabel(sed.ID(), sed.Label())),
									Box()(
										Link(
											sed.TheaterPath(),
											turbo.DataFrame("_top"),
										)(Text("View in Theater", Size2,
											// TODO(april): maybe make this the default for Text
											attr.Style("display: inline-block"),
										)),
									),
								),

								SettingsGroup()(
									SettingsItem()(
										SettingsItemLabel()(
											SettingsItemLabelTitle("Edition"),
										),
										SettingsTextField("/-/do/series-edition-set-label", "label", sed.Label(), seriesEditionLabelAttrClass(sed.ID()))(
											Hidden("id", sed.ID()),
										),
									),
									seriesPosterItem(sed),
								),
								seriesSummarySection(sed),
							),
						)
					},
				),

				SettingsGroup()(
					SettingsGroupHead()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Downloads"),
						),
						addTorrentButton("sed-id", sed.ID()),
					),
					turbo.StreamTarget("edition-torrents-"+sed.ID(), SettingsGroupItems)(
						html.Range(dls, downloadListItem),
					),
				),

				appSeriesDetailSeasonList(sed),

				SettingsGroup()(
					SettingsItem()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Edition"),
							SettingsItemLabelDescription("Create a new edition by duplicating this one"),
						),

						html.Form(
							attr.Method("POST"),
							attr.Action("/-/do/series-edition-add"),
						)(
							Hidden("edition-id", sed.ID()),
							Button(ButtonGhost, ButtonSize2)(Text("Duplicate")),
						),
					),
				),
			),
		),
	)
}

func seriesTitleItem(sr *model.SeriesHead) html.Node {
	return SettingsItem()(
		SettingsItemLabel()(
			SettingsItemLabelTitle("Title"),
		),
		SettingsTextField("/-/do/series-set-title", "title", sr.Title(), seriesTitleAttrClass(sr.ID()))(
			Hidden("id", sr.ID()),
		),
	)
}

func seriesPosterItem(sed *model.SeriesEdition) html.Node {
	return SettingsItem()(
		SettingsItemLabel()(
			SettingsItemLabelTitle("Poster"),
		),
		ImageFrame(attr.Style("width:30px"))(
			PosterImg(PosterFill, attr.Src(sed.TVmazeImageURL())),
		),
	)
}

func seriesSummarySection(sed *model.SeriesEdition) html.Node {
	return FlexCol(Gap2)(
		SettingsContent()(Text("Summary", Size2)),
		SettingsTextArea("/-/do/series-edition-set-summary", "summary", sed.Summary())(
			Hidden("id", sed.ID()),
		),
	)
}

func appSeriesEditionList(editions []*model.SeriesWork, current *model.SeriesEdition) html.Node {
	return FlexCol(Gap2)(
		html.Range(editions, func(ed *model.SeriesWork) html.Node {
			selected := attr.Group()
			href := attr.Group()
			if ed.SeriesEditionHead.ID() == current.ID() {
				selected = CardSelected
			} else {
				href = attr.Href(ed.EditorPath())
			}
			return Card(
				CardSurface,
				CardSize1,
				href,
				selected,
			)(
				FlexRow()(
					CardContent()(
						CardTitle()(
							seriesEditionLabel(ed.SeriesEditionHead.ID(), ed.SeriesEditionHead.Label()),
						),
						CardDescription()(
							seriesTheaterPathText(ed.SeriesHead.ID(), ed.SeriesHead.Slug(), ed.SeriesEditionHead.ID(), ed.SeriesEditionHead.Slug()),
						),
					),
				),
			)
		}),
	)
}

func appSeriesDetailSeasonList(sed *model.SeriesEdition) html.Node {
	return FlexCol(Gap4)(
		html.RangeSeq(sed.Seasons(), func(sn *model.Season) html.Node {
			return SettingsGroup()(
				SettingsGroupHead()(
					SettingsItemLabel()(
						SettingsItemLabelTitle(sn.Name()),
						SettingsItemLabelDescription(fmt.Sprintf("%d Episodes", sn.NumEpisodes(model.Significant))),
					),
				),
				// TODO(april): choose a better name when this is hooked up
				turbo.StreamTarget("series-edition-season-"+sed.ID())(
					html.RangeSeq(sn.Episodes(model.Significant), appSeriesDetailEpisodeListItem),
				),
			)
		}),
	)
}

func appSeriesDetailEpisodeListItem(ep *model.Episode) html.Node {
	return SettingsItem()(
		SettingsItemLabel()(
			SettingsItemLabelTitle(ep.Label()),
			progressContainer(ep.ID(), ep.Progress()),
		),
		DialogButton(ep.EditDialogPath(), ButtonGhost)(Icon("line/info-circle")),
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
				turbo.Frame("results")(Spinner(Class("v-media-dialog-spinner"))),
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
				seriesTitle(ep.SeriesHead().ID(), ep.SeriesHead().Title()),
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
				attr.Action("/-/do/video-reimport/"+v.ID),
				attr.Method("POST"),
			)(
				Button(Destructive)(
					html.Text("Re-import"),
				),
			),
			expr.IfElse(v.OriginalHash != "",
				func() html.Node {
					return html.Form(
						attr.Action("/-/do/video-reencode/"+v.ID),
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
											attr.Action("/-/do/series-add"),
											turbo.DataFrame(frameID),
										)(
											Hidden("id", strconv.Itoa(t.TVmaze.ID)),
											Button(ButtonSurface)(html.Text("Add")),
										),
									)
								},
								func() html.Node {
									return SeriesResultLink(t.Local.EditorPath())
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

func seriesEditionLabelTargetClass(id string) string {
	return "series-edition-" + id + "-label"
}

func seriesEditionLabelAttrClass(id string) string {
	return "series-edition-" + id + "-label-attr"
}

func SeriesEditionSetLabel(id, label string) html.Node {
	return html.Group(
		turbo.ReplaceTargets("."+seriesEditionLabelTargetClass(id), turbo.Morph)(
			seriesEditionLabel(id, label),
		),
		SettingsTextFieldSetValue("."+seriesEditionLabelAttrClass(id), label),
	)
}

func seriesEditionLabel(id, label string) html.Node {
	return html.Span(Class(seriesEditionLabelTargetClass(id)))(html.Text(label))
}

func seriesTitleTargetClass(id string) string {
	return "series-" + id + "-title"
}

func seriesTitleAttrClass(id string) string {
	return "series-" + id + "-title-attr"
}

func SeriesSetTitle(id, title string) html.Node {
	return html.Group(
		turbo.ReplaceTargets("."+seriesTitleTargetClass(id), turbo.Morph)(
			seriesTitle(id, title),
		),
		SettingsTextFieldSetValue("."+seriesTitleAttrClass(id), title),
	)
}

func seriesTitle(id, title string) html.Node {
	return html.Span(Class(seriesTitleTargetClass(id)))(html.Text(title))
}

func seriesSlugTargetClass(id string) string {
	return "series-" + id + "-slug"
}

func seriesSlug(id, slug string) html.Node {
	return html.Span(Class(seriesSlugTargetClass(id)))(html.Text(slug))
}

func seriesEditionSlugTargetClass(id string) string {
	return "series-edition-" + id + "-slug"
}

func seriesEditionSlug(id, slug string) html.Node {
	return html.Span(Class(seriesEditionSlugTargetClass(id)))(html.Text(slug))
}

func seriesTheaterPathText(seriesID, seriesSlugVal, editionID, editionSlugVal string) html.Node {
	if editionSlugVal == "" {
		return Group(html.Text("/"), seriesSlug(seriesID, seriesSlugVal))
	}
	return Group(
		html.Text("/"), seriesSlug(seriesID, seriesSlugVal),
		html.Text("/"), seriesEditionSlug(editionID, editionSlugVal),
	)
}

func SeriesSetSlug(id, slug string) html.Node {
	return turbo.ReplaceTargets("."+seriesSlugTargetClass(id), turbo.Morph)(
		seriesSlug(id, slug),
	)
}

func SeriesEditionSetSlug(id, slug string) html.Node {
	return turbo.ReplaceTargets("."+seriesEditionSlugTargetClass(id), turbo.Morph)(
		seriesEditionSlug(id, slug),
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
