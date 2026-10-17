package view

import (
	"fmt"

	"ily.dev/act3/hi"
)

type Filesystem struct {
	Type string
	Path []string
	Size int64
	Used int64
	Free int64
}

func AppStorage(fs []*Filesystem, cloneDegraded error) hi.View {
	return hi.ScrollView(hi.Vertical,
		hi.VStack(
			hi.Text("Storage").Title("Storage"),
			cloneWarning(cloneDegraded),
			hi.VStack(
				hi.Text("filesystem info"),
				hi.VStack(
					hi.ForEach(fs, nil, fsItem),
				).
					Alignment(hi.Leading),
			).
				Alignment(hi.Leading).
				Gap(12i),
		).
			Alignment(hi.Leading).
			Gap(32i).
			Padding(hi.Edges(16i)),
	)
}

// cloneWarning surfaces degraded clone traffic: bulk file copies
// have fallen back from filesystem clones to full byte copies,
// which keeps media flowing but wastes time and disk until the
// storage configuration is fixed.
func cloneWarning(err error) hi.View {
	if err == nil {
		return hi.Empty()
	}
	return hi.Text("⚠ " + err.Error())
}

func fsItem(fs *Filesystem) hi.View {
	return hi.Text(fmt.Sprintf("%s %d %d avail:%d %s", fs.Path, fs.Used, fs.Size, fs.Free, fs.Type))
}
