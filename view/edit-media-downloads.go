package view

import (
	"slices"
	"strings"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
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
				List("/edit/downloads/", "detail",
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
						return html.Div(
							attr.Class(`
								grid
								h-full
								w-full
								place-items-center
								text-gray-11/50
							`),
						)(html.Text("No Download Selected"))
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

func editMediaDownloadsDetail(dl *model.Download) html.Node {
	if dl.State() == "error" {
		return Box()(
			Text(dl.Title()),
			Text(dl.Error()),
		)
	}
	return ScrollArea()(
		html.Div(
			attr.Class("p-2"),
		)(
			html.H1(attr.Class("mb-2"))(Text(dl.Title())),
			html.Div()(
				editMediaDownloadsDoImportButton(dl.ID()),
			),
			html.Div(
				attr.Class("flex flex-col gap-2"),
			)(
				html.RangeSeq2(
					xslices.GroupBy(dl.Files(), (*model.DownloadFile).Season),
					editMediaDownloadsFileGroup,
				),
			),
		),
	)
}

func editMediaDownloadsDoImportButton(id string) html.Node {
	return html.Form(
		attr.Method("POST"),
		attr.Action("/do/import-download"),
	)(
		html.Input(attr.Type("hidden"), attr.Name("id"), attr.Value(id)),
		Button()(html.Text("Import")),
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
