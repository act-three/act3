package ui

import (
	"cmp"

	"ily.dev/domi"
)

// A Modifier configures the behavior of a View.
// It can be applied with [View.Modify].
//
// It is usually more convenient to call modifier methods directly on View.
//
// A nil Modifier has no effect.
type Modifier interface {
	modifier
	withState(State) Modifier
}

// A modifier configures the behavior of a View.
// It can be applied with View.modify.
type modifier interface {
	modify(node) node
}

// A nodeEnv adjusts the environment for its subtree.
type nodeEnv struct {
	f    func(environment) environment
	node node
}

func (m nodeEnv) render(env environment) box {
	return m.node.render(m.f(env))
}

// A nodeTransform adjusts the environment for its subtree, like nodeEnv.
// Additionally, if the environment has any paint modifiers
// (as indicated by nextenv.hasPaint),
// it first emits a layout-preserving wrapper box
// that applies all env modifiers,
// thus clearing any pending paint modifiers
// before applying f and rendering node.
type nodeTransform struct {
	f    func(environment) environment
	node node
}

func (m nodeTransform) render(env environment) box {
	if !env.hasPaint {
		return m.node.render(m.f(env))
	}
	return wrapMod(env, nodeEnv{f: m.f, node: m.node})
}

// NOTE: no exported construcor.
type modAttr struct{ attr domi.Attr }

func (m modAttr) modify(n node) node { return nodeEnv{f: m.environment, node: n} }

func (m modAttr) environment(env environment) environment {
	env.add(m.attr)
	return env
}

type modBackground struct{ term[color] }

// Background fills the background of a view with c.
func Background(c Color) Modifier { return modBackground{value: c.color()} }

func (m modBackground) withState(s State) Modifier { m.state = s; return m }

func (m modBackground) modify(n node) node { return nodeEnv{f: m.environment, node: n} }

func (m modBackground) environment(env environment) environment {
	env.bg = append(env.bg, m.term)
	env.hasPaint = true
	return env
}

type modBorderShape struct{ term[Shape] }

// BorderShape sets the shape of a view's border.
func BorderShape(s Shape) Modifier { return modBorderShape{value: s} }

func (m modBorderShape) withState(s State) Modifier { m.state = s; return m }

func (m modBorderShape) modify(n node) node { return nodeTransform{f: m.environment, node: n} }

func (m modBorderShape) environment(env environment) environment {
	env.shape = append(env.shape, m.term)
	return env
}

type modBorderStroke struct{ term[stroke] }

// BorderStroke draws a line
// of the given width and color
// over the inside edge of a view.
//
// The stroke paints over the view's content,
// inside its border shape.
// It takes no layout space.
// To add a border around the outside of a view,
// add padding inside the border.
func BorderStroke(px float64, c Color) Modifier {
	if !(px > 0) { // this is written weird b/c of NaNs lmao
		return nil
	}
	return modBorderStroke{value: stroke{px, c.color()}}
}

func (m modBorderStroke) withState(s State) Modifier { m.state = s; return m }

func (m modBorderStroke) modify(n node) node { return nodeEnv{f: m.environment, node: n} }

func (m modBorderStroke) environment(env environment) environment {
	env.stroke = append(env.stroke, m.term)
	env.hasPaint = true
	return env
}

// NOTE: no exported construcor.
type modDisabled struct{ d bool }

func (m modDisabled) modify(n node) node { return nodeEnv{f: m.environment, node: n} }

func (m modDisabled) environment(env environment) environment {
	env.disabled = env.disabled || m.d
	return env
}

// NOTE: no exported construcor.
type modFixedSize struct{ axes AxisSet }

func (m modFixedSize) modify(n node) node { return nodeEnv{f: m.environment, node: n} }

// environment gives the subtree unbounded available space on the given axes,
// so the subtree resolves to its ideal size there (see [build]).
// The subtree's outermost box is a fill boundary:
// it must sit in its real container at that resolved size,
// even when the container has slack to offer.
func (m modFixedSize) environment(env environment) environment {
	env.unbounded |= m.axes
	env.fillMask |= m.axes
	// Note that a box can override max-content with its own declaration.
	if m.axes.hasAll(Horizontal) {
		env.style.Set("width", "max-content")
	}
	if m.axes.hasAll(Vertical) {
		env.style.Set("height", "max-content")
	}
	return env
}

type modFont struct{ term[FontSize] }

// Font sets the font size for text in a view.
func Font(f FontSize) Modifier {
	if f == "" {
		return nil
	}
	return modFont{value: f}
}

func (m modFont) withState(s State) Modifier { m.state = s; return m }

func (m modFont) modify(n node) node { return nodeEnv{f: m.environment, node: n} }

func (m modFont) environment(env environment) environment {
	size, weight, height := m.value.values()
	env.fontSize = append(env.fontSize, term[string]{state: m.state, value: size})
	env.fontWeight = append(env.fontWeight, term[string]{state: m.state, value: weight})
	env.lineHeight = append(env.lineHeight, term[string]{state: m.state, value: height})
	return env
}

type modForeground struct{ term[color] }

// Foreground uses c to draw foreground elements in a view,
// such as text.
func Foreground(c Color) Modifier { return modForeground{value: c.color()} }

func (m modForeground) withState(s State) Modifier { m.state = s; return m }

func (m modForeground) modify(n node) node { return nodeEnv{f: m.environment, node: n} }

func (m modForeground) environment(env environment) environment {
	env.fg = append(env.fg, m.term)
	return env
}

type modOpacity struct{ term[float64] }

// Opacity sets a view's opacity to x, from 0 (transparent) to 1 (opaque).
func Opacity(x float64) Modifier { return modOpacity{value: x} }

func (m modOpacity) withState(s State) Modifier { m.state = s; return m }

func (m modOpacity) modify(n node) node { return nodeTransform{f: m.environment, node: n} }

func (m modOpacity) environment(env environment) environment {
	env.opacity = append(env.opacity, m.term)
	return env
}

// NOTE: no exported constructor.
// modStyle emits one CSS declaration onto a subview's outermost box.
// Containers can use it to apply a style to their direct subviews.
type modStyle struct{ property, value string }

func (m modStyle) modify(n node) node { return nodeEnv{f: m.environment, node: n} }

func (m modStyle) environment(env environment) environment {
	env.style.Set(m.property, m.value)
	return env
}

// NOTE: no exported construcor.
type modLineLimit struct{ n int }

func (m modLineLimit) modify(n node) node { return nodeEnv{f: m.environment, node: n} }

func (m modLineLimit) environment(env environment) environment {
	env.lineLimit = m.n
	return env
}

// NOTE: no exported construcor.
type modTag struct{ name string }

func (m modTag) modify(n node) node { return nodeEnv{f: m.environment, node: n} }

func (m modTag) environment(env environment) environment {
	env.tag = m.name
	return env
}

type modTagDefault struct{ name string }

func (m modTagDefault) modify(n node) node { return nodeEnv{f: m.environment, node: n} }

func (m modTagDefault) environment(env environment) environment {
	env.tag = cmp.Or(env.tag, m.name)
	return env
}
