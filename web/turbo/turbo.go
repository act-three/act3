package turbo

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	action = attr.Attr("action")
	frame  = html.Tag("turbo-frame")
	stream = html.Tag("turbo-stream")
	target = attr.Attr("target")
)

var (
	Action      = attr.Attr("data-action")
	Controller  = attr.Attr("data-controller")
	TurboAction = attr.Attr("data-turbo-action")
	TurboFrame  = attr.Attr("data-turbo-frame")
)

func init() {
	attr.RegisterCombining("data-action")
}

func Target(controller, name string) attr.Node {
	return attr.Attr("data-" + controller + "-target")(name)
}

func Value(controller, name string) attr.AttrName {
	return attr.Attr("data-" + controller + "-" + name + "-value")
}

func Frame(id string, attrs ...attr.Node) html.Element {
	return frame(
		attr.ID(id),
		attr.Class("contents"),
		attr.Group(attrs...),
	)
}

func Sink(id string, attrs ...attr.Node) html.Element {
	return html.Div(
		attr.ID(id),
		attr.Class("contents"),
		attr.Group(attrs...),
	)
}

func Prepend(id string, node ...html.Node) html.Node {
	return stream(action("prepend"), target(id))(
		html.Template()(node...),
	)
}
