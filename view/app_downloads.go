package view

import (
	"cmp"
	"iter"
	"path"
	"slices"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/xslices"
)

func AppDownloads(
	title string,
	items []*model.DownloadInfo,
	selected *model.Download,
) (string, html.Node) {
	const torrentListID = "torrent-list"
	return title, FlexCol(Class("v-media-page"))(
		ToolbarPrimary()(
			Box(),
			Box(Class("v-media-searchbar"))(
				appDownloadsSearchBar(),
			),
			Box(),
		),
		Split()(
			List("/app/downloads/", "detail",
				attr.ID(torrentListID),
				Style("flex: 1"),
			)(
				ListItems(items, appDownloadsListItem),
			),
			turbo.Frame("detail", turbo.Advance())(
				expr.IfElse(selected != nil,
					func() html.Node {
						return appDownloadsDetail(selected)
					},
					func() html.Node {
						return Center(Class("v-media-muted"))(
							html.Text("No Download Selected"),
						)
					},
				),
			),
		),
	)
}

func AppDownloadsDetailFrame(title string, dl *model.Download) html.Node {
	return PageFrame(title, "detail", appDownloadsDetail(dl))
}

func AppDownloadsStream(dls []*model.DownloadHead, edID string) html.Node {
	return turbo.Prepend("edition-torrents-"+edID,
		html.Range(dls, appDownloadsStreamItem),
	)
}

func appDownloadsSearchBar() html.Node {
	return Text("Download Searchbar")
}

func appDownloadsListItem(dl *model.DownloadInfo, attrs ...attr.Node) html.Node {
	return Card(CardGhost,
		group(attrs...),
		ListID(dl.InfoHash()),
		ListURL(dl.EditorPath()),
	)(
		CardContent()(
			expr.IfElse(dl.State() == "error",
				func() html.Node {
					return Group(
						CardTitle()(Text(dl.Title())),
						CardDescription(LineClamp2)(
							Text(dl.Error()),
						),
					)
				},
				func() html.Node {
					return Group(
						appDownloadsWorkLabel(dl),
						CardTitle(TextNormal)(Text(dl.Title())),
						CardDescription(LineClamp2)(
							Textf("%d/%d assigned",
								dl.PlanLen(),
								dl.FilesLen(),
							),
						),
					)
				},
			),
		),
	)
}

func appDownloadsWorkLabel(dl *model.DownloadInfo) html.Node {
	return expr.IfElse(dl.MovieWork() != nil,
		func() html.Node {
			return Label(
				"line/film-01",
				dl.MovieWork().Title(),
				Size3, LineClamp1,
			)
		},
		func() html.Node {
			return Label(
				"line/tv-03",
				dl.SeriesWork().Title(),
				Size3, LineClamp1,
			)
		},
	)
}

func appDownloadsStreamItem(dl *model.DownloadHead) html.Node {
	return downloadListItem(dl)
}

// downloadListItem renders a single download as a clickable link.
// Shared by the series, movie, and download views.
func downloadListItem(dl *model.DownloadHead) html.Node {
	return SettingsItem()(
		SettingsItemLabel()(
			html.A(
				Href(dl.EditorPath()),
				turbo.DataFrame("main"),
			)(
				Text(dl.Title(), Size2),
			),
		),
	)
}

// addTorrentButton renders a file-upload form for adding a
// torrent to an edition.
// Shared by the series and movie edit views.
func addTorrentButton(inputName, inputValue string) html.Node {
	return html.Form(
		Class("v-media-torrent-form"),
		attr.Method("POST"),
		attr.Enctype("multipart/form-data"),
		attr.Action("/-/do/torrent-add"),
		stimulus.Controller("upload"),
		stimulus.Action("turbo:submit-end->upload#reset"),
	)(
		Hidden(inputName, inputValue),
		html.Input(
			Class("v-media-torrent-picker"),
			attr.Type("file"),
			attr.Name("torrent"),
			stimulus.Target("upload", "picker"),
			stimulus.Action("change->upload#upload"),
		),
		Button(
			ButtonGhost,
			stimulus.Target("upload", "button"),
			stimulus.Action("click->upload#open:prevent"),
		)(
			html.Text("Add Torrent"),
		),
	)
}

