package view

import (
	"fmt"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

type Filesystem struct {
	Type string
	Path []string
	Size int64
	Used int64
	Free int64
}

func AppStorage(fs []*Filesystem) (string, html.Node) {
	return "Storage", html.Div(attr.Class("v-system"))(
		html.H2()(html.Text("filesystem info")),
		html.Ul()(
			html.Range(fs, fsItem),
		),
	)
}

func fsItem(fs *Filesystem) html.Node {
	s := fmt.Sprintf("%s %d %d avail:%d %s", fs.Path, fs.Used, fs.Size, fs.Free, fs.Type)
	return html.Li()(
		html.Text(s),
	)
}
