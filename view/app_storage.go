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

func AppStorage(fs []*Filesystem, cloneDegraded error) (title string, n domi.Node) {
	return "Storage", html.Div(attr.Class("v-system"))(
		cloneWarning(cloneDegraded),
		html.H2()(domi.Text("filesystem info")),
		html.UL()(
			rangeNodes(fs, fsItem),
		),
	)
}

// cloneWarning surfaces degraded clone traffic: bulk file copies
// have fallen back from filesystem clones to full byte copies,
// which keeps media flowing but wastes time and disk until the
// storage configuration is fixed.
func cloneWarning(err error) domi.Node {
	if err == nil {
		return nil
	}
	return html.P()(
		domi.Text("⚠ " + err.Error()),
	)
}

func fsItem(fs *Filesystem) domi.Node {
	s := fmt.Sprintf("%s %d %d avail:%d %s", fs.Path, fs.Used, fs.Size, fs.Free, fs.Type)
	return html.LI()(
		domi.Text(s),
	)
}
