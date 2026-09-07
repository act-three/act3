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

	// Selected gives the receiver a selected appearance.
	Selected(bool) ButtonView

	// MenuOpen gives the receiver an appearance indicating
	// that an associated menu is open.
	MenuOpen(bool) ButtonView
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

func (v buttonView) Selected(selected bool) ButtonView {
	v.base = v.modify(modEnv(func(env environment) environment {
		env.buttonSelected = selected
		return env
	}))
	return v
}

func (v buttonView) MenuOpen(open bool) ButtonView {
	v.base = v.modify(modEnv(func(env environment) environment {
		env.buttonMenuOpen = open
		return env
	}))
	return v
}

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
		face, foreground := style.face, style.label
		if env.buttonSelected || env.buttonMenuOpen && !env.disabled {
			face = style.hover
		}
		if env.buttonSelected && env.buttonRole == RoleDefault {
			foreground = Headline
		}
		opacity := 1.0
		if env.disabled {
			foreground = Secondary
			if !env.buttonSelected {
				opacity = 0.6
			}
		}
		v := base{label}.
			LineLimit(1).
			Padding(Edges(padding)).
			Modify(font(fontSize, "500", lineHeight)).
			Foreground(foreground)
		if !env.disabled {
			v = v.
				WhileHovered(Background(style.hover)).
				WhilePressed(Background(style.hover))
		}
		v = v.
			Background(face).
			Opacity(opacity).
			BorderShape(Capsule)
		env.style.Set("cursor", "default")
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
