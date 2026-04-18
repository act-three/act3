package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/ui/turbo"
)

const notePortController = "note-port"

// Variants
var (
	NoteInfo    = Attr("data-variant")("info") // default
	NoteSuccess = Attr("data-variant")("success")
	NoteWarning = Attr("data-variant")("warning")
)

// NotePort is the fixed-position container where notes
// appear. Place it once in the base app layout.
func NotePort(attrs ...attr.Node) html.Node {
	return html.Div(
		attr.ID("note-port"),
		Class("u-note-port"),
		attr.Role("region"),
		Attr("aria-label")("Notifications"),
		stimulus.Controller(notePortController),
		stimulus.Action("mouseenter->note-port#pause"),
		stimulus.Action("mouseleave->note-port#resume"),
		stimulus.Action("visibilitychange@document->note-port#togglePaused"),
		group(attrs...),
	)()
}

// Note renders a notification as a Turbo Streams action.
// Use NoteIcon, NoteTitle, NoteDescription, and
// NoteAction as children.
func Note(attrs ...attr.Node) html.Element {
	return func(nodes ...html.Node) html.Node {
		return turbo.Append("note-port",
			html.Div(
				Class("u-note"),
				attr.Role("status"),
				Attr("aria-live")("polite"),
				stimulus.Target(notePortController, "note"),
				stimulus.Action("pointerdown->note-port#swipeStart"),
				stimulus.Action("pointermove->note-port#swipeMove"),
				stimulus.Action("pointerup->note-port#swipeEnd"),
				group(attrs...),
			)(nodes...),
		)
	}
}

// NoteTitle renders the title line of a note.
func NoteTitle(attrs ...attr.Node) html.Element {
	return html.Div(
		Class("u-note-title"),
		group(attrs...),
	)
}

// NoteDescription renders the body text of a note.
func NoteDescription(attrs ...attr.Node) html.Element {
	return html.Div(
		Class("u-note-description"),
		group(attrs...),
	)
}

// NoteIcon renders the leading icon of a note. The icon
// height matches the text line-height for alignment.
func NoteIcon(children ...html.Node) html.Node {
	return html.Div(
		Class("u-note-icon"),
	)(children...)
}

// NoteAction renders an action button inside a note.
// It renders as <a> if an href attr is provided.
func NoteAction(attrs ...attr.Node) html.Element {
	a := group(attrs...)
	tag := "button"
	if a.Has("href") {
		tag = "a"
	}
	return html.Tag(tag)(
		Class("u-note-action"),
		a,
	)
}
