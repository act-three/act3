package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
)

// base is the shared concrete View implementation.
// A view is a list of nodes rendered in sequence.
// Most views hold a single node, but Group and For hold one per member.
// An applied modifier modifies each node individually.
type base []node

func (v base) nodes() []node { return v }

func (v base) Modify(mods ...Modifier) View {
	for _, m := range mods {
		if m == nil {
			continue
		}
		out := make(base, len(v))
		for i, n := range v {
			out[i] = m.modify(n)
		}
		v = out
	}
	return v
}

func (v base) Attr(a ...domi.Attr) View {
	return v.Modify(modAttr{attr: domi.Group(a...)})
}

func (v base) Background(c Color) View {
	return v.Modify(Background(c))
}

func (v base) BorderShape(s Shape) View {
	return v.Modify(BorderShape(s))
}

func (v base) BorderStroke(px float64, c Color) View {
	return v.Modify(BorderStroke(px, c))
}

func (v base) Class(c ...string) View {
	return v.Modify(modAttr{attr: attr.Class(c...)})
}

func (v base) FixedSize() View {
	return v.
		Modify(modFixedSize{Horizontal | Vertical}).
		Class("ui-fixed-size")
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
	return v.Modify(modTag{name: name})
}

func (v base) LayerUnder(a Alignment, u View) View {
	return v.Modify(wrapLayer{view: u, over: false, alignment: a})
}

func (v base) LayerOver(a Alignment, o View) View {
	return v.Modify(wrapLayer{view: o, over: true, alignment: a})
}

func (v base) Padding(s ...EdgeSpace) View {
	var sum EdgeSpace
	for _, s := range s {
		sum = sum.add(s)
	}
	return v.Modify(wrapPadding{space: sum})
}

func (v base) Frame(o ...FrameOption) View {
	var w wrapFrame
	for _, o := range o {
		o.applyFrame(&w)
	}
	return v.Modify(w)
}

func (v base) FrameBounds(o ...FrameBoundsOption) View {
	var w wrapFrameBounds
	for _, o := range o {
		o.applyFrameBounds(&w)
	}
	return v.Modify(w)
}
