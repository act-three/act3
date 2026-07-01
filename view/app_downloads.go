package view

import (
	"fmt"
	"hash/fnv"
	"io"
	"iter"
	"path"
	"slices"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/expr"
	"ily.dev/act3/model"
	"ily.dev/act3/model/kind"
	"ily.dev/act3/msg"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/xslices"
	"ily.dev/act3/xstrings"
)

const AppDownloadsListItems = "download-list-items"

func AppDownloads(
	items []*model.DownloadInfo,
	selected *model.Download,
	notFound bool,
) (title string, n domi.Node) {
	title = "Downloads"
	if selected != nil {
		title = selected.Title()
	}
	return title, FlexCol(Class("v-media-page"))(
		ToolbarPrimary()(
			Box(),
			Box(Class("v-media-searchbar"))(
				appDownloadsSearchBar(),
			),
			Box(),
		),
		Split()(
			List(Style("flex: 1"))(
				ListItems(items, func(dl *model.DownloadInfo) bool {
					return selected != nil && dl.InfoHash() == selected.InfoHash()
				}, appDownloadsListItem),
			),
			appDownloadSelection(selected, notFound),
		),
	)
}

func appDownloadSelection(selected *model.Download, notFound bool) domi.Node {
	switch {
	case selected != nil:
		return appDownloadsDetail(selected)
	case notFound:
		return Center(Class("v-media-muted"))(
			domi.Text("Not Found"),
		)
	}
	return Center(Class("v-media-muted"))(
		domi.Text("No Download Selected"),
	)
}

func appDownloadsSearchBar() domi.Node {
	return Text("Download Searchbar")
}

