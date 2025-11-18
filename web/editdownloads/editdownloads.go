package editdownloads

import (
	"iter"
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/anacrolix/torrent/metainfo"
	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/web/app"
	"ily.dev/act3/web/editseries"
	"ily.dev/act3/web/item"
	"ily.dev/act3/web/list"
	"ily.dev/act3/web/toolbar"
	"ily.dev/act3/web/turbo"
	"ily.dev/act3/web/view"
	"ily.dev/act3/web/web"
	"ily.dev/act3/xslices"
	"ily.dev/act3/xstrings"
)

var (
	div   = html.Div
	text  = html.Text
	textf = html.Textf
	class = attr.Class
)

const (
	torrentListID = "torrent-list"
)

func Editor(
	title string,
	items []*model.DownloadHead,
	selected *model.Download,
) http.Handler {
	return app.Page(title,
		FlexCol(class("place-self-stretch"))(
			toolbar.Primary()(
				div(),
				div(class("relative w-md"))(
					searchBar(),
				),
				div(),
			),
			view.Split()(
				list.List("/edit/downloads/", "detail",
					attr.ID(torrentListID),
					class("flex-1"),
				)(
					list.Items(items, listItem),
				),
				expr.IfElse(selected != nil,
					func() html.Node {
						return detail(selected)
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

func Stream(dls []*model.DownloadHead, edID string) http.Handler {
	return web.Stream(
		turbo.Prepend("edition-torrents-"+edID,
			editseries.ListDownloadDetail(dls),
		),
	)
}

func DetailFrame(title string, dl *model.Download) http.Handler {
	return app.PageFrame(title, "detail", detail(dl))
}

func detail(dl *model.Download) html.Node {
	if dl.State() == "error" {
		return html.Div()(
			div()(text(dl.Title())),
			div()(text(dl.Error())),
		)
	}
	return ScrollArea()(
		html.Div(
			attr.Class("p-2"),
		)(
			html.H1(attr.Class("mb-2"))(text(dl.Title())),
			html.Div()(
				doImportButton(dl.ID()),
			),
			html.Div(
				attr.Class("flex flex-col gap-2"),
			)(
				html.RangeSeq2(
					xslices.GroupBy(dl.Files(), (*model.DownloadFile).Season),
					fileGroup,
				),
			),
		),
	)
}

func doImportButton(id string) html.Node {
	return html.Form(
		attr.Method("POST"),
		attr.Action("/do/import-download"),
	)(
		html.Input(attr.Type("hidden"), attr.Name("id"), attr.Value(id)),
		Button()(html.Text("Import")),
	)
}

func filesInSameGroup(a, b metainfo.FileInfo, dl *model.Download) bool {
	aep := dl.PlanEpisode(path.Join(a.BestPath()...))
	bep := dl.PlanEpisode(path.Join(b.BestPath()...))
	if aep == nil && bep == nil {
		return true
	}
	return aep != nil && bep != nil && aep.SeasonHead().ID() == bep.SeasonHead().ID()
}

func hasSeason(fi metainfo.FileInfo, dl *model.Download) bool {
	id, _ := dl.PlanFor(path.Join(fi.BestPath()...))
	return id != ""
}

func dirPrefix(dfs []*model.DownloadFile) (dir, prefix string) {
	lcp := xstrings.LongestCommonPrefix(expr.Range(slices.Values(dfs),
		(*model.DownloadFile).Path,
	))
	dir, _, found := xstrings.LastCut(lcp, "/")
	if !found {
		return "", ""
	}
	return dir, dir + "/"
}

func fileGroup(sn *model.Season, dfs []*model.DownloadFile) html.Node {
	displayDir, prefix := dirPrefix(dfs)
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
				return item.Item()(
					item.Content()(
						expr.IfElse(ep != nil,
							func() html.Node {
								return item.Title()(
									html.Textf("%s. %s", ep.Label(), ep.Title()),
								)
							},
							func() html.Node {
								return html.Group()
							},
						),
						item.Description()(html.Text(displayPath)),
						expr.IfElse(df.Progress() >= 0,
							func() html.Node {
								return Progress(df.Progress(), class("mt-1")).With(ProgressSM)
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

func longestCommonPrefix(it iter.Seq[string]) string {
	s := slices.Collect(it)
	if len(s) == 0 {
		return ""
	}
	min := slices.Min(s)
	max := slices.Max(s)
	for i := range min {
		if i >= len(max) || min[i] != max[i] {
			return min[:i]
		}
	}
	return min
}

func listItem(dl *model.DownloadHead, attrs ...attr.Node) html.Node {
	return item.Item(
		attr.Group(attrs...),
		list.ID(dl.ID()),
		list.URL(dl.URL()),
	)(
		item.Content()(
			expr.IfElse(dl.State() == "error",
				func() html.Node {
					return html.Group(
						item.Title()(text(dl.Title())),
						item.Description(attr.Class("line-clamp-2"))(
							text(dl.Error()),
						),
					)
				},
				func() html.Node {
					return html.Group(
						item.Title(class("font-normal"))(text(dl.Title())),
						item.Description(attr.Class("line-clamp-2"))(
							textf("%d/%d assigned",
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

func searchBar() html.Node {
	return text("Download Searchbar")
}
