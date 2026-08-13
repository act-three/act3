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

// attrs returns the class indicating the role visually, if any.
func (r ButtonRole) attrs() domi.Attr {
	switch r {
	case RolePrimary:
		return attr.Class("ui-role-primary")
	case RoleDestructive:
		return attr.Class("ui-role-destructive")
	default:
		return nil
	}
}

// A ButtonView is a control that sends an event when clicked.
type ButtonView struct{ base }

// Button returns a button with the given label.
// It sends msg when clicked.
func Button[Msg any](label View, msg Msg) ButtonView {
	return ButtonView{base{buttonNode{label: label, onClick: event.Click(msg)}}}
}

// Role sets the semantic role of v.
func (v ButtonView) Role(r ButtonRole) ButtonView {
	n := v.base[0].(buttonNode)
	n.role = r
	v.base = base{n}
	return v
}

// Disabled disables v when d is true.
func (v ButtonView) Disabled(d bool) ButtonView {
	n := v.base[0].(buttonNode)
	n.disabled = d
	v.base = base{n}
	return v
}

type buttonNode struct {
	label    View
	onClick  domi.Attr
	role     ButtonRole
	disabled bool
}

// render delegates plan construction to the row the button is: an ordinary
// HStack whose element carries the button name, chrome, and behavior.
func (n buttonNode) render(env environment) plan {
	b := HStack(n.label).
		Tag("button").
		Class("ui-button").
		Attr(attr.Type("button"), n.onClick, n.role.attrs(), attr.Disabled(n.disabled)).(base)
	return b[0].render(env)
}