func appDownloadsDetail(dl *model.Download) html.Node {
	return FlexCol(Class("v-media-detail"))(
		ScrollY(Class("v-media-detail-body"))(
			SettingsPage()(
				FlexCol(Gap6)(
					SettingsContent()(
						FlexCol(Gap6)(
							FlexCol(Gap1)(
								TextNode(Size5)(html.Text(dl.Title())),
								expr.IfElse(dl.State() == "error",
									func() html.Node {
										return Label("line/alert-triangle", dl.Error(), Size3)
									},
									func() html.Node {
										return TextNode(Size3)(
											html.Textf("%d/%d assigned", dl.PlanLen(), dl.FilesLen()),
										)
									},
								),
							),
							Link(dl.Work().EditorPath(), Size4,
								turbo.DataFrame("_top"),
							)(
								Text(dl.Work().Title()),
							),
						),
					),

					html.If(dl.State() != "error", func() html.Node {
						return SettingsGroup()(
							appDownloadsImportControl(dl),
						)
					}),
				),

				appDownloadsFileList(dl.Files()),
			),
		),
	)
}

func appDownloadsImportControl(dl *model.Download) html.Node {
	switch dl.State() {
	case "downloaded":
		return SettingsItem()(
			SettingsItemLabel()(
				SettingsItemLabelTitle("Import"),
				SettingsItemLabelDescription("Import downloaded files into the library"),
			),
			html.Form(
				attr.Method("POST"),
				attr.Action("/-/do/download-import"),
			)(
				Hidden("id", dl.InfoHash()),
				Button(ButtonGhost, ButtonSize2)(html.Text("Import")),
			),
		)
	case "queued", "downloading":
		return SettingsItem()(
			SettingsItemLabel()(
				SettingsItemLabelTitle("Auto-Import"),
				SettingsItemLabelDescription("Automatically import when download completes"),
			),
			SettingsToggle("/-/do/download-auto-import", "auto-import", dl.AutoImport(),
				map[string]string{"id": dl.InfoHash()},
			),
		)
	default: // "imported", "error"
		return Group()
	}
}

func AppDownloadFileAttachPopover(
	triggerID string,
	sed *model.SeriesEdition,
	infoHash, filePath, currentVideoID string,
	linked map[string]bool,
) html.Node {
	var attachedEps []*model.Episode
	for ep := range sed.Episodes(model.AnyEpisode) {
		if linked[ep.ID()] {
			attachedEps = append(attachedEps, ep)
		}
	}
	return PopoverStream(triggerID,
		FlexCol(
			Style("width: 300px; height: 350px"),
			stimulus.Controller("picker"),
		)(
			InputText(
				attr.Autofocus,
				attr.Placeholder("Filter..."),
				Class("u-picker-filter"),
				stimulus.Action("input->picker#filter"),
			),
			ScrollY()(
				html.If(len(attachedEps) > 0, func() html.Node {
					return PickerGroup()(
						downloadAttachPickerEpisodes(slices.Values(attachedEps), infoHash, filePath, currentVideoID, linked),
					)
				}),
				html.RangeSeq(sed.Seasons(), func(sn *model.Season) html.Node {
					return PickerGroup()(
						PickerGroupHead()(
							PickerItemLabel()(Text(sn.Title(), Size2)),
						),
						downloadAttachPickerEpisodes(sn.Episodes(model.AnyEpisode), infoHash, filePath, currentVideoID, linked),
					)
				}),
			),
		),
	)
}

