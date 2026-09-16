package hi

import (
	"context"

	"ily.dev/domi"
	"ily.dev/domi/attr"
)

// Notify requests display of text as a notification in the browser.
func Notify[Msg any](text string) domi.Cmd[Msg] {
	return domi.Effect[Msg](notifyEffect{text: text})
}

type notifyEffect struct{ text string }

func notifyHandler(_ context.Context, n notifyEffect) msg {
	return msgNotify{text: n.text}
}

type note struct {
	id   string
	text string
}

func (n note) key() string { return n.id }

func (n note) view() View {
	return HStack(
		Text(n.text),
		Text("×").
			Tag("button").
			Attr(
				attr.Type("button"),
				domi.Name("aria-label", "Dismiss"),
			).
			modify(modStyle("cursor", "pointer")),
	).
		modify(modStyle("max-height", cssLength(96i))).
		Frame(Width(360), Top). // Top needed for animating height.
		Tag("hi-note").
		Background(backgroundColor).
		BorderStroke(1, Primary).
		BorderClipped().
		Opacity(0). // starting opacity for entrance transition
		modify(modEnv(func(env environment) environment {
			env.style.Set("touch-action", "none")
			env.style.Set("user-select", "none")
			env.style.Set("-webkit-user-select", "none")
			return env
		}))
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
