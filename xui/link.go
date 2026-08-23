package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/event"

	"ily.dev/act3/xui/internal/sheet"
)

// A LinkView is a span of text that performs an action when clicked.
type LinkView interface {
	TextView

	// RequirePageLoad requires a URL link to use the browser's
	// built-in navigation.
	//
	// Ordinarily, a link to a same-origin URL is handled by domi,
	// which updates the URL bar and patches the DOM instead of
	// using the browser's built-in navigation to load the new URL.
	// RequirePageLoad bypasses this behavior.
	//
	// If the receiver is a link that sends a message,
	// RequirePageLoad has no effect.
	RequirePageLoad() LinkView

	// Disabled disables the receiver when d is true.
	Disabled(d bool) LinkView
}

// Link returns a link with the given label.
//
// If a is a string, it must be a URL,
// and the link navigates to it.
// Otherwise, the link sends a to [domi.App.Update].
//
// If a is not a string
// or the app's Msg type or a type that implements it,
// the LinkView panics.
func Link[Action any](a Action, label TextView) LinkView {
	var action any = a
	if _, ok := action.(string); !ok {
		action = event.Click(a)
	}
	return linkView{textView{base{linkNode{textLink{
		action: action,
		run:    label.text(),
	}}}}}
}

type linkView struct{ textView }

func (v linkView) RequirePageLoad() LinkView {
	n := v.base[0].(linkNode)
	n.bypass = true
	v.base = base{n}
	return v
}

func (v linkView) Disabled(d bool) LinkView {
	n := v.base[0].(linkNode)
	n.disabled = d
	v.base = base{n}
	return v
}

// linkNode is the box of a link view.
// The box is itself the element that performs the action,
// so that box modifiers apply to the link.
type linkNode struct{ textLink }

func (n linkNode) text() textRun { return n.textLink }

func (n linkNode) render(env environment) box {
	env.tag = n.tag()
	env.add(n.attrs())
	n.setStyles(&env.style)
	linkColor := func(s *textStyle) { s.color = Accent.color() }
	return textNode{textMod{f: linkColor, run: n.run}}.render(env)
}

// textLink performs an action when clicked.
type textLink struct {
	action   any // URL string or onclick domi.Attr
	bypass   bool
	disabled bool
	run      textRun
}

func (l textLink) html(env textenv) domi.Node {
	env.style.color = Accent.color()
	var ss sheet.StyleSet
	l.setStyles(&ss)
	env.style.setStyles(&ss)
	env.style = textStyle{}
	class := attr.Class(env.sheet.ClassFor(ss))
	return domi.Tag(l.tag(), l.attrs(), class)(l.run.html(env))
}

// tag returns the name of the element that performs the action.
func (l textLink) tag() string {
	if _, ok := l.action.(string); ok {
		return "a"
	}
	return "button"
}

// attrs returns the attributes that perform the action,
// or that mark the element disabled.
func (l textLink) attrs() domi.Attr {
	switch action := l.action.(type) {
	case string:
		if l.disabled {
			return domi.Group(attr.Role("link"), domi.Name("aria-disabled", "true"))
		}
		if l.bypass {
			return domi.Group(attr.Href(action), domi.Bypass)
		}
		return attr.Href(action)
	case domi.Attr:
		return domi.Group(attr.Type("button"), attr.Disabled(l.disabled), action)
	}
	panic("ui: unreachable")
}

func (l textLink) setStyles(ss *sheet.StyleSet) {
	if l.disabled {
		ss.Set("cursor", "default")
		ss.Set("opacity", "0.5")
	} else {
		ss.Set("cursor", "pointer")
	}
}
