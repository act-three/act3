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

// NOTE: no exported construcor.
// A modBox modifies the box rendered by a node.
type modBox func(box) box

func (m modBox) modify(n node) node {
	return func(env environment) box { return m(n(env)) }
}
