package ui

// A Modifier configures the behavior of a View.
// It can be applied with [View.Modify].
//
// It is usually more convenient to call modifier methods directly on View.
//
// A nil Modifier has no effect.
type Modifier interface {
	withState(State) modifier
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

type modBorderShape struct{ term[Shape] }

// BorderShape sets the shape of a view's border.
func BorderShape(s Shape) Modifier { return modBorderShape{value: s} }

func (m modBorderShape) withState(s State) modifier { m.state = s; return m }

func (m modBorderShape) modify(n node) node { return nodeTransform{f: m.environment, node: n} }

func (m modBorderShape) environment(env environment) environment {
	env.shape = append(env.shape, m.term)
	return env
}

type modOpacity struct{ term[float64] }

// Opacity sets a view's opacity to x, from 0 (transparent) to 1 (opaque).
func Opacity(x float64) Modifier { return modOpacity{value: x} }

func (m modOpacity) withState(s State) modifier { m.state = s; return m }

func (m modOpacity) modify(n node) node { return nodeTransform{f: m.environment, node: n} }

func (m modOpacity) environment(env environment) environment {
	env.opacity = append(env.opacity, m.term)
	return env
}

// NOTE: no exported construcor.
// modBox calls f to modify the box rendered by n.
type modBox struct {
	f func(box) box
	n node
}

func (m modBox) modify(n node) node { m.n = n; return m }

func (m modBox) render(env environment) box {
	return m.f(m.n.render(env))
}
