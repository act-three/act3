package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
)

const noteController = "note"

// Variants
var (
	NoteInfo    = Class("u-note+info")    // default
	NoteSuccess = Class("u-note+success")
	NoteError   = Class("u-note+error")
	NoteWarning = Class("u-note+warning")
)

// NoteViewport is the fixed-position container where notes
// appear. Place it once in the base app layout.
func NoteViewport(attrs ...attr.Node) html.Node {
	return html.Div(
		attr.ID("note-viewport"),
		attr.Class("u-note-viewport"),
		attr.Role("region"),
		attr.Attr("aria-label")("Notifications"),
		group(attrs...),
	)()
}

// Note renders a notification. Use NoteTitle, NoteDescription,
// NoteAction, and NoteClose as children.
func Note(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("u-note"),
		attr.Role("status"),
		attr.Attr("aria-live")("polite"),
		attr.Attr("data-state")("open"),
		stimulus.Controller(noteController),
		stimulus.Action("mouseenter->note#pause"),
		stimulus.Action("mouseleave->note#resume"),
		group(attrs...),
	)
}

// NoteTitle renders the title line of a note.
func NoteTitle(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("u-note-title"),
		group(attrs...),
	)
}

// NoteDescription renders the body text of a note.
func NoteDescription(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("u-note-description"),
		group(attrs...),
	)
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
		attr.Class("u-note-action"),
		a,
	)
}

// NoteClose renders a dismiss button for the note.
func NoteClose(attrs ...attr.Node) html.Node {
	return html.Tag("button")(
		attr.Class("u-note-close"),
		attr.Attr("aria-label")("Close"),
		stimulus.Action("click->note#dismiss"),
		group(attrs...),
	)(
		Icon("x"),
	)
}
