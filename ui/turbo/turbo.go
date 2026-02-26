package turbo

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	frame  = html.Tag("turbo-frame")
	stream = html.Tag("turbo-stream")
)

var (
	action = attr.Attr("action")
	target = attr.Attr("target")
)

var (
	DataAction = attr.Attr("data-turbo-action")
	DataFrame  = attr.Attr("data-turbo-frame")
)

func Frame(id string, attrs ...attr.Node) html.Element {
	return frame(
		attr.ID(id),
		attr.Class("u-contents"),
		attr.Group(attrs...),
	)
}

func Sink(id string, attrs ...attr.Node) html.Element {
	return html.Div(
		attr.ID(id),
		attr.Class("u-contents"),
		attr.Group(attrs...),
	)
}

func Prepend(id string, node ...html.Node) html.Node {
	return stream(action("prepend"), target(id))(
		html.Template()(node...),
	)
}
