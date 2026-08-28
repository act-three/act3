package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/event"

	"ily.dev/act3/xui/internal/canon"
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
	return linkView{textView{base{textLink{
		action: action,
		run:    label.text(),
	}}}}
}

type linkView struct{ textView }

func (v linkView) RequirePageLoad() LinkView {
	v.textView = v.styledWith(func(env *environment) {
		env.linkBypass = true
	})
	return v
}

// textLink performs an action when clicked.
type textLink struct {
	action any // URL string or onclick domi.Attr
	run    textRun
}

var _ textRun = textLink{}

// render lowers the link as a box.
// The box is itself the element that performs the action,
// so that box modifiers apply to the link.
func (l textLink) render(env environment) box {
	env.tag = l.tag()
	env.add(l.attrs(env.disabled, env.linkBypass))
	l.setStyles(&env.style, env.disabled)
	env.fg = append(env.fg, term[color]{value: Accent.color()})
	if env.disabled {
		env.opacity = append(env.opacity, term[float64]{value: 0.5})
	}
	return buildText(env, l.run)
}

func (l textLink) renderText(env environment) domi.Node {
	env.fg = append(env.fg, term[color]{value: Accent.color()})
	if env.disabled {
		env.opacity = append(env.opacity, term[float64]{value: 0.5})
	}
	var ss canon.StyleSet
	l.setStyles(&ss, env.disabled)
	styles := ss.Decls()
	for _, d := range env.paintUnder(0).decls(false) {
		styles.Set(d.property, d.value)
	}
	class := attr.Class(env.sheet.ClassFor(styles))
	attrs := l.attrs(env.disabled, env.linkBypass)
	env.nextenv = nextenv{}
	return domi.Tag(l.tag(), attrs, class)(l.run.renderText(env))
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
func (l textLink) attrs(disabled, bypass bool) domi.Attr {
	switch action := l.action.(type) {
	case string:
		if disabled {
			return domi.Group(attr.Role("link"), domi.Name("aria-disabled", "true"))
		}
		if bypass {
			return domi.Group(attr.Href(action), domi.Bypass)
		}
		return attr.Href(action)
	case domi.Attr:
		return domi.Group(attr.Type("button"), attr.Disabled(disabled), action)
	}
	panic("ui: unreachable")
}

func (l textLink) setStyles(ss *canon.StyleSet, disabled bool) {
	if disabled {
		ss.Set("cursor", "default")
	} else {
		ss.Set("cursor", "pointer")
	}
}
