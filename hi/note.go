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
	return Text(n.text)
}

func notePort(notes []note) View {
	port := view(func(env environment) box {
		env.tag = "hi-note-port"
		env.style.Set("display", "block")
		b := build(env, plan{})
		b.node = domi.WithKeyOpaque("port", b.node)
		return b
	}).
		Attr(attr.Role("status"), domi.Name("aria-live", "polite"))

	entries := ForEach(notes, note.key, note.view)
	outbox := view(func(env environment) box {
		env.tag = "hi-note-outbox"
		env.style.Set("display", "none")
		return build(env, renderSubviewList(env, entries))
	})

	return VStack(port, outbox)
}
