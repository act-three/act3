package view

import (
	"fmt"
	"strconv"
	"time"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/expr"
	"ily.dev/act3/model"
	"ily.dev/act3/msg"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
)

func AppSeries(
	s []*model.SeriesWork,
	selected *model.SeriesEdition,
	editions []*model.SeriesWork,
	dls []*model.DownloadHead,
) (title string, n domi.Node) {
	if selected == nil {
		return "Edit Series", appSeries(s, "", nil)
	}
	sr := selected.SeriesHead()
	return sr.Title(), appSeries(s, sr.ID(), appSeriesDetail(selected, editions, dls))
}

func AppSeriesEpisode(
	s []*model.SeriesWork,
	ep *model.Episode,
	episodeEditions []*model.Episode,
	renditions []schema.Rendition,
	uploads []model.Upload,
) (title string, n domi.Node) {
	return ep.Title(), appSeries(s, ep.SeriesHead().ID(), appEpisodeDetail(ep, episodeEditions, renditions, uploads))
}

func appSeries(s []*model.SeriesWork, selectedID string, detail domi.Node) domi.Node {
	return FlexCol(Class("v-media-page"))(
		ToolbarPrimary()(
			Button(onClick(&msg.SeriesAddOpen{}), ButtonSurface)(
				Text("Add Series"),
			),
		),
		Split()(
			List()(
				ListItems(s, func(ss *model.SeriesWork) bool {
					return ss.SeriesHead.ID() == selectedID
				}, AppSeriesListItem),
			),
			expr.IfElse(detail != nil,
				func() domi.Node {
					return detail
				},
				func() domi.Node {
					return Center(Class("v-media-muted"))(
						domi.Text("No Series Selected"),
					)
				},
			),
		),
	)
}

func AppSeriesListItem(ss *model.SeriesWork, attrs ...domi.Attr) domi.Node {
	return CardLink(ss.EditorPath(), CardGhost,
		group(attrs...),
	)(
		CardMedia()(html.Img(imgAttrs(ss.Poster()))),
		CardContent()(
			CardTitle()(domi.Text(ss.SeriesHead.Title())),
			CardDescription(LineClamp2)(
				iff(ss.PremieredOn() != "",
					func() domi.Node { return domi.Text(ss.PremieredOn()) }),

				domi.Text(ss.Status()),
			),
		),
	)
}

