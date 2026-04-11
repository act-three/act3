package view

import (
	"cmp"
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
	items []*model.DownloadHead,
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
				attr.Style("flex: 1"),
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

func appDownloadsListItem(dl *model.DownloadHead, attrs ...attr.Node) html.Node {
	return Card(CardGhost,
		attr.Group(attrs...),
		ListID(dl.InfoHash()),
		ListURL(dl.EditorPath()),
	)(
		CardContent()(
			expr.IfElse(dl.State() == "error",
				func() html.Node {
					return html.Group(
						CardTitle()(Text(dl.Title())),
						CardDescription(LineClamp2)(
							Text(dl.Error()),
						),
					)
				},
				func() html.Node {
					return html.Group(
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

func appDownloadsStreamItem(dl *model.DownloadHead) html.Node {
	return downloadListItem(dl)
}

// downloadListItem renders a single download as a clickable link.
// Shared by the series, movie, and download views.
func downloadListItem(dl *model.DownloadHead) html.Node {
	return SettingsItem()(
		SettingsItemLabel()(
			html.A(
				attr.Href(dl.EditorPath()),
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
		attr.Class("v-media-torrent-form"),
		attr.Method("POST"),
		attr.Enctype("multipart/form-data"),
		attr.Action("/-/do/torrent-add"),
		stimulus.Controller("upload"),
		stimulus.Action("turbo:submit-end->upload#reset"),
	)(
		Hidden(inputName, inputValue),
		html.Input(
			attr.Class("v-media-torrent-picker"),
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
						TextNode(Size6)(html.Text(dl.Title())),
						expr.IfElse(dl.State() == "error",
							func() html.Node {
								return Label("line/alert-triangle", dl.Error(), Size2)
							},
							func() html.Node {
								return TextNode(Size2)(
									html.Textf("%d/%d assigned", dl.PlanLen(), dl.FilesLen()),
								)
							},
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
			SettingsToggle("/-/do/download-auto-import", "auto-import", dl.AutoImport())(
				Hidden("id", dl.InfoHash()),
			),
		)
	default: // "imported", "error"
		return html.Group()
	}
}

func AppDownloadFileAttachDialog(
	sed *model.SeriesEdition,
	infoHash, filePath string,
	linked map[string]bool,
) html.Node {
	return DialogStream(
		FlexCol(Gap2, Class("v-media-dialog"))(
			html.Div(Class("v-media-dialog-fixed"))(
				Text("Attach to Episode"),
			),
			ScrollY(Class("v-media-dialog-results"))(
				html.RangeSeq(sed.Seasons(), func(sn *model.Season) html.Node {
					return SettingsGroup()(
						SettingsGroupHead()(
							SettingsItemLabel()(
								SettingsItemLabelTitle(sn.Title()),
							),
						),
						html.RangeSeq(sn.Episodes(model.AnyEpisode), func(ep *model.Episode) html.Node {
							attached := linked[ep.ID()]
							return SettingsItem(
								attr.Style("isolation: isolate"),
								stimulus.Controller("episode-attach"),
								stimulus.Action("settings-toggle:commit->episode-attach#commit"),
							)(
								SettingsItemLabel()(
									SettingsItemLabelTitle(ep.SnnEnn()+" "+ep.Title()),
								),
								SettingsToggle("/-/do/episode-video-set", "attach", attached,
									attr.Style("position: relative; z-index: 1"),
								)(
									Hidden("infohash", infoHash),
									Hidden("path", filePath),
									Hidden("episode-id", ep.ID()),
								),
								html.Button(
									attr.Type("button"),
									attr.Style("position: absolute; inset: 0; background: none; border: none; cursor: pointer"),
									stimulus.Action("click->episode-attach#attach"),
								),
							)
						}),
					)
				}),
			),
		),
	)
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
					return html.Form(
						attr.Method("get"),
						attr.Action("/-/dialog/download-file-attach"),
					)(
						Hidden("infohash", df.InfoHash()),
						Hidden("path", df.Path()),
						Button(SettingsHover, ButtonGhost, ButtonSize2)(Text("Attach")),
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
