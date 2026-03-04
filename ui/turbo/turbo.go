package turbo

import (
	"bytes"
	"fmt"
	"io"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	frame  = html.Tag("turbo-frame")
	stream = html.Tag("turbo-stream")

	streamSource = html.Tag("turbo-stream-source")
)

var (
	action  = attr.Attr("action")
	target  = attr.Attr("target")
	targets = attr.Attr("targets")
)

var (
	DataAction = attr.Attr("data-turbo-action")
	DataFrame  = attr.Attr("data-turbo-frame")
)

var (
	Morph = attr.Attr("method")("morph")
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

func StreamSource(u string) html.Node {
	return streamSource(attr.Src(u))
}

// EncodeSSE encodes node as an SSE message event.
func EncodeSSE(w io.Writer, node html.Node) {
	var buf bytes.Buffer
	html.Render(&buf, node)
	data := buf.Bytes()
	// The turbo streams JS doesn't check for event types other than "message",
	// and it doesn't inspect the ID field. No point in adding those fields.
	for line := range bytes.SplitSeq(data, []byte("\n")) {
		fmt.Fprintf(w, "data: %s\n", line)
	}
	io.WriteString(w, "\n")
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

func AppendTargets(selector string, node ...html.Node) html.Node {
	return stream(action("append"), targets(selector))(
		html.Template()(node...),
	)
}

func ReplaceTargets(selector string, attrs ...attr.Node) html.Element {
	return func(node ...html.Node) html.Node {
		return stream(action("replace"), targets(selector), attr.Group(attrs...))(
			html.Template()(node...),
		)
	}
}

func UpdateTargets(selector string, node ...html.Node) html.Node {
	return stream(action("update"), targets(selector))(
		html.Template()(node...),
	)
}

func RemoveTargets(selector string) html.Node {
	return stream(action("remove"), targets(selector))
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
