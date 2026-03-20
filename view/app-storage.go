package view

import (
	"fmt"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

type Filesystem struct {
	Type string
	Path []string
	Size int64
	Used int64
	Free int64
}

func AppStorage(
	storage []*model.Storage,
	fs []*Filesystem,
) html.Node {
	return app("Storage",
		html.Div(attr.Class("v-system"))(
			html.H2()(html.Text("storage locations")),
			html.Form(
				attr.Attr("x-target")("dirs"),
				attr.Method("post"),
				attr.Action("/-/do/add-directory"),
				attr.Style("padding: 1em; border: 1px solid #00000022"),
			)(
				html.H3()(html.Text("add directory")),
				html.Div()(
					html.Label(
						attr.Style("display: block; font-weight: bold"),
					)(
						html.Text("contents"),
					),
					html.Div()(
						html.Input(
							attr.Type("radio"),
							attr.ID("contents-movie"),
							attr.Name("contents"),
							attr.Value("movie"),
						),
						html.Label(attr.For("contents-movie"))(html.Text("movies")),
					),
					html.Div()(
						html.Input(
							attr.Type("radio"),
							attr.ID("contents-series"),
							attr.Name("contents"),
							attr.Value("series"),
						),
						html.Label(attr.For("contents-series"))(html.Text("series")),
					),
					html.Label(
						attr.For("path"),
						attr.Style("display: block; font-weight: bold"),
					)(
						html.Text("filesystem path"),
					),
					InputText(
						attr.Class("v-system-input"),
						attr.ID("path"),
						attr.Name("path"),
					),
					InputSubmit(attr.Value("add")),
				),
			),
			html.Div(attr.ID("dirs"))(
				html.Range(storage, storageItem),
				html.If(len(storage) == 0, func() html.Node {
					return html.P()(html.I()(html.Text("no directories defined")))
				}),
			),
			html.H2()(html.Text("filesystem info")),
			html.Ul()(
				html.Range(fs, fsItem),
			),
		),
	)
}

func storageItem(it *model.Storage) html.Node {
	return html.Div()(
		html.Text(it.Contents()),
		html.Text(" "),
		html.Text(it.Path()),
	)
}

func fsItem(fs *Filesystem) html.Node {
	s := fmt.Sprintf("%s %d %d avail:%d %s", fs.Path, fs.Used, fs.Size, fs.Free, fs.Type)
	return html.Li()(
		html.Text(s),
	)
}
