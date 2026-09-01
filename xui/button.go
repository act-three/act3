package ui

import (
	"cmp"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/event"
)

// ButtonRole specifies the purpose of a button.
// Button views use it to present the button appropriately,
// such as through its visual appearance.
type ButtonRole int

const (
	RoleDefault ButtonRole = iota
	RolePrimary
	RoleDestructive
)

// A ButtonView is a control that performs an action when clicked.
type ButtonView interface {
	View

	// Role sets the semantic role of the receiver.
	Role(ButtonRole) ButtonView
}

// Button returns a button with the given label.
//
// If a is a string, it must be a URL,
// and the button navigates to it.
// Otherwise, the button sends a to [domi.App.Update].
//
// If a is not a string
// or the app's Msg type or a type that implements it,
// the ButtonView panics.
func Button[Action any](a Action, label View) ButtonView {
	var action any = a
	if _, ok := action.(string); !ok {
		action = event.Click(a)
	}
	return buttonView{base{buttonNode{
		label:  unary(HStack, label),
		action: action,
	}}}
}

type buttonView struct{ base }

func (v buttonView) Role(r ButtonRole) ButtonView {
	v.base = v.modify(modEnv(func(env environment) environment {
		env.buttonRole = r
		return env
	}))
	return v
}

type buttonNode struct {
	label  node
	action any // URL string or onclick domi.Attr
}

func (n buttonNode) render(env environment) box {
	c := map[ButtonRole]Color{
		RolePrimary:     Accent,
		RoleDestructive: Danger,
	}[env.buttonRole]
	var fg Modifier
	if c != nil {
		fg = Foreground(CSSColor("#fff"))
	}
	v := base{n.label}.
		Padding(EdgesLetterbox(8), EdgesPillarbox(12)).
		Modify(fg).
		Background(cmp.Or(c, surfaceColor)).
		BorderStroke(1, cmp.Or(c, borderColor)).
		BorderShape(RoundedRectangle)
	cursor := "pointer"
	if env.disabled {
		cursor = "default"
		v = v.Opacity(0.5)
	}
	env.style.Set("cursor", cursor)
	switch action := n.action.(type) {
	case string:
		v = v.Tag("a")
		if env.disabled {
			v = v.Attr(
				attr.Role("link"),
				domi.Name("aria-disabled", "true"),
			)
		} else {
			v = v.Attr(attr.Href(action))
		}
	case domi.Attr:
		v = v.
			Tag("button").
			Attr(
				attr.Type("button"),
				attr.Disabled(env.disabled),
				action,
			)
	}
	return v.nodes()[0].render(env)
}