func downloadAttachPickerEpisodes(
	eps iter.Seq[*model.Episode],
	infoHash, filePath, currentVideoID string,
	linked map[string]bool,
) html.Node {
	return html.RangeSeq(eps, func(ep *model.Episode) html.Node {
		attached := linked[ep.ID()]
		label := ep.SnnEnn() + " " + ep.Title()
		imported := slices.ContainsFunc(ep.Videos(), func(v *model.Video) bool {
			return v.ID() != currentVideoID
		})
		return PickerItem(
			Style("isolation: isolate"),
			Attr("data-filter-text")(label),
			stimulus.Controller("episode-attach"),
			stimulus.Action("settings-toggle:commit->episode-attach#commit"),
		)(
			PickerItemLabel()(
				Text(label, Size2),
			),
			FlexRow(Gap2, Style("align-items:center"))(
				html.If(imported, func() html.Node {
					return Theme(Style("color:var(--text-3)"))(Icon("line/paperclip"))
				}),
				SettingsToggle("/-/do/episode-video-set", "attach", attached,
					map[string]string{
						"infohash":   infoHash,
						"path":       filePath,
						"episode-id": ep.ID(),
					},
					LiveAddr(model.EpisodeAttachToggleAddr(infoHash, filePath, ep.ID())),
					Style("position: relative; z-index: 1"),
				),
				html.Button(
					attr.Type("button"),
					Style("position: absolute; inset: 0; background: none; border: none; cursor: pointer"),
					stimulus.Action("click->episode-attach#attach"),
				),
			),
		)
	})
}

func appDownloadsFileList(files []*model.DownloadFile) html.Node {
	slices.SortStableFunc(files, func(a, b *model.DownloadFile) int {
		return cmp.Compare(path.Dir(a.Path()), path.Dir(b.Path()))
	})
	return html.RangeSeq2(
		xslices.GroupBy(files, func(df *model.DownloadFile) string {
			return path.Dir(df.Path())
		}),
		appDownloadsFileGroup,
	)
}

func appDownloadsFileGroup(dir string, dfs []*model.DownloadFile) html.Node {
	found := false
	for _, df := range dfs {
		found = found || df.HasVideoExtension()
	}
	if !found {
		return Group()
	}
	return SettingsGroup()(
		html.If(dir != ".", func() html.Node {
			return SettingsGroupHead()(
				SettingsItemLabel()(
					SettingsItemLabelTitle("/" + dir),
				),
			)
		}),
		html.Range(dfs, func(df *model.DownloadFile) html.Node {
			if !df.HasVideoExtension() {
				return Group()
			}
			return SettingsItem()(
				SettingsItemLabel()(
					SettingsItemLabelTitle(path.Base(df.Path())),
					downloadFileEpisodes(df),
					expr.IfElse(df.Progress() >= 0,
						func() html.Node {
							return Progress(df.Progress(), ProgressSM)
						},
						func() html.Node {
							return Group()
						},
					),
				),
				html.If(df.SeriesEdition() != nil, func() html.Node {
					// Disabled when the Video is gone — e.g. merged
					// away into a duplicate during ingest.
					return PopoverButton("/-/dialog/download-file-attach",
						Text("Attach"),
						Disabled(df.VideoID() == ""),
						SettingsHover, ButtonGhost, ButtonSize2,
					)(
						Hidden("infohash", df.InfoHash()),
						Hidden("path", df.Path()),
					)
				}),
			)
		}),
	)
}

func downloadFileEpisodes(df *model.DownloadFile) html.Node {
	return turbo.StreamTarget("dl-file-episodes-" + df.VideoID())(
		html.Range(df.Episodes(), func(ep *model.Episode) html.Node {
			return SettingsItemLabelDescription(
				ep.SnnEnn() + " " + ep.Title(),
			)
		}),
	)
}

func DownloadFileEpisodesUpdate(df *model.DownloadFile) html.Node {
	return turbo.Replace("dl-file-episodes-" + df.VideoID())(
		downloadFileEpisodes(df),
	)
}