func appSeriesDetail(
	sed *model.SeriesEdition,
	editions []*model.SeriesWork,
	dls []*model.DownloadHead,
) domi.Node {
	sr := sed.SeriesHead()
	return FlexCol(Class("v-media-detail"))(
		ScrollY(
			Class("v-media-detail-body"),
		)(
			SettingsPage()(
				expr.IfElse(len(editions) < 2,
					func() domi.Node {
						return Group(
							FlexCol(Gap6)(
								SettingsContent()(
									TextNode(Size6)(domi.Text(sr.Title())),
									Box()(
										Link(
											sr.TheaterPath(),
										)(Text("View in Theater", Size3,
											Style("display: inline-block"),
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
					func() domi.Node {
						return Group(
							FlexCol(Gap4)(
								SettingsContent()(
									TextNode(Size6)(domi.Text(sr.Title())),
								),
								SettingsGroup()(
									seriesTitleItem(sr),
								),
							),

							appSeriesEditionList(editions, sed),

							FlexCol(Gap6)(
								SettingsContent()(
									TextNode(Size4)(domi.Text(sed.Label())),
									Box()(
										Link(
											sed.TheaterPath(),
										)(Text("View in Theater", Size2,
											// TODO(april): maybe make this the default for Text
											Style("display: inline-block"),
										)),
									),
								),

								SettingsGroup()(
									SettingsItem()(
										SettingsItemLabel()(
											SettingsItemLabelTitle("Edition"),
										),
										SettingsTextField(sed.Label(), func(v string) msg.Msg {
											return &msg.SeriesEditionSetLabel{ID: sed.ID(), Label: v}
										}),
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
					rangeNodes(dls, downloadListItem),
				),

				appSeriesDetailSeasonList(sed),

				SettingsGroup()(
					SettingsItem()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("New Season"),
						),

						Button(onClick(&msg.SeasonAdd{EditionID: sed.ID()}),
							ButtonGhost, ButtonSize2,
						)(Text("Add")),
					),
				),

				SettingsGroup()(
					SettingsItem()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Edition"),
							SettingsItemLabelDescription("Create a new edition by duplicating this one"),
						),

						Button(onClick(&msg.SeriesEditionAdd{EditionID: sed.ID()}),
							ButtonGhost, ButtonSize2,
						)(Text("Duplicate")),
					),
				),

				SettingsGroup()(
					SettingsItem()(
						SettingsItemLabel()(
							expr.IfElse(len(editions) > 1,
								func() domi.Node {
									return SettingsItemLabelTitle("Delete Edition")
								},
								func() domi.Node {
									return SettingsItemLabelTitle("Delete Series")
								},
							),
							SettingsItemLabelDescription("Deleted items remain in Trash for 30 days"),
						),
						expr.IfElse(len(editions) > 1,
							func() domi.Node {
								return trashForm(sed.ID())
							},
							func() domi.Node {
								return trashForm(sr.ID())
							},
						),
					),
				),
			),
		),
	)
}

func seriesTitleItem(sr *model.SeriesHead) domi.Node {
	return SettingsItem()(
		SettingsItemLabel()(
			SettingsItemLabelTitle("Title"),
		),
		SettingsTextField(sr.Title(), func(v string) msg.Msg {
			return &msg.SeriesSetTitle{ID: sr.ID(), Title: v}
		}),
	)
}

func seriesPosterItem(sed *model.SeriesEdition) domi.Node {
	return SettingsItem()(
		SettingsItemLabel()(
			SettingsItemLabelTitle("Poster"),
		),
		buttonImageEdit(
			&msg.ImageDialogOpen{ID: sed.ID()},
			sed.Poster(),
			AspectPoster,
		),
	)
}

func AppSeriesEditionPosterDialog(sed *model.SeriesEdition) domi.Node {
	return imageDialog(&msg.DialogClose{}, AspectPoster)(
		buttonUpload()(
			Hidden("sed-id", sed.ID()),
			PosterImg(AspectPoster, PosterFill, imgAttrs(sed.Poster())),
		),
	)
}

func seriesSummarySection(sed *model.SeriesEdition) domi.Node {
	return FlexCol(Gap2)(
		SettingsContent()(Text("Summary", Size2)),
		SettingsTextArea(sed.Summary(), func(v string) msg.Msg {
			return &msg.SeriesEditionSetSummary{ID: sed.ID(), Summary: v}
		}),
	)
}

func appSeriesEditionList(editions []*model.SeriesWork, current *model.SeriesEdition) domi.Node {
	return FlexCol(Gap2)(
		rangeNodes(editions, func(ed *model.SeriesWork) domi.Node {
			var card domi.Element
			if ed.SeriesEditionHead.ID() == current.ID() {
				card = Card(CardSurface, CardSize1, CardSelected)
			} else {
				card = CardLink(ed.EditorPath(), CardSurface, CardSize1)
			}
			return card(
				FlexRow()(
					CardContent()(
						CardTitle()(
							domi.Text(ed.SeriesEditionHead.Label()),
						),
						CardDescription()(
							seriesTheaterPathText(&ed.SeriesHead, &ed.SeriesEditionHead),
						),
					),
				),
			)
		}),
	)
}

func appSeriesDetailSeasonList(sed *model.SeriesEdition) domi.Node {
	return FlexCol(Gap8)(
		rangeSeq(sed.Seasons(), func(sn *model.Season) domi.Node {
			return appSeriesDetailSeasonItem(sn)
		}),
	)
}

func appSeriesDetailSeasonItem(sn *model.Season) domi.Node {
	return SettingsGroup()(
		SettingsGroupHead(Class("v-season-group-head"))(
			SettingsItemLabel(Class("v-season-label"),
				stimulus.Controller("season-title"),
				stimulus.Value("season-title", "mode")("display"),
			)(
				html.Div(Class("v-season-display"))(
					FlexRow(Class("v-season-title-row"))(
						TextNode(Size2)(domi.Text(sn.Title())),
						Button(ButtonCircle, ButtonGhost, ButtonSize1,
							Class("v-season-title-edit-button"),
							stimulus.Action("click->season-title#edit"),
						)(Icon("line/edit-02")),
					),
					SettingsItemLabelDescription(fmt.Sprintf("%d Episodes", sn.NumEpisodes(model.Significant))),
				),
				html.Div(Class("v-season-edit"),
					stimulus.Action("change->season-title#display"),
					stimulus.Action("keydown.esc->season-title#display"),
				)(
					SettingsTextField(sn.Title(), func(v string) msg.Msg {
						return &msg.SeasonSetTitle{ID: sn.ID(), Title: v}
					}),
				),
			),
			FlexRow(Gap2)(
				Button(onClick(&msg.EpisodeCreate{SeasonID: sn.ID()}),
					ButtonGhost, ButtonSize2,
				)(Text("Add Episode")),
				trashForm(sn.ID()),
			),
		),
		domi.Keyed("div")(
			Class("u-contents"),
			stimulus.Controller("sortable"),
			stimulus.Action("keydown.esc@document->sortable#cancel"),
			Attr("data-season-id")(sn.ID()),
			onEpisodeMove(),
		)(func(yield func(string, domi.Node) bool) {
			for ep := range sn.Episodes(model.AnyEpisode) {
				if !yield(ep.ID(), appSeriesDetailEpisodeListItem(ep)) {
					return
				}
			}
		}),
	)
}

func appSeriesDetailEpisodeListItem(ep *model.Episode) domi.Node {
	icon := Group()
	switch ep.State() {
	case model.EpIsEmpty:
		icon = Icon("line/x")
	case model.EpIsDownloading:
		// TODO(april): icon = LiveProgress(...)
		icon = Spinner()
	case model.EpIsPlayable:
		icon = Icon("solid/check-circle")
	}
	return SettingsItem(Attr("data-episode-id")(ep.ID()))(
		FlexRow(Style("align-items:center"), Gap4)(
			SettingsItemLabelIcon()(icon),
			SettingsItemLabel()(
				SettingsItemLabelTitle(ep.Label()),
				progressContainer(ep.ID(), ep.Progress()),
			),
		),
		ButtonLink(ep.EditorPath(), ButtonGhost)(Icon("line/info-circle")),
		Grip(),
	)
}

// AppSeriesAddDialog renders the add-series dialog: a TVmaze search
// box and its results. While a search is in flight, searching shows
// a spinner in place of the previous results.
func AppSeriesAddDialog(query string, searching bool, results []model.SeriesSearchResult) domi.Node {
	return dialog(&msg.DialogClose{})(
		FlexCol(Gap2, Class("v-media-dialog"))(
			html.Div(
				Class("v-media-dialog-fixed"),
			)(
				domi.Text("Add Series"),
			),
			html.Form(
				onSubmit("q", func(v string) msg.Msg {
					return &msg.SeriesSearch{Query: v}
				}),
			)(
				InputText(
					Attr("autofocus")(""),
					Class("v-media-dialog-fixed"),
					attr.Name("q"),
					attr.Value(query),
				),
			),
			html.Div(
				Class("v-media-dialog-results"),
			)(
				expr.IfElse(searching,
					func() domi.Node {
						return Spinner(Class("v-media-dialog-spinner"))
					},
					func() domi.Node {
						return AppSeriesSearchResults(results)
					},
				),
			),
		),
	)
}

// appEpisodeDetail renders the page for inspecting an
// episode's videos, renditions, and metadata.
func appEpisodeDetail(
	ep *model.Episode,
	episodeEditions []*model.Episode,
	renditions []schema.Rendition,
	uploads []model.Upload,
) domi.Node {
	videos := ep.Videos()
	backLabel := ep.SeriesHead().Title()
	if ep.SeriesEditionHead().Slug() != "" {
		backLabel += " — " + ep.SeriesEditionHead().Label()
	}
	return ScrollY(Style("padding:0.5rem"))(
		FlexRow()(
			ButtonLink(model.SeriesEditionEditorPath(ep.SeriesHead(), ep.SeriesEditionHead()), ButtonGhost)(
				Label("line/chevron-left", backLabel),
			),
		),
		SettingsPage()(
			FlexCol(Gap8)(
				FlexCol(Gap4)(
					expr.IfElse(len(episodeEditions) > 1,
						func() domi.Node {
							return FlexCol(Gap8)(
								SettingsGroup()(
									SettingsGroupHead(Class("v-season-group-head"))(
										SettingsItemLabel()(
											SettingsItemLabelTitle(fmt.Sprintf("%d Editions", len(episodeEditions))),
											SettingsItemLabelDescription("Editing this episode affects these editions."),
										),
									),
									rangeNodes(episodeEditions, func(eped *model.Episode) domi.Node {
										return SettingsItem()(
											SettingsItemLabel()(
												TextNode(Size2)(
													domi.Text(eped.SeriesEditionHead().Label()),
													domi.Text(" "),
													domi.Text(eped.SnnEnn()),
												),
											),
											AppEpisodeEditionButton(eped.SeasonHead().ID(), eped.ID(), true, 0),
										)
									}),
								),
								SettingsContent()(
									TextNode(Size5)(domi.Text(ep.Title())),
								),
							)
						},
						func() domi.Node {
							return SettingsContent()(
								TextNode(Size3)(
									domi.Text(ep.SeriesEditionHead().Label()),
									domi.Text(" "),
									domi.Text(ep.SnnEnn()),
								),
								TextNode(Size5)(domi.Text(ep.Title())),
							)
						},
					),

					SettingsGroup()(
						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Title"),
							),
							SettingsTextField(ep.Title(), func(v string) msg.Msg {
								return &msg.EpisodeSetTitle{ID: ep.ID(), Title: v}
							}),
						),
						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Airdate"),
							),
							SettingsTextField(ep.Airdate(), func(v string) msg.Msg {
								return &msg.EpisodeSetAirdate{ID: ep.ID(), Airdate: v}
							}),
						),
						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Thumbnail"),
							),
							buttonImageEdit(
								&msg.ImageDialogOpen{ID: ep.ID()},
								ep.Thumbnail(),
								AspectThumbnail,
							),
						),
						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Type"),
							),
							SettingsButtonRow()(
								episodeTypeButton(ep, "regular", "Regular"),
								episodeTypeButton(ep, "significant_special", "Special"),
								episodeTypeButton(ep, "insignificant_special", "Insignificant"),
							),
						),
					),
				),

				FlexCol(Gap2)(
					SettingsContent()(Text("Summary", Size2)),
					SettingsTextArea(ep.Summary(), func(v string) msg.Msg {
						return &msg.EpisodeSetSummary{ID: ep.ID(), Summary: v}
					}),
				),

				SettingsGroup()(
					SettingsGroupHead()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Upload"),
						),
						uploadVideoControl("ep-id", ep.ID(), uploads),
					),
				),

				TextNode(TextBold, Style("margin-top: 1rem"))(domi.Text("Videos")),
				expr.IfElse(len(videos) == 0,
					func() domi.Node {
						return html.Div(
							Class("v-media-muted"),
						)(domi.Text("No videos found"))
					},
					func() domi.Node { return Group() },
				),
				rangeNodes(videos, func(v *model.Video) domi.Node {
					return appEpisodeDialogVideo(ep, v)
				}),

				TextNode(TextBold, Style("margin-top: 1rem"))(domi.Text("Renditions for Streaming")),
				expr.IfElse(len(renditions) == 0,
					func() domi.Node {
						return html.Div(
							Class("v-media-muted"),
						)(domi.Text("No renditions found"))
					},
					func() domi.Node { return Group() },
				),
				rangeNodes(renditions, func(r schema.Rendition) domi.Node {
					return appEpisodeDialogRendition(r)
				}),

				SettingsGroup()(
					SettingsItem()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Delete Episode"),
							SettingsItemLabelDescription("Deleted items remain in Trash for 30 days"),
						),
						trashForm(ep.ID()),
					),
				),
			),
		),
	)
}

