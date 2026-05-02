package view

import (
	"fmt"
	"path"
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

const AppSeriesListItems = "series-list-items"

func AppSeries(
	title string,
	s []*model.SeriesWork,
	detail ...html.Node,
) (string, html.Node) {
	return title, FlexCol(Class("v-media-page"))(
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
	)
}

func AppSeriesListItem(ss *model.SeriesWork, attrs ...attr.Node) html.Node {
	return Card(CardGhost,
		group(attrs...),
		ListID(ss.SeriesHead.ID()),
		ListURL(ss.EditorPath()),
	)(
		CardMedia()(html.Img(imgAttrs(ss.PosterField()))),
		CardContent()(
			CardTitle()(LiveText(ss.SeriesHead.TitleField())),
			CardDescription(LineClamp2)(
				html.If(ss.PremieredOn() != "",
					func() html.Node { return html.Text(ss.PremieredOn()) },
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
									TextNode(Size6)(LiveText(sr.TitleField())),
									Box()(
										Link(
											sr.TheaterPath(),
											turbo.DataFrame("_top"),
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
					func() html.Node {
						return Group(
							FlexCol(Gap4)(
								SettingsContent()(
									TextNode(Size6)(LiveText(sr.TitleField())),
								),
								SettingsGroup()(
									seriesTitleItem(sr),
								),
							),

							appSeriesEditionList(editions, sed),

							FlexCol(Gap6)(
								SettingsContent()(
									TextNode(Size4)(LiveText(sed.LabelField())),
									Box()(
										Link(
											sed.TheaterPath(),
											turbo.DataFrame("_top"),
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
										SettingsTextField("/-/do/series-edition-set-label", "label", sed.Label(), LiveAddr(sed.LabelAddr()))(
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
					turbo.StreamTarget("edition-torrents-"+sed.ID())(
						html.Range(dls, downloadListItem),
					),
				),

				appSeriesDetailSeasonList(sed),

				SettingsGroup()(
					SettingsItem()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("New Season"),
						),

						html.Form(
							attr.Method("POST"),
							attr.Action("/-/do/season-add"),
						)(
							Hidden("edition-id", sed.ID()),
							Button(ButtonGhost, ButtonSize2)(Text("Add")),
						),
					),
				),

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

				SettingsGroup()(
					SettingsItem()(
						SettingsItemLabel()(
							expr.IfElse(len(editions) > 1,
								func() html.Node {
									return SettingsItemLabelTitle("Delete Edition")
								},
								func() html.Node {
									return SettingsItemLabelTitle("Delete Series")
								},
							),
							SettingsItemLabelDescription("Deleted items remain in Trash for 30 days"),
						),
						expr.IfElse(len(editions) > 1,
							func() html.Node {
								return trashForm(sed.ID())
							},
							func() html.Node {
								return trashForm(sr.ID())
							},
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
		SettingsTextField("/-/do/series-set-title", "title", sr.Title(), LiveAddr(sr.TitleAddr()))(
			Hidden("id", sr.ID()),
		),
	)
}

func seriesPosterItem(sed *model.SeriesEdition) html.Node {
	return SettingsItem()(
		SettingsItemLabel()(
			SettingsItemLabelTitle("Poster"),
		),
		buttonImageEdit(
			"/-/dialog/series-edition-poster/"+sed.ID(),
			sed.Poster(), sed.PosterAddr(),
			AspectPoster,
		),
	)
}

func AppSeriesEditionPosterDialog(sed *model.SeriesEdition) html.Node {
	return ImageDialogStream(AspectPoster)(
		buttonUpload()(
			Hidden("sed-id", sed.ID()),
			PosterImg(AspectPoster, PosterFill, imgAttrs(sed.PosterField())),
		),
	)
}

func seriesSummarySection(sed *model.SeriesEdition) html.Node {
	return FlexCol(Gap2)(
		SettingsContent()(Text("Summary", Size2)),
		SettingsTextArea("/-/do/series-edition-set-summary", "summary", sed.Summary(), LiveAddr(sed.SummaryAddr()))(
			Hidden("id", sed.ID()),
		),
	)
}

func appSeriesEditionList(editions []*model.SeriesWork, current *model.SeriesEdition) html.Node {
	return FlexCol(Gap2)(
		html.Range(editions, func(ed *model.SeriesWork) html.Node {
			selected := group()
			href := group()
			if ed.SeriesEditionHead.ID() == current.ID() {
				selected = CardSelected
			} else {
				href = Href(ed.EditorPath())
			}
			return turbo.StreamTarget("edition-tab-" + ed.SeriesEditionHead.ID())(
				Card(
					CardSurface,
					CardSize1,
					href,
					selected,
				)(
					FlexRow()(
						CardContent()(
							CardTitle()(
								LiveText(ed.SeriesEditionHead.LabelField()),
							),
							CardDescription()(
								seriesTheaterPathText(&ed.SeriesHead, &ed.SeriesEditionHead),
							),
						),
					),
				),
			)
		}),
	)
}

func appSeriesDetailSeasonList(sed *model.SeriesEdition) html.Node {
	return FlexCol(Gap8)(
		turbo.StreamTarget("edition-seasons-" + sed.ID())(
			html.RangeSeq(sed.Seasons(), func(sn *model.Season) html.Node {
				return appSeriesDetailSeasonItem(sn)
			}),
		),
	)
}

func appSeriesDetailSeasonItem(sn *model.Season) html.Node {
	return turbo.StreamTarget("season-" + sn.ID())(
		SettingsGroup()(
			SettingsGroupHead(Class("v-season-group-head"))(
				SettingsItemLabel(Class("v-season-label"),
					stimulus.Controller("season-title"),
					stimulus.Value("season-title", "mode")("display"),
				)(
					html.Div(Class("v-season-display"))(
						FlexRow(Class("v-season-title-row"))(
							TextNode(Size2)(LiveText(sn.TitleField())),
							Button(ButtonCircle, ButtonGhost, ButtonSize1,
								Class("v-season-title-edit-button"),
								stimulus.Action("click->season-title#edit"),
							)(Icon("line/edit-02")),
						),
						SettingsItemLabelDescription(fmt.Sprintf("%d Episodes", sn.NumEpisodes(model.Significant))),
					),
					html.Div(Class("v-season-edit"),
						stimulus.Action("settings-text-field:commit->season-title#display"),
						stimulus.Action("settings-text-field:error->season-title#display"),
						stimulus.Action("settings-text-field:cancel->season-title#display"),
					)(
						SettingsTextField("/-/do/season-set-title", "title", sn.Title(), LiveAddr(sn.TitleAddr()))(
							html.Input(attr.Type("hidden"), attr.Name("id"), attr.Value(sn.ID())),
						),
					),
				),
				FlexRow(Gap2)(
					ActionButton("/-/do/series-episode-add",
						map[string]string{"season-id": sn.ID()},
						ButtonGhost, ButtonSize2,
					)(Text("Add Episode")),
					trashForm(sn.ID()),
				),
			),
			turbo.StreamTarget("season-episodes-"+sn.ID(),
				stimulus.Controller("sortable"),
				stimulus.Action("keydown.esc@document->sortable#cancel"),
				Attr("data-season-id")(sn.ID()),
			)(
				html.RangeSeq(sn.Episodes(model.AnyEpisode), appSeriesDetailEpisodeListItem),
			),
		),
	)
}

func appSeriesDetailEpisodeListItem(ep *model.Episode) html.Node {
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
		Button(Href(ep.EditorPath()), ButtonGhost)(Icon("line/info-circle")),
		Grip(),
	)
}

func AppSeriesAddDialog() html.Node {
	return DialogStream(
		FlexCol(Gap2, Class("v-media-dialog"))(
			html.Div(
				Class("v-media-dialog-fixed"),
			)(
				html.Text("Add Series"),
			),
			html.Form(
				attr.Action("/-/part/series-search"),
				Attr("data-turbo-frame")("results"),
			)(
				InputText(
					Attr("autofocus"),
					Class("v-media-dialog-fixed"),
					attr.Name("q"),
				),
			),
			html.Div(
				Class("v-media-dialog-results"),
			)(
				turbo.Frame("results")(Spinner(Class("v-media-dialog-spinner"))),
			),
		),
	)
}

// AppEpisodeDetail renders the page for inspecting an
// episode's videos, renditions, and metadata.
func AppEpisodeDetail(
	ep *model.Episode,
	episodeEditions []*model.Episode,
	renditions []schema.Rendition,
) html.Node {
	videos := ep.Videos()
	backLabel := ep.SeriesHead().Title()
	if ep.SeriesEditionHead().Slug() != "" {
		backLabel += " — " + ep.SeriesEditionHead().Label()
	}
	return ScrollY(Style("padding:0.5rem"))(
		FlexRow()(
			Button(Href(model.SeriesEditionEditorPath(ep.SeriesHead(), ep.SeriesEditionHead())), ButtonGhost)(
				Label("line/chevron-left", backLabel),
			),
		),
		SettingsPage()(
			FlexCol(Gap8)(
				FlexCol(Gap4)(
					expr.IfElse(len(episodeEditions) > 1,
						func() html.Node {
							return FlexCol(Gap8)(
								SettingsGroup()(
									SettingsGroupHead(Class("v-season-group-head"))(
										SettingsItemLabel()(
											SettingsItemLabelTitle(fmt.Sprintf("%d Editions", len(episodeEditions))),
											SettingsItemLabelDescription("Editing this episode affects these editions."),
										),
									),
									html.Range(episodeEditions, func(eped *model.Episode) html.Node {
										return SettingsItem()(
											SettingsItemLabel()(
												TextNode(Size2)(
													LiveText(eped.SeriesEditionHead().LabelField()),
													html.Text(" "),
													html.Text(eped.SnnEnn()),
												),
											),
											AppEpisodeEditionButton(eped.SeasonHead().ID(), eped.ID(), true, 0),
										)
									}),
								),
								SettingsContent()(
									TextNode(Size5)(LiveText((ep.TitleField()))),
								),
							)
						},
						func() html.Node {
							return SettingsContent()(
								TextNode(Size3)(
									LiveText(ep.SeriesEditionHead().LabelField()),
									html.Text(" "),
									html.Text(ep.SnnEnn()),
								),
								TextNode(Size5)(LiveText((ep.TitleField()))),
							)
						},
					),

					SettingsGroup()(
						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Title"),
							),
							SettingsTextField("/-/do/episode-set-title", "title", ep.Title(), LiveAddr(ep.TitleAddr()))(
								Hidden("id", ep.ID()),
							),
						),
						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Airdate"),
							),
							SettingsTextField("/-/do/episode-set-airdate", "airdate", ep.Airdate(), LiveAddr(ep.AirdateAddr()))(
								Hidden("id", ep.ID()),
							),
						),
						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Thumbnail"),
							),
							buttonImageEdit(
								"/-/dialog/episode-thumbnail/"+ep.ID(),
								ep.Thumbnail(), ep.ThumbnailAddr(),
								AspectThumbnail,
							),
						),
						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Type"),
							),
							SettingsButtonRow("/-/do/episode-set-type", "type", ep.Type(), LiveAddr(ep.TypeAddr()))(
								Hidden("id", ep.ID()),
								SettingsButtonRowItem("regular", Text("Regular")),
								SettingsButtonRowItem("significant_special", Text("Special")),
								SettingsButtonRowItem("insignificant_special", Text("Insignificant")),
							),
						),
					),
				),

				FlexCol(Gap2)(
					SettingsContent()(Text("Summary", Size2)),
					SettingsTextArea("/-/do/episode-set-summary", "summary", ep.Summary(), LiveAddr(ep.SummaryAddr()))(
						Hidden("id", ep.ID()),
					),
				),

				TextNode(TextBold, Style("margin-top: 1rem"))(html.Text("Videos")),
				expr.IfElse(len(videos) == 0,
					func() html.Node {
						return html.Div(
							Class("v-media-muted"),
						)(html.Text("No videos found"))
					},
					func() html.Node { return Group() },
				),
				html.Range(videos, func(v *model.Video) html.Node {
					return appEpisodeDialogVideo(ep, v)
				}),

				TextNode(TextBold, Style("margin-top: 1rem"))(html.Text("Renditions for Streaming")),
				expr.IfElse(len(renditions) == 0,
					func() html.Node {
						return html.Div(
							Class("v-media-muted"),
						)(html.Text("No renditions found"))
					},
					func() html.Node { return Group() },
				),
				html.Range(renditions, func(r schema.Rendition) html.Node {
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
func AppEpisodeEditionButton(seasonID, episodeID string, inEdition bool, undoSortKey int64) html.Node {
	return turbo.StreamTarget(appEpisodeEditionButtonID(seasonID, episodeID))(
		expr.IfElse(inEdition,
			func() html.Node {
				return ActionButton(
					path.Join("/-/do/season-remove-episode/", seasonID, episodeID),
					nil, Destructive, SettingsHover,
				)(
					Text("Remove"),
				)
			},
			func() html.Node {
				return FlexRow(Gap4, Style("align-items:center"))(
					ActionButton(
						path.Join("/-/do/season-add-episode/", seasonID, episodeID, strconv.FormatInt(undoSortKey, 10)),
						nil, ButtonGhost,
					)(
						Text("Undo"),
					),
					Text("Removed"),
				)
			},
		),
	)
}

func appEpisodeEditionButtonID(seasonID, episodeID string) string {
	return "episode-edition-button-" + seasonID + "-" + episodeID
}

func EpisodeEditionButtonUpdate(seasonID, episodeID string, inEdition bool, undoSortKey int64) html.Node {
	return turbo.Replace(appEpisodeEditionButtonID(seasonID, episodeID))(
		AppEpisodeEditionButton(seasonID, episodeID, inEdition, undoSortKey),
	)
}

func appEpisodeDialogVideo(ep *model.Episode, v *model.Video) html.Node {
	return html.Div(
		Class("v-media-indent"),
	)(
		html.Div()(
			html.Text("ID: "),
			html.Text(v.ID()),
		),
		html.Div()(
			html.Text("Name: "),
			html.Text(v.Name()),
		),
		html.Div()(
			html.Text("Original Key: "),
			html.Text(v.OriginalKey()),
		),
		html.Div()(
			html.Text("Playable: "),
			html.Text(strconv.FormatBool(v.Playable())),
		),
		FlexRow(Gap2, Style("margin-top: 0.5rem"))(
			activeVideoControl(v, "/-/do/episode-video-set-active/"+ep.ID()+"/"+v.ID()),
			ActionButton("/-/do/video-reimport/"+v.ID(), nil, Destructive)(
				html.Text("Re-import"),
			),
			expr.IfElse(v.OriginalKey() != "",
				func() html.Node {
					return ActionButton("/-/do/video-reencode/"+v.ID(), nil, Destructive)(
						html.Text("Re-encode"),
					)
				},
				func() html.Node { return Group() },
			),
			trashForm(v.ID()),
		),
	)
}

// activeVideoControl renders either an "Active" badge (when v is the
// active video) or a "Set active" button (when v is playable and not
// active). Unplayable videos render nothing — they cannot be made
// active.
func activeVideoControl(v *model.Video, setActivePath string) html.Node {
	switch {
	case v.Active():
		return TextNode(TextBold)(html.Text("Active"))
	case v.Playable():
		return ActionButton(setActivePath, nil)(html.Text("Set active"))
	default:
		return Group()
	}
}

func appEpisodeDialogRendition(r schema.Rendition) html.Node {
	return html.Div(
		Class("v-media-indent"),
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
		expr.IfElse(r.MaxHeight != 0,
			func() html.Node {
				return html.Div()(
					html.Textf("Max Height: %d", r.MaxHeight),
				)
			},
			func() html.Node { return Group() },
		),
		expr.IfElse(r.MaxFPS != 0,
			func() html.Node {
				return html.Div()(
					html.Textf("Max FPS: %d", r.MaxFPS),
				)
			},
			func() html.Node { return Group() },
		),
		expr.IfElse(r.Key != "",
			func() html.Node {
				return html.Div()(
					html.Text("Key: "),
					html.Text(r.Key),
				)
			},
			func() html.Node { return Group() },
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
					FlexRow(Gap4, Style("height: 100%"))(
						Inset(InsetSideLeft, Class("v-media-search-poster"))(
							PosterImg(AspectPoster, Style("height: 100%"), attr.Src(t.TVmaze.Image.Medium())),
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
		)(
			Text("Edit"),
		),
	)
}

func seriesTheaterPathText(sr *model.SeriesHead, sed *model.SeriesEditionHead) html.Node {
	if sed.Slug() == "" {
		return Group(html.Text("/"), LiveText(sr.SlugField()))
	}
	return Group(
		html.Text("/"), LiveText(sr.SlugField()),
		html.Text("/"), LiveText(sed.SlugField()),
	)
}

func SeasonAppend(sn *model.Season) html.Node {
	return turbo.Append("edition-seasons-"+sn.EditionID(),
		appSeriesDetailSeasonItem(sn),
	)
}

func SeasonEpisodesUpdate(sn *model.Season) html.Node {
	return turbo.Update("season-episodes-"+sn.ID(), turbo.Morph)(
		html.RangeSeq(sn.Episodes(model.AnyEpisode), appSeriesDetailEpisodeListItem),
	)
}

func SeriesSetSlug(sr *model.SeriesHead, oldSlug string, editions []*model.SeriesWork) html.Node {
	nodes := []html.Node{
		LiveTextUpdate(sr.SlugField()),
		turbo.SetTargets("[data-list-id-param=\""+sr.ID()+"\"]",
			html.Div(ListURL(sr.EditorPath()))(),
		),
	}
	for _, ed := range editions {
		edSlug := ed.SeriesEditionHead.Slug()
		oldTheaterPath := path.Join("/", oldSlug, edSlug)
		oldEditorPath := path.Join("/app/series", oldSlug, edSlug)
		nodes = append(nodes,
			turbo.URLReplace(oldTheaterPath, ed.TheaterPath()),
			turbo.URLReplace(oldEditorPath, ed.EditorPath()),
		)
	}
	return Group(nodes...)
}

func SeriesEditionSetSlug(ed *model.SeriesWork, oldSlug string) html.Node {
	oldTheaterPath := path.Join(ed.SeriesHead.TheaterPath(), oldSlug)
	oldEditorPath := path.Join(ed.SeriesHead.EditorPath(), oldSlug)
	return Group(
		LiveTextUpdate(ed.SeriesEditionHead.SlugField()),
		turbo.URLReplace(oldTheaterPath, ed.TheaterPath()),
		turbo.URLReplace(oldEditorPath, ed.EditorPath()),
	)
}

func SeriesEditionChangePoster(sed *model.SeriesEditionHead) html.Node {
	return liveImgUpdate(sed.PosterField())
}

func AppEpisodeThumbnailDialog(ep *model.EpisodeHead) html.Node {
	return ImageDialogStream(AspectThumbnail)(
		buttonUpload()(
			Hidden("ep-id", ep.ID()),
			PosterImg(AspectThumbnail, PosterFill, imgAttrs(ep.ThumbnailField())),
		),
	)
}

func EpisodeChangeThumbnail(ep *model.EpisodeHead) html.Node {
	return liveImgUpdate(ep.ThumbnailField())
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
