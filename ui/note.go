package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/ui/stimulus"
)

const notePortController = "note-port"

// A NoteVariant selects a note's visual treatment.
type NoteVariant string

const (
	NoteInfo    NoteVariant = ""
	NoteSuccess NoteVariant = "success"
	NoteWarning NoteVariant = "warning"
	NoteError   NoteVariant = "error"
)

// A Note is a transient notification queued for display.
type Note struct {
	ID          string
	Variant     NoteVariant
	Title       string
	Description string
}

// NotePort renders the notification system: a fixed, client-owned
// port where notes appear, and a hidden server-owned outbox that
// delivers server-rendered notes into it. The note-port controller
// clones each new outbox entry into the port, where it receives the
// standard mount, auto-dismiss, and swipe treatment. An entry only
// needs to reach the client once, so callers should render each note
// for a single update cycle and then drop it from notes.
// Client-origin notes (see notify in note-port.js) append directly
// into the port and never involve the server.
//
// Place NotePort once in the base layout.
func NotePort(notes []Note) domi.Node {
	return html.Div(
		stimulus.Controller(notePortController),
		stimulus.Action("visibilitychange@document->note-port#togglePaused"),
	)(
		domi.WithKeyOpaque("port", notePort()),
		domi.WithKey("outbox", noteOutbox(notes)),
	)
}

func notePort() domi.Node {
	return html.Div(
		attr.ID("note-port"),
		Class("u-note-port"),
		attr.Role("region"),
		Attr("aria-label")("Notifications"),
		stimulus.Target(notePortController, "port"),
		stimulus.Action("mouseenter->note-port#pause"),
		stimulus.Action("mouseleave->note-port#resume"),
	)()
}

func noteOutbox(notes []Note) domi.Node {
	var entries []domi.Node
	for _, n := range notes {
		entry := html.Div(
			stimulus.Target(notePortController, "outbox"),
		)(noteView(n))
		entries = append(entries, domi.WithKey(n.ID, entry))
	}
	return html.Div(domi.Name("hidden"))(entries...)
}

// noteView renders a single note.
func noteView(n Note) domi.Node {
	attrs := []domi.Attr{
		Class("u-note"),
		attr.Role("status"),
		Attr("aria-live")("polite"),
	}
	if n.Variant != NoteInfo {
		attrs = append(attrs, Attr("data-variant")(string(n.Variant)))
	}
	if n.Variant == NoteError {
		attrs = append(attrs, Destructive)
	}
	children := []domi.Node{NoteTitle()(domi.Text(n.Title))}
	if n.Description != "" {
		children = append(children, NoteDescription()(domi.Text(n.Description)))
	}
	return html.Div(attrs...)(children...)
}

// NoteTitle renders the title line of a note.
func NoteTitle(attrs ...domi.Attr) domi.Element {
	return html.Div(
		Class("u-note-title"),
		group(attrs...),
	)
}

// NoteDescription renders the body text of a note.
func NoteDescription(attrs ...domi.Attr) domi.Element {
	return html.Div(
		Class("u-note-description"),
		group(attrs...),
	)
}

// NoteIcon renders the leading icon of a note. The icon
// height matches the text line-height for alignment.
func NoteIcon(children ...domi.Node) domi.Node {
	return html.Div(
		Class("u-note-icon"),
	)(children...)
}

// NoteAction renders an action button inside a note.
func NoteAction(attrs ...domi.Attr) domi.Element {
	return html.Button(
		Class("u-note-action"),
		group(attrs...),
	)
}

// NoteActionLink renders a note action as an <a> with the given href.
func NoteActionLink(href string, attrs ...domi.Attr) domi.Element {
	return html.A(
		Class("u-note-action"),
		Href(href),
		group(attrs...),
	)
}
