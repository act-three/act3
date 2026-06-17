package view

import (
	"ily.dev/domi"
	"ily.dev/domi/html"

	"ily.dev/act3/model/progress"
	. "ily.dev/act3/ui"
)

func progressItemClass(pi *progress.Item) string {
	return "progress-item-" + pi.Key()
}

func progressContainerClass(key string) string {
	return "progress-container-" + key
}

func progressContainer(key string, items []*progress.Item) domi.Node {
	return html.Div(Contents, Class(progressContainerClass(key)))(
		rangeNodes(items, progressItem),
	)
}

// progressItem renders a single progress item with
// description, status, ETA, and a progress bar.
func progressItem(pi *progress.Item) domi.Node {
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
	var etaNode domi.Node
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
		Progress(pi.Progress(), Class("v-progress-bar"), ProgressSM),
	)
}
