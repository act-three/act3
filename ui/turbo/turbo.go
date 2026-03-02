package turbo

import (
	"bytes"

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

func Append(id string, node ...html.Node) html.Node {
	return stream(action("append"), target(id))(
		html.Template()(node...),
	)
}

var streamTagMagic = func() []byte {
	var b bytes.Buffer
	html.Render(&b, stream())
	magic := b.Bytes()
	magic, _, _ = bytes.Cut(magic, []byte(">"))
	magic, _, _ = bytes.Cut(magic, []byte(" "))
	return append(magic, ' ')
}()

// SniffStream returns whether b begins with a <turbo-stream> tag.
func SniffStream(b []byte) bool {
	b = bytes.TrimSpace(b)
	return bytes.HasPrefix(b, streamTagMagic)
}
