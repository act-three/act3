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
		progressItem(pi),
	)
}

func ProgressItemUpdate(pi *progress.Item) html.Node {
	return turbo.ReplaceTargets("."+progressItemClass(pi), turbo.Morph)(
		progressItem(pi),
	)
}

func ProgressItemRemove(pi *progress.Item) html.Node {
	return turbo.RemoveTargets("." + progressItemClass(pi))
}

func progressContainer(key string, items []*progress.Item) html.Node {
	return html.Div(Contents, Class(progressContainerClass(key)))(
		html.Range(items, progressItem),
	)
}

// progressItem renders a single progress item with
// description, status, ETA, and a progress bar.
func progressItem(pi *progress.Item) html.Node {
	if err := pi.Error(); err != nil {
		return FlexCol(
			Class(progressItemClass(pi)),
			Class("v-progress-item-error"),
		)(
			Text(pi.Description()),
			html.Div(Class("v-progress-item-error-detail"))(
				Text(truncate(err.Error(), 60)),
			),
		)
	}
	var etaNode html.Node
	if eta := pi.ETA(); eta > 0 {
		etaNode = html.Div(Class("v-progress-muted"))(
			Text(formatDuration(eta)),
		)
	}
	return FlexCol(
		Class(progressItemClass(pi)),
		Class("v-progress-item"),
	)(
		FlexRow(Gap2)(
			Text(pi.Description()),
			html.Div(Class("v-progress-muted"))(
				Text(pi.Status()),
			),
			etaNode,
		),
		Progress(pi.Progress(), attr.Class("v-progress-bar"), ProgressSM),
	)
}
