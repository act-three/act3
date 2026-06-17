package ui

import (
	"fmt"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"
)

// Progress displays a progress bar indicating the completion of a task.
// The value is clamped to the range [0, 1].
func Progress(value float64, attrs ...domi.Attr) domi.Node {
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
