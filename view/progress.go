package view

import (
	"strings"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model/progress"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
)

func progressItemClass(pi *progress.Item) string {
	return "progress-item-" + pi.Key()
}

func progressContainerClass(key string) string {
	return "progress-container-" + key
}

func ProgressItemAppend(pi *progress.Item) html.Node {
	var a []string
	for k := range pi.Parents() {
		a = append(a, "."+progressContainerClass(k))
	}
	return turbo.AppendTargets(strings.Join(a, ","),
		ProgressItem(pi),
	)
}

func ProgressItemUpdate(pi *progress.Item) html.Node {
	return turbo.ReplaceTargets("."+progressItemClass(pi), turbo.Morph)(
		ProgressItem(pi),
	)
}

func ProgressItemRemove(pi *progress.Item) html.Node {
	return turbo.RemoveTargets("." + progressItemClass(pi))
}

func ProgressContainer(key string, items []*progress.Item) html.Node {
	return html.Div(Contents, Class(progressContainerClass(key)))(
		html.Range(items, ProgressItem),
	)
}

// ProgressItem renders a single progress item with
// description, status, ETA, and a progress bar.
func ProgressItem(pi *progress.Item) html.Node {
	if err := pi.Error(); err != nil {
		return FlexCol(
			Class(progressItemClass(pi)),
			Class("text-sm text-red-11"),
		)(
			Text(pi.Description()),
			html.Div(Class("text-red-11/60"))(
				Text(truncate(err.Error(), 60)),
			),
		)
	}
	var etaNode html.Node
	if eta := pi.ETA(); eta > 0 {
		etaNode = html.Div(Class("text-gray-11/50"))(
			Text(formatDuration(eta)),
		)
	}
	return FlexCol(
		Class(progressItemClass(pi)),
		Class("text-gray-11/80 text-sm"),
	)(
		FlexRow(Class("gap-2"))(
			Text(pi.Description()),
			html.Div(Class("text-gray-11/50"))(
				Text(pi.Status()),
			),
			etaNode,
		),
		Progress(pi.Progress(), attr.Class("max-w-xs"), ProgressSM),
	)
}