func appDownloadsListItem(dl *model.DownloadInfo, attrs ...domi.Attr) domi.Node {
	return CardLink(dl.EditorPath(), CardGhost,
		group(attrs...),
	)(
		CardContent()(
			expr.IfElse(dl.State() == "error",
				func() domi.Node {
					return Group(
						CardTitle()(Text(dl.Title())),
						CardDescription(LineClamp2)(
							Text(dl.Error()),
						),
					)
				},
				func() domi.Node {
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

func appDownloadsWorkLabel(dl *model.DownloadInfo) domi.Node {
	return expr.IfElse(dl.MovieWork() != nil,
		func() domi.Node {
			return Label(
				"line/film-01",
				dl.MovieWork().Title(),
				Size3, LineClamp1,
			)
		},
		func() domi.Node {
			return Label(
				"line/tv-03",
				dl.SeriesWork().Title(),
				Size3, LineClamp1,
			)
		},
	)
}

// downloadListItem renders a single download as a clickable link.
// Shared by the series, movie, and download views.
func downloadListItem(dl *model.DownloadHead) domi.Node {
	return SettingsItem()(
		SettingsItemLabel()(
			html.A(
				Href(dl.EditorPath()),
			)(
				Text(dl.Title(), Size2),
			),
		),
	)
}

// addTorrentButton renders a file-upload form for adding a
// torrent to an edition.
// Shared by the series and movie edit views.
func addTorrentButton(k kind.TorrentTarget, id string) domi.Node {
	return html.Form(
		Class("v-media-torrent-form"),
		attr.Method("POST"),
		attr.Enctype("multipart/form-data"),
		attr.Action("/-/do/torrent-add"),
		stimulus.Controller("upload"),
	)(
		Hidden("kind", k.String()),
		Hidden("id", id),
		html.Input(
			Class("v-media-torrent-picker"),
			attr.Type("file"),
			attr.Name("torrent"),
			stimulus.Target("upload", "picker"),
			stimulus.Action("change->upload#upload"),
		),
		Button(
			ButtonGhost,
			stimulus.Action("click->upload#open:prevent"),
		)(
			domi.Text("Add Torrent"),
		),
	)
}

// uploadVideoControl renders the video upload control for the
// target (an episode or movie edition, named by targetName's form
// field): an upload button, or the upload's progress while one is
// in flight for this target.
func uploadVideoControl(targetName, targetID string, uploads []model.Upload) domi.Node {
	for _, u := range uploads {
		if u.TargetID == targetID {
			return uploadProgressBar(u)
		}
	}
	return uploadVideoForm(targetName, targetID)
}

func uploadVideoForm(targetName, targetValue string) domi.Node {
	return html.Form(
		Class("v-media-torrent-form"),
		attr.Method("POST"),
		attr.Enctype("multipart/form-data"),
		attr.Action("/-/do/video-upload"),
		stimulus.Controller("upload"),
	)(
		Hidden(targetName, targetValue),
		html.Input(
			Class("v-media-torrent-picker"),
			attr.Type("file"),
			attr.Name("video"),
			attr.Accept(".mkv,.mp4,video/*"),
			stimulus.Target("upload", "picker"),
			stimulus.Action("change->upload#upload"),
		),
		Button(
			ButtonGhost,
			stimulus.Action("click->upload#open:prevent"),
		)(
			domi.Text("Upload Video"),
		),
	)
}

// uploadProgressBar renders an in-flight upload's name and progress.
func uploadProgressBar(u model.Upload) domi.Node {
	return FlexRow(Gap2, Style("align-items:center"))(
		Text(u.Name, Size2),
		Progress(u.Frac, Class("v-upload-progress")),
	)
}

func appDownloadsDetail(dl *model.Download) domi.Node {
	return FlexCol(Class("v-media-detail"))(
		ScrollY(Class("v-media-detail-body"))(
			SettingsPage()(
				FlexCol(Gap6)(
					SettingsContent()(
						FlexCol(Gap6)(
							FlexCol(Gap1)(
								TextNode(Size5)(domi.Text(dl.Title())),
								expr.IfElse(dl.State() == "error",
									func() domi.Node {
										return Label("line/alert-triangle", dl.Error(), Size3)
									},
									func() domi.Node {
										return TextNode(Size3)(
											domi.Textf("%d/%d assigned", dl.PlanLen(), dl.FilesLen()),
										)
									},
								),
							),
							Link(dl.Work().EditorPath(), Size4)(
								Text(dl.Work().Title()),
							),
						),
					),

					iff(dl.State() != "error", func() domi.Node {
						return SettingsGroup()(
							appDownloadsImportControl(dl),
						)
					}),
				),

				appDownloadsFileList(dl.Files()),

				SettingsGroup()(
					SettingsItem()(
						SettingsItemLabel()(
							SettingsItemLabelTitle("Delete"),
							SettingsItemLabelDescription("Deleted torrents remain in Transmission"),
						),
						trashForm(dl.InfoHash()),
					),
				),
			),
		),
	)
}

func appDownloadsImportControl(dl *model.Download) domi.Node {
	switch dl.State() {
	case "downloaded":
		return SettingsItem()(
			SettingsItemLabel()(
				SettingsItemLabelTitle("Import"),
				SettingsItemLabelDescription("Import downloaded files into the library"),
			),
			Button(onClick(&msg.DownloadImport{ID: dl.InfoHash()}),
				ButtonGhost, ButtonSize2,
			)(domi.Text("Import")),
		)
	case "queued", "downloading":
		return SettingsItem()(
			SettingsItemLabel()(
				SettingsItemLabelTitle("Auto-Import"),
				SettingsItemLabelDescription("Automatically import when download completes"),
			),
			settingsToggle(dl.AutoImport(), &msg.DownloadSetAutoImport{
				ID: dl.InfoHash(),
				On: !dl.AutoImport(),
			}),
		)
	default: // "imported", "error"
		return Group()
	}
}

// downloadFileAttachTriggerID derives a stable element ID for the
// file's Attach button, anchoring the popover to it.
func downloadFileAttachTriggerID(infoHash, path string) string {
	h := fnv.New32a()
	io.WriteString(h, infoHash)
	io.WriteString(h, "\x00")
	io.WriteString(h, path)
	return fmt.Sprintf("attach-%08x", h.Sum32())
}

// AppDownloadFileAttachPopover renders the episode picker for
// attaching a downloaded file, anchored to the file's Attach
// button: a client-side filter over the edition's episodes, each
// with an attach toggle. Clicking elsewhere in an unattached row
// attaches and closes the picker. The group at the top lists the
// episodes in attached — those attached when the picker opened —
// so the list keeps its shape as episodes are toggled.
func AppDownloadFileAttachPopover(
	sed *model.SeriesEdition,
	infoHash, filePath, currentVideoID string,
	attached []string,
	linked map[string]bool,
) domi.Node {
	pinned := map[string]bool{}
	for _, epID := range attached {
		pinned[epID] = true
	}
	var attachedEps []*model.Episode
	for ep := range sed.Episodes(model.AnyEpisode) {
		if pinned[ep.ID()] {
			attachedEps = append(attachedEps, ep)
		}
	}
	return popover(&msg.DialogClose{}, downloadFileAttachTriggerID(infoHash, filePath))(
		FlexCol(
			Style("width: 300px; height: 350px"),
			stimulus.Controller("picker"),
		)(
			InputText(
				attr.Autofocus(true),
				attr.Placeholder("Filter..."),
				Class("u-picker-filter"),
				stimulus.Action("input->picker#filter"),
			),
			ScrollY()(
				iff(len(attachedEps) > 0, func() domi.Node {
					return PickerGroup()(
						downloadAttachPickerEpisodes(slices.Values(attachedEps), infoHash, filePath, currentVideoID, linked),
					)
				}),

				rangeSeq(sed.Seasons(), func(sn *model.Season) domi.Node {
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
) domi.Node {
	return rangeSeq(eps, func(ep *model.Episode) domi.Node {
		attached := linked[ep.ID()]
		label := ep.SnnEnn() + " " + ep.Title()
		imported := slices.ContainsFunc(ep.Videos(), func(v *model.Video) bool {
			return v.ID() != currentVideoID
		})
		return PickerItem(
			Style("isolation: isolate"),
			Attr("data-filter-text")(label),
		)(
			PickerItemLabel()(
				Text(label, Size2),
			),
			FlexRow(Gap2, Style("align-items:center"))(
				iff(imported, func() domi.Node {
					return Theme(Style("color:var(--text-3)"))(Icon("line/paperclip"))
				}),

				settingsToggle(attached, &msg.EpisodeVideoSet{
					InfoHash:  infoHash,
					Path:      filePath,
					EpisodeID: ep.ID(),
					Attach:    !attached,
				}, Style("position: relative; z-index: 1")),
				// The rest of an unattached row is one big attach
				// target, which also closes the picker.
				iff(!attached, func() domi.Node {
					return html.Button(
						attr.Type("button"),
						Style("position: absolute; inset: 0; background: none; border: none; cursor: pointer"),
						onClick(&msg.DownloadFileAttachPick{
							InfoHash:  infoHash,
							Path:      filePath,
							EpisodeID: ep.ID(),
						}),
					)
				}),
			),
		)
	})

}

func appDownloadsFileList(files []*model.DownloadFile) domi.Node {
	slices.SortStableFunc(files, func(a, b *model.DownloadFile) int {
		return xstrings.CompareNatural(a.Path(), b.Path())
	})
	return rangeSeq2(
		xslices.GroupBy(files, func(df *model.DownloadFile) string {
			return path.Dir(df.Path())
		}),
		appDownloadsFileGroup)

}

func appDownloadsFileGroup(dir string, dfs []*model.DownloadFile) domi.Node {
	found := false
	for _, df := range dfs {
		found = found || df.HasVideoExtension()
	}
	if !found {
		return Group()
	}
	return SettingsGroup()(
		iff(dir != ".", func() domi.Node {
			return SettingsGroupHead()(
				SettingsItemLabel()(
					SettingsItemLabelTitle("/" + dir),
				),
			)
		}),

		rangeNodes(dfs, func(df *model.DownloadFile) domi.Node {
			if !df.HasVideoExtension() {
				return Group()
			}
			return SettingsItem()(
				SettingsItemLabel()(
					SettingsItemLabelTitle(path.Base(df.Path())),
					downloadFileEpisodes(df),
					expr.IfElse(df.Progress() >= 0,
						func() domi.Node {
							return Progress(df.Progress(), ProgressSM)
						},
						func() domi.Node {
							return Group()
						},
					),
				),
				iff(df.SeriesEdition() != nil, func() domi.Node {
					return Button(
						attr.ID(downloadFileAttachTriggerID(df.InfoHash(), df.Path())),
						onClick(&msg.DownloadFileAttachOpen{
							InfoHash: df.InfoHash(),
							Path:     df.Path(),
						}),
						Disabled(df.VideoID() == ""),
						SettingsHover, ButtonGhost, ButtonSize2,
					)(Text("Attach"))
				}),
			)
		}),
	)
}

func downloadFileEpisodes(df *model.DownloadFile) domi.Node {
	return rangeNodes(df.Episodes(), func(ep *model.Episode) domi.Node {
		return SettingsItemLabelDescription(
			ep.SnnEnn() + " " + ep.Title(),
		)
	})
}
