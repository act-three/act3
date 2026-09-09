package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/event"
)

// A ButtonView is a control that performs an action when clicked.
type ButtonView interface {
	View

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

func nodeButton(action any, label node) node {
	return func(env environment) box {
		fontSize, lineHeight, padding := buttonMetrics(env.controlSize)
		style, ok := buttonRecipe[env.buttonStyle]
		if !ok {
			style = buttonRecipe[Bordered]
		}
		face := style.face
		edge := style.edge
		foreground := style.label
		hoverForeground := style.hoverLabel
		if env.buttonSelected {
			face = style.selected
			edge = style.hoverEdge
			foreground = style.selectedLabel
		}
		// Bordered keeps its selected label color during interaction.
		if env.buttonSelected && env.buttonStyle == Bordered {
			hoverForeground = style.selectedLabel
		}
		if env.buttonMenuOpen {
			face = style.hover
			edge = style.hoverEdge
			foreground = hoverForeground
		}
		if env.disabled && style.disabledLabel != nil {
			foreground = style.disabledLabel
		}
		_, isLink := action.(string)
		opacity := 1.0
		if env.disabled && (!isLink || !env.buttonSelected) {
			opacity = 0.6
		}
		v := base{label}.
			LineLimit(1).
			Padding(Edges(padding)).
			BorderStroke(0.5, edge).
			Modify(font(fontSize, "500", lineHeight))
		if !env.disabled {
			v = v.
				WhileHovered(BorderStroke(0.5, style.hoverEdge)).
				WhilePressed(BorderStroke(0.5, style.hoverEdge)).
				WhileHovered(Background(style.hover)).
				WhileHovered(Foreground(hoverForeground)).
				WhilePressed(Background(style.hover)).
				WhilePressed(Foreground(hoverForeground))
		}
		v = v.
			Foreground(foreground).
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

// The edge paints within the padding, so ordinary text reaches the
// 24/28/32/44px target heights without a layout-affecting border.
func buttonMetrics(s ControlSize) (fontSize, lineHeight string, padding complex128) {
	switch s {
	case Mini:
		return "12px", "16px", 4
	case Small:
		return "12px", "16px", 6
	case Large:
		return "13px", "18px", 13
	default:
		return "13px", "18px", 7
	}
}

// buttonStyle contains only paint supported by the shared modifiers.
// Shadows, focus outlines, and transitions remain deferred.
type buttonStyle struct {
	face          Color
	hover         Color
	selected      Color
	edge          Color
	hoverEdge     Color
	label         Color
	hoverLabel    Color
	selectedLabel Color
	disabledLabel Color // nil preserves the foreground from other states
}

var buttonRecipe = map[ButtonStyle]buttonStyle{
	Bordered: {
		face:          controlSecondary,
		hover:         controlSecondaryHover,
		selected:      controlSecondaryHover,
		edge:          controlSecondaryEdge,
		hoverEdge:     controlSecondaryEdgeHover,
		label:         Primary,
		hoverLabel:    Primary,
		selectedLabel: Headline,
		disabledLabel: Secondary,
	},
	Prominent: {
		face:          Accent,
		hover:         accentHover,
		selected:      accentHover,
		edge:          Transparent,
		hoverEdge:     Transparent,
		label:         accentTextColor,
		hoverLabel:    accentTextColor,
		selectedLabel: accentTextColor,
	},
	Subtle: {
		face:          Transparent,
		hover:         controlTertiaryHover,
		selected:      controlTertiarySelected,
		edge:          Transparent,
		hoverEdge:     Transparent,
		label:         Primary,
		hoverLabel:    Headline,
		selectedLabel: Headline,
	},
	Borderless: {
		face:          Transparent,
		hover:         Transparent,
		selected:      Transparent,
		edge:          Transparent,
		hoverEdge:     Transparent,
		label:         Primary,
		hoverLabel:    Headline,
		selectedLabel: Headline,
	},
	Destructive: {
		face:          Red,
		hover:         redHover,
		selected:      redHover,
		edge:          Transparent,
		hoverEdge:     Transparent,
		label:         White,
		hoverLabel:    White,
		selectedLabel: White,
	},
	DestructiveSubtle: {
		face:          Transparent,
		hover:         redTint,
		selected:      Transparent,
		edge:          Transparent,
		hoverEdge:     Transparent,
		label:         redText,
		hoverLabel:    redText,
		selectedLabel: redText,
		disabledLabel: Primary,
	},
}
