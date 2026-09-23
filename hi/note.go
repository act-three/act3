package hi

import (
	"context"
	"fmt"
	"time"

	"ily.dev/domi"
	"ily.dev/domi/attr"
)

// Notify displays a Note in the browser.
func Notify[Msg any](n Note) domi.Cmd[Msg] {
	return domi.Effect[Msg](notifyEffect{note: n})
}

// RegisterNote registers a named note template
// for the JavaScript function Hi.notify.
//
//	RegisterNote[Msg]("upload-failed", Note{
//		Icon:        "circle-x",
//		Message:     "Upload failed",
//		Description: "Could not reach the server",
//	})
//
// If name is empty, RegisterNote panics.
//
// After registration,
// application JavaScript can display the note
// without a server request:
//
//	Hi.notify("upload-failed");
//
// Each call displays a fresh copy with its own lifetime.
func RegisterNote[Msg any](name string, n Note) domi.Cmd[Msg] {
	if name == "" {
		panic("hi: RegisterNote requires a nonempty name")
	}
	return domi.Effect[Msg](notifyEffect{note: n, template: name})
}

// A Note specifies a message for the user
// with associated configuration.
// See [Notify] and [RegisterNote].
type Note struct {
	Message     string        // Required primary text.
	Description string        // Optional supporting text.
	Icon        string        // Optional icon name. See IconSource.
	Action      View          // Optional Button or Link.
	Duration    time.Duration // Optional lifetime. Default is 4s.
}

type notifyEffect struct {
	note     Note
	template string
}

func notifyHandler(_ context.Context, n notifyEffect) msg {
	return msgNotify{note: n.note, template: n.template}
}

type note struct {
	id       string
	template string
	Note
}

func (n note) key() string { return n.id }

func (n note) view() View {
	return HStack(
		n.icon(),
		VStack(
			n.message(),
			n.description(),
			n.action(),
		).
			Alignment(Leading).
			Gap(6i).
			ControlSize(Small),
		Spacer(),
		n.dismiss(),
	).
		Alignment(FirstBaseline).
		Gap(8i).
		Padding(Edges(12i)).
		Font(SizeEmAbs(13i, 16i)).
		modify(modStyle("max-height", cssLength(112i))).
		BorderClipped().
		Frame(Width(360), Top). // Top needed for animating height.
		Tag("hi-note").
		Attr(
			domi.Name("data-duration", fmt.Sprint(n.Duration.Milliseconds())),
			n.templateAttr(),
		).
		BorderStroke(0.5, borderColor).
		WhileFocused(BorderOutline(0, 1, Accent)).
		modify(shadowMedium).
		ThemeBackground(menuColor).
		BorderShape(RoundedRectangle(12i)).
		Opacity(0). // starting opacity for entrance transition
		modify(modEnv(func(env environment) environment {
			env.style.Set("touch-action", "none")
			env.style.Set("user-select", "none")
			env.style.Set("-webkit-user-select", "none")
			return env
		}))
}

func (n note) message() View {
	return Text(n.Message).
		Foreground(Headline).
		Font(Medium).
		LineLimit(2)
}

func (n note) description() View {
	return If(n.Description != "", Text(n.Description)).
		Foreground(Secondary).
		Font(Weight(450)).
		LineLimit(3)
}

func (n note) icon() View {
	return If(n.Icon != "", Icon(n.Icon)).
		Foreground(Secondary)
}

func (n note) action() View {
	return If(n.Action != nil, n.Action).
		Padding(EdgeTop(4i))
}

func (n note) dismiss() View {
	return Button(noAction{}, Icon("x")).
		ButtonStyle(Subtle).
		Attr(
			domi.Name("aria-label", "Dismiss"),
			domi.Name("data-dismiss", ""),
		).
		ControlSize(Mini)
}

func (n note) templateAttr() domi.Attr {
	if n.template != "" {
		return domi.Name("data-note-template", n.template)
	}
	return domi.Group()
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
			Padding(EdgeBottom(16i)).
			modify(modStyle("pointer-events", "none")))
}