// AppEpisodeEditionButton renders the per-edition remove/undo control.
// undoSortKey is only consulted when inEdition is false: it's the
// SortKey the undo action should reinsert the episode at.
func AppEpisodeEditionButton(seasonID, episodeID string, inEdition bool, undoSortKey int64) domi.Node {
	return expr.IfElse(inEdition,
		func() domi.Node {
			return Button(
				onClick(&msg.SeasonRemoveEpisode{SeasonID: seasonID, EpisodeID: episodeID}),
				Destructive, SettingsHover,
			)(
				Text("Remove"),
			)
		},
		func() domi.Node {
			return FlexRow(Gap4, Style("align-items:center"))(
				Button(
					onClick(&msg.SeasonAddEpisode{SeasonID: seasonID, EpisodeID: episodeID, SortKey: undoSortKey}),
					ButtonGhost,
				)(
					Text("Undo"),
				),
				Text("Removed"),
			)
		},
	)
}

func appEpisodeDialogVideo(ep *model.Episode, v *model.Video) domi.Node {
	return html.Div(
		Class("v-media-indent"),
	)(
		html.Div()(
			domi.Text("ID: "),
			domi.Text(v.ID()),
		),
		html.Div()(
			domi.Text("Name: "),
			domi.Text(v.Name()),
		),
		html.Div()(
			domi.Text("Original Key: "),
			domi.Text(v.OriginalKey()),
		),
		html.Div()(
			domi.Text("Playable: "),
			domi.Text(strconv.FormatBool(v.Playable())),
		),
		FlexRow(Gap2, Style("margin-top: 0.5rem"))(
			activeVideoControl(v, &msg.EpisodeVideoSetActive{EpisodeID: ep.ID(), VideoID: v.ID()}),
			Button(onClick(&msg.VideoReimport{ID: v.ID()}), Destructive)(
				domi.Text("Re-import"),
			),
			expr.IfElse(v.OriginalKey() != "",
				func() domi.Node {
					return Button(onClick(&msg.VideoReencode{ID: v.ID()}), Destructive)(
						domi.Text("Re-encode"),
					)
				},
				func() domi.Node { return Group() },
			),
			trashForm(v.ID()),
		),
	)
}

