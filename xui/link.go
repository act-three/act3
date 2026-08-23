package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/event"
	"ily.dev/domi/html"

	"ily.dev/act3/xui/internal/sheet"
)

// A Link displays the given label
// and activates when clicked.
//
// If a is a string, it must be a URL,
// and the link navigates to it.
// Otherwise, the link sends a to [domi.App.Update].
//
// If a is not a string
// or the app's Msg type or a type that implements it,
// the returned TextView panics.
func Link[Action any](a Action, label TextView) TextView {
	var action any = a
	if _, ok := action.(string); !ok {
		action = event.Click(a)
	}
	return textView{base{textNode{textLink{
		action: action,
		run:    label.node().run,
	}}}}
}

// textLink performs an action when clicked.
type textLink struct {
	action any // URL string or onclick domi.Attr
	run    textRun
}

func (l textLink) html(env textenv) domi.Node {
	env.style.color = Accent.color()
	var ss sheet.StyleSet
	ss.Set("cursor", "pointer")
	env.style.setStyles(&ss)
	env.style = textStyle{}
	class := attr.Class(env.sheet.ClassFor(ss))
	switch action := l.action.(type) {
	case string:
		return html.A(attr.Href(action), class)(l.run.html(env))
	case domi.Attr:
		return html.Button(attr.Type("button"), action, class)(l.run.html(env))
	}
	panic("ui: unreachable")
}
