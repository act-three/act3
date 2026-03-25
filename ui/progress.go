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
		attr.Class("u-progress"),
		attr.Group(attrs...),
	)(
		html.Div(
			attr.Class("u-progress-fill"),
			attr.Stylef("width: %.2f%%", 100*value),
		),
	)
}

var (
	ProgressSM = attr.Attr("data-progress-size")("sm")
	ProgressMD = attr.Attr("data-progress-size")("md")
	ProgressLG = attr.Attr("data-progress-size")("lg")
)
