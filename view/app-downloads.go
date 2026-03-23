package view

import (
	"slices"
	"strings"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/xslices"
	"ily.dev/act3/xstrings"
)

func AppDownloads(
	title string,
	items []*model.DownloadHead,
	selected *model.Download,
) html.Node {
	const torrentListID = "torrent-list"
	return app(title,
		FlexCol(Class("v-media-page"))(
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
		ListID(dl.ID()),
		ListURL(dl.URL()),
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
						CardTitle(FontNormal)(Text(dl.Title())),
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
				attr.Href(dl.URL()),
				turbo.DataFrame("main"),
			)(
				Text(dl.Title(), TextSize2),
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
		stimulus.Controller("add-torrent"),
		stimulus.Action("turbo:submit-end->add-torrent#reset"),
	)(
		html.Input(
			attr.Type("hidden"),
			attr.Name(inputName),
			attr.Value(inputValue),
		),
		html.Input(
			attr.Class("v-media-torrent-picker"),
			attr.Type("file"),
			attr.Name("torrent"),
			stimulus.Target("add-torrent", "picker"),
			stimulus.Action("change->add-torrent#upload"),
		),
		Button(
			ButtonGhost,
			stimulus.Target("add-torrent", "button"),
			stimulus.Action("click->add-torrent#open:prevent"),
		)(
			html.Text("Add Torrent"),
		),
	)
}

func appDownloadsDetail(dl *model.Download) html.Node {
	if dl.State() == "error" {
		return Box()(
			Text(dl.Title()),
			Text(dl.Error()),
		)
	}
	return ScrollY()(
		html.Div(
			attr.Class("v-media-file-group-body"),
		)(
			html.H1(attr.Style("margin-bottom: 0.5rem"))(Text(dl.Title())),
			html.Div()(
				appDownloadsImportControl(dl),
			),
			FlexCol(Gap2)(
				html.RangeSeq2(
					xslices.GroupBy(dl.Files(), (*model.DownloadFile).Season),
					appDownloadsFileGroup,
				),
			),
		),
	)
}

func appDownloadsImportControl(dl *model.Download) html.Node {
	switch dl.State() {
	case "downloaded":
		return appDownloadsImportButton(dl.ID())
	case "queued", "downloading":
		return appDownloadsAutoImportToggle(dl)
	default: // "imported", "error"
		return html.Group()
	}
}

func appDownloadsImportButton(id string) html.Node {
	return html.Form(
		attr.Method("POST"),
		attr.Action("/-/do/download-import"),
	)(
		html.Input(attr.Type("hidden"), attr.Name("id"), attr.Value(id)),
		Button()(html.Text("Import")),
	)
}

func appDownloadsAutoImportToggle(dl *model.Download) html.Node {
	return html.Label(attr.Class("v-media-auto-import"))(
		SettingsToggle("/-/do/download-auto-import", "auto-import", dl.AutoImport())(
			html.Input(attr.Type("hidden"), attr.Name("id"), attr.Value(dl.ID())),
		),
		html.Text("Automatically import when download completes"),
	)
}

func appDownloadsFileGroup(sn *model.Season, dfs []*model.DownloadFile) html.Node {
	displayDir, prefix := downloadsDirPrefix(dfs)
	return html.Div(
		attr.Class("v-media-file-group"),
	)(
		expr.IfElse(sn != nil,
			func() html.Node {
				// TODO(april): handle movies
				return html.Div(
					attr.Class("v-media-file-group-header"),
				)(
					html.B()(html.Text(sn.Series().Title() + " — " + sn.Name())),
				)
			},
			func() html.Node {
				return html.Group()
			},
		),
		html.Div(
			attr.Class("v-media-file-group-body"),
		)(
			html.Text(displayDir),
		),
		html.Div(
			attr.Class("v-media-file-group-body"),
		)(
			html.Range(dfs, func(df *model.DownloadFile) html.Node {
				ep := df.Episode()
				displayPath := strings.TrimPrefix(df.Path(), prefix)
				return Card(CardGhost)(
					CardContent()(
						expr.IfElse(ep != nil,
							func() html.Node {
								return CardTitle()(
									html.Textf("%s. %s", ep.Label(), ep.Title()),
								)
							},
							func() html.Node {
								return html.Group()
							},
						),
						CardDescription()(html.Text(displayPath)),
						expr.IfElse(df.Progress() >= 0,
							func() html.Node {
								return Progress(df.Progress(), attr.Style("margin-top: 0.25rem"), ProgressSM)
							},
							func() html.Node {
								return html.Group()
							},
						),
					),
				)
			}),
		),
	)
}

func downloadsDirPrefix(dfs []*model.DownloadFile) (dir, prefix string) {
	lcp := xstrings.LongestCommonPrefix(expr.Range(slices.Values(dfs),
		(*model.DownloadFile).Path,
	))
	dir, _, found := xstrings.LastCut(lcp, "/")
	if !found {
		return "", ""
	}
	return dir, dir + "/"
}
