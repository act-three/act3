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
		Attr("aria-valuenow")(fmt.Sprintf("%.1f", value*100)),
		Attr("data-state")(state),
		Attr("data-value")(fmt.Sprintf("%.1f", value*100)),
		Class("u-progress"),
		group(attrs...),
	)(
		html.Div(
			Class("u-progress-fill"),
			Stylef("width: %.2f%%", 100*value),
		),
	)
}

var (
	ProgressSM = Attr("data-progress-size")("sm")
	ProgressMD = Attr("data-progress-size")("md")
	ProgressLG = Attr("data-progress-size")("lg")
)
