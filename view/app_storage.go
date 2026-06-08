package view

import (
	"fmt"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"
)

type Filesystem struct {
	Type string
	Path []string
	Size int64
	Used int64
	Free int64
}

func AppStorage(fs []*Filesystem) (title string, n domi.Node) {
	return "Storage", html.Div(attr.Class("v-system"))(
		html.H2()(domi.Text("filesystem info")),
		html.UL()(
			rangeNodes(fs, fsItem),
		),
	)
}

func fsItem(fs *Filesystem) domi.Node {
	s := fmt.Sprintf("%s %d %d avail:%d %s", fs.Path, fs.Used, fs.Size, fs.Free, fs.Type)
	return html.LI()(
		domi.Text(s),
	)
}