func episodeTypeButton(ep *model.Episode, value, label string) domi.Node {
	return settingsButtonRowItem(ep.Type() == value, &msg.EpisodeSetType{
		ID:   ep.ID(),
		Type: value,
	}, Text(label))
}

// activeVideoControl renders either an "Active" badge (when v is the
// active video) or a "Set active" button (when v is playable and not
// active). Unplayable videos render nothing — they cannot be made
// active.
func activeVideoControl(v *model.Video, setActive msg.Msg) domi.Node {
	switch {
	case v.Active():
		return TextNode(TextBold)(domi.Text("Active"))
	case v.Playable():
		return Button(onClick(setActive))(domi.Text("Set active"))
	default:
		return Group()
	}
}

func appEpisodeDialogRendition(r schema.Rendition) domi.Node {
	return html.Div(
		Class("v-media-indent"),
	)(
		html.Div()(
			domi.Text("ID: "),
			domi.Text(r.ID),
		),
		html.Div()(
			domi.Text("Video ID: "),
			domi.Text(r.VideoID),
		),
		html.Div()(
			domi.Text("Codec: "),
			domi.Text(r.Codec),
		),
		html.Div()(
			domi.Textf("Target Bitrate: %d kbit/s", r.TargetBitrate),
		),
		html.Div()(
			domi.Textf("Remux: %v", r.Remux != 0),
		),
		expr.IfElse(r.MaxHeight != 0,
			func() domi.Node {
				return html.Div()(
					domi.Textf("Max Height: %d", r.MaxHeight),
				)
			},
			func() domi.Node { return Group() },
		),
		expr.IfElse(r.MaxFPS != 0,
			func() domi.Node {
				return html.Div()(
					domi.Textf("Max FPS: %d", r.MaxFPS),
				)
			},
			func() domi.Node { return Group() },
		),
		expr.IfElse(r.Key != "",
			func() domi.Node {
				return html.Div()(
					domi.Text("Key: "),
					domi.Text(r.Key),
				)
			},
			func() domi.Node { return Group() },
		),
	)
}

