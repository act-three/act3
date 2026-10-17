package view

import (
	"fmt"

	. "ily.dev/act3/hi"
)

type Filesystem struct {
	Type string
	Path []string
	Size int64
	Used int64
	Free int64
}

func AppStorage(fs []*Filesystem, cloneDegraded error) View {
	return ScrollView(Vertical,
		VStack(
			Text("Storage").Title("Storage"),
			cloneWarning(cloneDegraded),
			VStack(
				Text("filesystem info"),
				VStack(
					ForEach(fs, nil, fsItem),
				).
					Alignment(Leading),
			).
				Alignment(Leading).
				Gap(12i),
		).
			Alignment(Leading).
			Gap(32i).
			Padding(Edges(16i)),
	)
}

// cloneWarning surfaces degraded clone traffic: bulk file copies
// have fallen back from filesystem clones to full byte copies,
// which keeps media flowing but wastes time and disk until the
// storage configuration is fixed.
func cloneWarning(err error) View {
	if err == nil {
		return Empty()
	}
	return Text("⚠ " + err.Error())
}

func fsItem(fs *Filesystem) View {
	return Text(fmt.Sprintf("%s %d %d avail:%d %s", fs.Path, fs.Used, fs.Size, fs.Free, fs.Type))
}
