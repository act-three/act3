package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

// ScrollXY is a simple overflow container using native scrollbars.
// It scrolls along the X axis.
//
// Unlike the Radix ScrollArea component,
// which provides custom styled scrollbar tracks and thumbs,
// this relies on the browser's default scrollbar rendering.
func ScrollX(attrs ...domi.Attr) domi.Element {
	return html.Div(
		Class("u-scroll u-scroll+x"),
		group(attrs...),
	)
}

// ScrollXY is a simple overflow container using native scrollbars.
// It scrolls along the Y axis.
//
// Unlike the Radix ScrollArea component,
// which provides custom styled scrollbar tracks and thumbs,
// this relies on the browser's default scrollbar rendering.
func ScrollY(attrs ...domi.Attr) domi.Element {
	return html.Div(
		Class("u-scroll u-scroll+y"),
		group(attrs...),
	)
}

// ScrollXY is a simple overflow container using native scrollbars.
// It scrolls along both X and Y axes.
//
// Unlike the Radix ScrollArea component,
// which provides custom styled scrollbar tracks and thumbs,
// this relies on the browser's default scrollbar rendering.
func ScrollXY(attrs ...domi.Attr) domi.Element {
	return html.Div(
		Class("u-scroll u-scroll+xy"),
		group(attrs...),
	)
}
