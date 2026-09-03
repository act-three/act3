package ui

import (
	"fmt"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/event"

	"ily.dev/act3/xui/internal/canon"
)

// LinkPolicy specifies which link navigations the app handles
// and which it leaves to the browser.
// See [View.LinkPolicy].
//
// When the app handles a navigation,
// it updates the page in place
// instead of loading the new URL from scratch.
type LinkPolicy int

const (
	// HandleSameOrigin handles links to the app's own origin
	// and leaves the rest to the browser.
	HandleSameOrigin LinkPolicy = iota

	// HandleAll handles every link.
	HandleAll

	// HandleNone leaves every link to the browser.
	HandleNone
)

// attr returns the annotation that requests p on a navigating element,
// or nil for the default policy.
func (p LinkPolicy) attr() domi.Attr {
	switch p {
	case HandleSameOrigin:
		return nil
	case HandleAll:
		return domi.HandleLink("yes")
	case HandleNone:
		return domi.HandleLink("no")
	}
	panic(fmt.Sprintf("ui: invalid LinkPolicy %d", p))
}

// Link returns a link with the given label.
//
// If a is a string, it must be a URL,
// and the link navigates to it.
// Otherwise, the link sends a to [domi.App.Update].
//
// If a is not a string
// or the app's Msg type or a type that implements it,
// the link panics.
func Link[Action any](a Action, label TextView) TextView {
	var action any = a
	if _, ok := action.(string); !ok {
		action = event.Click(a)
	}
	return newTextView(textLink{
		action: action,
		run:    label.text(),
	})
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
	env.add(l.attrs(env.disabled, env.linkPolicy))
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
	attrs := l.attrs(env.disabled, env.linkPolicy)
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
func (l textLink) attrs(disabled bool, policy LinkPolicy) domi.Attr {
	switch action := l.action.(type) {
	case string:
		if disabled {
			return domi.Group(attr.Role("link"), domi.Name("aria-disabled", "true"))
		}
		return domi.Group(attr.Href(action), policy.attr())
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
