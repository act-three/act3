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

func EditMediaDownloads(
	title string,
	items []*model.DownloadHead,
	selected *model.Download,
) html.Node {
	const torrentListID = "torrent-list"
	return app(title,
		FlexCol(Class("place-self-stretch"))(
			ToolbarPrimary()(
				Box(),
				Box(Class("relative w-md"))(
					editMediaDownloadsSearchBar(),
				),
				Box(),
			),
			Split()(
				List("/app/downloads/", "detail",
					attr.ID(torrentListID),
					Class("flex-1"),
				)(
					ListItems(items, editMediaDownloadsListItem),
				),
				expr.IfElse(selected != nil,
					func() html.Node {
						return editMediaDownloadsDetail(selected)
					},
					func() html.Node {
						return Center(Class("text-gray-11/50"))(
							html.Text("No Download Selected"),
						)
					},
				),
			),
		),
	)
}

func EditMediaDownloadsDetailFrame(title string, dl *model.Download) html.Node {
	return PageFrame(title, "detail", editMediaDownloadsDetail(dl))
}

func EditMediaDownloadsStream(dls []*model.DownloadHead, edID string) html.Node {
	return turbo.Prepend("edition-torrents-"+edID,
		html.Range(dls, editMediaDownloadsStreamItem),
	)
}

func editMediaDownloadsSearchBar() html.Node {
	return Text("Download Searchbar")
}

func editMediaDownloadsListItem(dl *model.DownloadHead, attrs ...attr.Node) html.Node {
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
						CardDescription(attr.Class("line-clamp-2"))(
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

func editMediaDownloadsStreamItem(dl *model.DownloadHead) html.Node {
	return DownloadListItem(dl)
}

// DownloadListItem renders a single download as a clickable link.
// Shared by the series, movie, and download views.
func DownloadListItem(dl *model.DownloadHead) html.Node {
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

// AddTorrentButton renders a file-upload form for adding a
// torrent to an edition.
// Shared by the series and movie edit views.
func AddTorrentButton(inputName, inputValue string) html.Node {
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
			attr.Name(inputName),
			attr.Value(inputValue),
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

func editMediaDownloadsDetail(dl *model.Download) html.Node {
	if dl.State() == "error" {
		return Box()(
			Text(dl.Title()),
			Text(dl.Error()),
		)
	}
	return ScrollY()(
		html.Div(
			attr.Class("p-2"),
		)(
			html.H1(attr.Class("mb-2"))(Text(dl.Title())),
			html.Div()(
				editMediaDownloadsImportControl(dl),
			),
			FlexCol(Gap2)(
				html.RangeSeq2(
					xslices.GroupBy(dl.Files(), (*model.DownloadFile).Season),
					editMediaDownloadsFileGroup,
				),
			),
		),
	)
}

func editMediaDownloadsImportControl(dl *model.Download) html.Node {
	switch dl.State() {
	case "downloaded":
		return editMediaDownloadsImportButton(dl.ID())
	case "queued", "downloading":
		return editMediaDownloadsAutoImportCheckbox(dl)
	default: // "imported", "error"
		return html.Group()
	}
}

func editMediaDownloadsImportButton(id string) html.Node {
	return html.Form(
		attr.Method("POST"),
		attr.Action("/-/do/import-download"),
	)(
		html.Input(attr.Type("hidden"), attr.Name("id"), attr.Value(id)),
		Button()(html.Text("Import")),
	)
}

func editMediaDownloadsAutoImportCheckbox(dl *model.Download) html.Node {
	checkboxAttrs := []attr.Node{
		attr.Type("checkbox"),
		attr.Name("auto-import"),
		attr.Value("1"),
		attr.Attr("onchange")("this.form.requestSubmit()"),
	}
	if dl.AutoImport() {
		checkboxAttrs = append(checkboxAttrs, attr.Checked)
	}
	return html.Form(
		attr.Method("POST"),
		attr.Action("/-/do/auto-import-download"),
	)(
		html.Input(attr.Type("hidden"), attr.Name("id"), attr.Value(dl.ID())),
		html.Label(attr.Class("flex items-center gap-2"))(
			html.Input(checkboxAttrs...),
			html.Text("Automatically import when download completes"),
		),
	)
}

func editMediaDownloadsFileGroup(sn *model.Season, dfs []*model.DownloadFile) html.Node {
	displayDir, prefix := downloadsDirPrefix(dfs)
	return html.Div(
		attr.Class("border rounded-sm"),
	)(
		expr.IfElse(sn != nil,
			func() html.Node {
				// TODO(april): handle movies
				return html.Div(
					attr.Class("bg-accent-9 p-2 sticky top-0"),
				)(
					html.B()(html.Text(sn.Series().Title() + " — " + sn.Name())),
				)
			},
			func() html.Node {
				return html.Group()
			},
		),
		html.Div(
			attr.Class("p-2"),
		)(
			html.Text(displayDir),
		),
		html.Div(
			attr.Class("p-2"),
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
								return Progress(df.Progress(), Class("mt-1"), ProgressSM)
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
