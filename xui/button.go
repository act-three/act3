package ui

import (
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
	return buttonView{base{nodeButton(action, unary(HStack, label))}}
}

type buttonView struct{ base }

func (v buttonView) Role(r ButtonRole) ButtonView {
	v.base = v.modify(modEnv(func(env environment) environment {
		env.buttonRole = r
		return env
	}))
	return v
}

func nodeButton(action any, label node) node {
	return func(env environment) box {
		fontSize, lineHeight, padding := buttonMetrics(env.controlSize)
		style := map[ButtonRole]struct{ face, hover, label Color }{
			RoleDefault:     {controlSecondary, controlSecondaryHover, Primary},
			RolePrimary:     {Accent, accentHover, accentTextColor},
			RoleDestructive: {Red, hoverOf(Red.color()), White},
		}[env.buttonRole]
		v := base{label}.
			LineLimit(1).
			Padding(Edges(padding)).
			Modify(font(fontSize, "500", lineHeight)).
			Foreground(style.label).
			WhileHovered(Background(style.hover)).
			Background(style.face).
			BorderShape(Capsule)
		cursor := "pointer"
		if env.disabled {
			cursor = "default"
			v = v.Opacity(0.5)
		}
		env.style.Set("cursor", cursor)
		switch action := action.(type) {
		case string:
			v = v.Tag("a")
			if env.disabled {
				v = v.Attr(
					attr.Role("link"),
					domi.Name("aria-disabled", "true"),
				)
			} else {
				v = v.Attr(attr.Href(action), env.linkPolicy.attr())
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
		return v.nodes()[0](env)
	}
}

// These provisional recipes leave 1px per axis for the eventual 0.5px
// reserved border: ordinary text then targets 24/28/32/44px heights.
// Review the geometry again when that paint construction is available.
func buttonMetrics(s ControlSize) (fontSize, lineHeight string, padding float64) {
	switch s {
	case Mini:
		return "12px", "16px", 3.5
	case Small:
		return "12px", "16px", 5.5
	case Large:
		return "13px", "18px", 12.5
	default:
		return "13px", "18px", 6.5
	}
}
