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

// A ButtonView is a control that sends an event when clicked.
type ButtonView interface {
	View

	// Role sets the semantic role of the receiver.
	Role(ButtonRole) ButtonView

	// Disabled disables the receiver when d is true.
	Disabled(d bool) ButtonView
}

// Button returns a button with the given label.
// It sends msg when clicked.
func Button[Msg any](label View, msg Msg) ButtonView {
	return buttonView{base{buttonNode{label: label, onClick: event.Click(msg)}}}
}

type buttonView struct{ base }

func (v buttonView) Role(r ButtonRole) ButtonView {
	n := v.base[0].(buttonNode)
	n.role = r
	v.base = base{n}
	return v
}

func (v buttonView) Disabled(d bool) ButtonView {
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

func (n buttonNode) render(env environment) box {
	c := map[ButtonRole]Color{
		RolePrimary:     Accent,
		RoleDestructive: Danger,
	}[n.role]
	var fg Modifier
	if c != nil {
		fg = Foreground(CSSColor("#fff"))
	}
	v := HStack(n.label).
		Padding(EdgesLetterbox(8), EdgesPillarbox(12)).
		Tag("button").
		Modify(fg).
		Background(cmp.Or(c, surfaceColor)).
		BorderStroke(1, cmp.Or(c, borderColor)).
		BorderShape(RoundedRectangle).
		Attr(attr.Type("button"), n.onClick, attr.Disabled(n.disabled))
	cursor := "pointer"
	if n.disabled {
		cursor = "default"
		v = v.Opacity(0.5)
	}
	env.style.Set("cursor", cursor)
	return v.(base)[0].render(env)
}
