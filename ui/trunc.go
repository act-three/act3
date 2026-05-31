package ui

import "ily.dev/act3/html"

// truncTailRunes is how many trailing runes TruncTail keeps visible.
const truncTailRunes = 10

// TruncTail renders s on a single line that collapses from the front
// when horizontal space runs short, always keeping the final runes
// visible with an ellipsis standing in for the hidden start, and
// exposes the full text as a native hover tooltip. It suits menu and
// list labels whose distinguishing part — a codec, channel count, or
// part number — sits at the end, where a plain end-ellipsis would
// hide exactly the part that tells entries apart.
//
// Strings short enough to have nothing worth collapsing render as
// plain text, with no tooltip. The caller is responsible for placing
// the result in a width-constrained context (e.g. a flex item with a
// max-width); on its own TruncTail only describes how to give way.
func TruncTail(s string) html.Node {
	r := []rune(s)
	if len(r) <= truncTailRunes {
		return html.Text(s)
	}
	head := string(r[:len(r)-truncTailRunes])
	tail := string(r[len(r)-truncTailRunes:])
	return html.Span(Class("u-trunc"), Attr("title")(s))(
		html.Span(Class("u-trunc-head"))(html.Text(head)),
		html.Span(Class("u-trunc-tail"))(html.Text(tail)),
	)
}
