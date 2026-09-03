package ui

import (
	"cmp"
	"fmt"

	"ily.dev/domi"
	"ily.dev/domi/attr"
)

// base is the shared concrete View implementation.
// A view is a list of nodes rendered in sequence.
// Most views hold a single node, but Group and For hold one per member.
// An applied modifier modifies each node individually.
type base []node

func (v base) nodes() []node { return v }

func (v base) Modify(m Modifier, states ...State) View {
	if m == nil {
		return v
	}
	var s State
	for _, x := range states {
		s |= x
	}
	return v.modify(m.withState(s))
}

func (v base) modify(m modifier) base {
	if m == nil {
		return v
	}
	out := make(base, len(v))
	for i, n := range v {
		out[i] = m.modify(n)
	}
	return out
}

func (v base) WhileHovered(m Modifier) View     { return v.Modify(m, Hovered) }
func (v base) WhileFocused(m Modifier) View     { return v.Modify(m, Focused) }
func (v base) WhilePressed(m Modifier) View     { return v.Modify(m, Pressed) }
func (v base) WhileDisabled(m Modifier) View    { return v.Modify(m, Disabled) }
func (v base) WhileChecked(m Modifier) View     { return v.Modify(m, Checked) }
func (v base) WhileInvalid(m Modifier) View     { return v.Modify(m, Invalid) }
func (v base) WhilePlaceholder(m Modifier) View { return v.Modify(m, Placeholder) }

func (v base) Attr(a ...domi.Attr) View {
	return v.modify(modAttr(domi.Group(a...)))
}

func (v base) Background(c Color) View {
	return v.Modify(Background(c))
}

func (v base) BorderClipped() View {
	return v.modify(modTransform(func(env environment) environment {
		env.style.Set("overflow-x", "clip")
		env.style.Set("overflow-y", "clip")
		return env
	}))
}

func (v base) BorderShape(s Shape) View {
	return v.Modify(BorderShape(s))
}

func (v base) BorderStroke(px float64, c Color) View {
	return v.Modify(BorderStroke(px, c))
}

func (v base) Class(c ...string) View {
	return v.modify(modAttr(attr.Class(c...)))
}

func (v base) Disabled(d bool) View {
	return v.modify(modEnv(func(env environment) environment {
		env.disabled = env.disabled || d
		return env
	}))
}

func (v base) FixedSize() View {
	return v.
		modify(modFixedSize(Horizontal | Vertical)).
		Class("ui-fixed-size")
}

func (v base) LinkPolicy(p LinkPolicy) View {
	return v.modify(modEnv(func(env environment) environment {
		env.linkPolicy = p
		return env
	}))
}

func (v base) LineLimit(n int) View {
	return v.modify(modEnv(func(env environment) environment {
		env.lineLimit = max(n, 1)
		return env
	}))
}

func (v base) Font(f FontSize) View {
	return v.Modify(Font(f))
}

func (v base) Foreground(c Color) View {
	return v.Modify(Foreground(c))
}

func (v base) Opacity(x float64) View {
	return v.Modify(Opacity(x))
}

func (v base) Tag(name string) View {
	return v.modify(modEnv(func(env environment) environment {
		env.tag = name
		return env
	}))
}

func (v base) Title(t string) View {
	return v.modify(modBox(func(b box) box {
		b.title = cmp.Or(b.title, t)
		return b
	}))
}

func (v base) Underlay(a Alignment, u View) View {
	return v.UnderlayAt(a, a, u)
}

func (v base) UnderlayAt(at, anchor Alignment, u View) View {
	return v.modify(wrapLayer{
		layer:  unary(ZStack, u),
		over:   false,
		at:     at,
		anchor: anchor,
	})
}

func (v base) Overlay(a Alignment, o View) View {
	return v.OverlayAt(a, a, o)
}

func (v base) OverlayAt(at, anchor Alignment, o View) View {
	return v.modify(wrapLayer{
		layer:  unary(ZStack, o),
		over:   true,
		at:     at,
		anchor: anchor,
	})
}

func (v base) Padding(s ...EdgeSpace) View {
	return v.modify(wrapPadding{space: edgeSum(s...)})
}

func (v base) Sticky(s ...EdgeSpace) View {
	return v.modify(wrapSticky{inset: edgeSum(s...)})
}

func (v base) Frame(o ...FrameOption) View {
	var w wrapFrame
	for _, o := range o {
		o.applyFrame(&w)
	}
	return v.modify(w)
}

func (v base) FrameBounds(o ...FrameBoundsOption) View {
	var w wrapFrameBounds
	for _, o := range o {
		o.applyFrameBounds(&w)
	}
	return v.modify(w)
}

func (v base) FrameRatio(w, h int, anchor AxisSet, o ...FrameRatioOption) View {
	if !(w > 0 && h > 0) {
		panic(fmt.Sprintf("ui: FrameRatio(%d, %d) is not a ratio", w, h))
	}
	if anchor != Horizontal && anchor != Vertical {
		panic("ui: FrameRatio anchor must be Horizontal or Vertical")
	}
	f := wrapFrameRatio{w: w, h: h, anchor: anchor}
	for _, o := range o {
		o.applyFrameRatio(&f)
	}
	return v.modify(f)
}