// AppSeriesSearchResults renders the search results for
// adding a series.
func AppSeriesSearchResults(results []model.SeriesSearchResult) domi.Node {
	return FlexCol(Gap4, Class("v-media-detail-body"))(
		rangeNodes(results, func(t model.SeriesSearchResult) domi.Node {
			return Card(CardSurface, CardSize3, Class("v-media-search-card"))(
				FlexRow(Gap4, Style("height: 100%"))(
					Inset(InsetSideLeft, Class("v-media-search-poster"))(
						PosterImg(AspectPoster, Style("height: 100%"), attr.Src(t.Show.Image.Medium())),
					),
					FlexCol(Gap2)(
						domi.Text(t.Show.Name),
						expr.IfElse(t.Local == nil,
							func() domi.Node {
								return Button(
									onClick(&msg.SeriesAdd{TVmazeID: t.Show.ID}),
									ButtonSurface,
								)(domi.Text("Add"))
							},
							func() domi.Node {
								return seriesResultLink(t.Local.EditorPath())
							},
						),
						TextNode(LineClamp3)(domi.Safe(t.Show.Summary)),
					),
				),
			)
		}),
	)
}

func seriesResultLink(editorURL string) domi.Node {
	return FlexRow(Gap2)(
		Label("line/check-circle", "In Library"),
		ButtonLink(editorURL)(
			Text("Edit"),
		),
	)
}

func seriesTheaterPathText(sr *model.SeriesHead, sed *model.SeriesEditionHead) domi.Node {
	if sed.Slug() == "" {
		return Group(domi.Text("/"), domi.Text(sr.Slug()))
	}
	return Group(
		domi.Text("/"), domi.Text(sr.Slug()),
		domi.Text("/"), domi.Text(sed.Slug()),
	)
}

func AppEpisodeThumbnailDialog(ep *model.EpisodeHead) domi.Node {
	return imageDialog(&msg.DialogClose{}, AspectThumbnail)(
		buttonUpload()(
			Hidden("ep-id", ep.ID()),
			PosterImg(AspectThumbnail, PosterFill, imgAttrs(ep.Thumbnail())),
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
