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
type modifier func(node) node

// modBox calls f to modify the box rendered by a node.
func modBox(f func(box) box) modifier {
	return func(n node) node {
		return func(env environment) box { return f(n(env)) }
	}
}
