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
