package hi

import (
	"context"
	"strconv"
	"time"

	"ily.dev/domi"
	"ily.dev/domi/attr"
)

// Notify displays a Note in the browser.
func Notify[Msg any](n Note) domi.Cmd[Msg] {
	return domi.Effect[Msg](notifyEffect{note: n})
}

// A Note specifies a message for the user
// with associated configuration.
// See [Notify].
type Note struct {
	Message     string        // Required primary text.
	Description string        // Optional supporting text.
	Icon        string        // Optional icon name. See IconSource.
	Action      View          // Optional Button or Link.
	Duration    time.Duration // Optional lifetime. Default is 4s.
}

type notifyEffect struct{ note Note }

func notifyHandler(_ context.Context, n notifyEffect) msg {
	return msgNotify{note: n.note}
}

type note struct {
	id string
	Note
}

func (n note) key() string { return n.id }

func (n note) view() View {
	return HStack(
		If(n.Icon != "", Icon(n.Icon)),
		VStack(
			Text(n.Message),
			If(n.Description != "", Text(n.Description)),
		).
			Alignment(Leading),
		HStack(n.action()).
			Tag("hi-note-action"),
	).
		modify(modStyle("max-height", cssLength(96i))).
		BorderClipped().
		Frame(Width(360), Top). // Top needed for animating height.
		Tag("hi-note").
		Attr(domi.Name("data-duration", strconv.FormatFloat(float64(n.Duration)/float64(time.Millisecond), 'f', -1, 64))).
		Background(backgroundColor).
		BorderStroke(1, Primary).
		WhileFocused(BorderOutline(2, 1, Accent)).
		Opacity(0). // starting opacity for entrance transition
		modify(modEnv(func(env environment) environment {
			env.style.Set("touch-action", "none")
			env.style.Set("user-select", "none")
			env.style.Set("-webkit-user-select", "none")
			return env
		}))
}

func (n note) action() View {
	if n.Action != nil {
		return n.Action
	}
	return Text("×").
		Tag("button").
		WhileFocused(BorderStroke(1, Accent)).
		Attr(
			attr.Type("button"),
			domi.Name("aria-label", "Dismiss"),
			domi.Name("data-dismiss", ""),
		).
		modify(modStyle("cursor", "pointer"))
}

func notePortOverlay(root View, notes []note) View {
	// JS clones each note from the outbox ZStack, which is hidden,
	// to the display ZStack, where JS owns the note lifecycle.
	return root.
		Overlay(Center, ZStack(ForEach(notes, note.key, note.view)).
			Tag("hi-note-outbox").
			modify(modStyle("content-visibility", "hidden"))).
		Overlay(Bottom, ZStack().
			Alignment(Bottom).
			Tag("hi-note-display").
			Attr(
				attr.Role("region"),
				domi.Name("aria-label", "Notifications"),
				domi.Name("aria-live", "polite"),
				domi.Name("aria-relevant", "additions text"),
				domi.Name("aria-atomic", "false"),
			).
			modify(modStyle("pointer-events", "auto")).
			modify(modBox(func(b box) box {
				b.node = domi.WithKeyOpaque("display", b.node)
				return b
			})).
			Padding(EdgeBottom(16)).
			modify(modStyle("pointer-events", "none")))
}
