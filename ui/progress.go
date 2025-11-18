package ui

import (
	"fmt"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

// Progress displays a progress bar indicating the completion of a task.
// The value is clamped to the range [0, 1].
func Progress(value float64, attrs ...attr.Node) html.Node {
	value = min(max(value, 0), 1)
	state := "loading"
	if value >= 1 {
		state = "complete"
	}

	return html.Div(
		attr.Role("progressbar"),
		attr.Attr("aria-valuenow")(fmt.Sprintf("%.1f", value*100)),
		attr.Attr("data-state")(state),
		attr.Attr("data-value")(fmt.Sprintf("%.1f", value*100)),
		Class(`
			overflow-hidden
			relative
			w-full
			rounded-full
			bg-gray-3
		`),
		attr.EnvAttr("class", progressSizeKey, progressMD),
		attr.Group(attrs...),
	)(
		html.Div(
			Class(`
				h-full
				rounded-full
				bg-accent-9
				transition-[width]
				duration-600
				ease-linear
			`),
			attr.Style(fmt.Sprintf("width: %.2f%%", 100*value)),
		),
	)
}

var (
	ProgressSM = html.WithValue(progressSizeKey, progressSM)
	ProgressMD = html.WithValue(progressSizeKey, progressMD) // default
	ProgressLG = html.WithValue(progressSizeKey, progressLG)
)

const (
	progressSM = "h-1"
	progressMD = "h-2"
	progressLG = "h-3"
)
