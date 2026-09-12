package hi

import (
	"slices"

	"ily.dev/domi"
)

// Empty is an empty view.
// It occupies no space.
// Modifiers applied to it have no effect.
func Empty() View { return view() }

// Lazy defers the construction of a view.
//
// When the lazy view is rendered,
// it calls f and renders the result.
//
// If the lazy view is not rendered,
// it does not call f.
// For example, If(false, Lazy(f)) never calls f.
func Lazy(f func() View) View {
	return base(func(renv resenv) []node {
		return f().resolve(renv)
	})
}

// A Group is a sequence of views.
// It contributes the views to its container
// as if they had been written there directly.
//
// A modifier applied to a Group is applied to each member individually.
// These are equivalent:
//
//	Group(a, b).Background(Red).Padding()
//	Group(
//	    a.Background(Red).Padding(),
//	    b.Background(Red).Padding(),
//	)
func Group(v ...View) View {
	v = slices.Clone(v)
	return base(func(renv resenv) []node {
		var ns []node
		for _, c := range v {
			ns = append(ns, c.resolve(renv)...)
		}
		return ns
	})
}

// If returns v if cond is true and [Empty] otherwise.
func If(cond bool, v View) View { return IfElse(cond, v, Empty()) }

// IfElse returns a if cond is true and b otherwise.
func IfElse(cond bool, a, b View) View {
	if cond {
		return a
	}
	return b
}

// When calls f and returns the result when cond is true.
// Otherwise, it returns [Empty].
func When(cond bool, f func() View) View { return WhenElse(cond, f, Empty) }

// WhenElse calls a when cond is true
// or else b.
// It returns the result.
func WhenElse(cond bool, a, b func() View) View {
	if cond {
		return a()
	}
	return b()
}

// First displays the first of the given views that is not empty.
// The remaining views are discarded and not rendered.
// If all of the given views are empty,
// the returned view is also empty.
func First(v ...View) View {
	v = slices.Clone(v)
	return base(func(renv resenv) []node {
		for _, c := range v {
			if ns := c.resolve(renv); len(ns) > 0 {
				return ns
			}
		}
		return nil
	})
}

// ForEach calls f once for each item in items,
// and returns the resulting views as a [Group].
//
// If key is not nil,
// ForEach also calls key for each item,
// and uses the returned string
// as the item's key.
// Reordering keyed items
// moves their HTML elements intact to their new positions
// instead of replacing their contents.
// This lets the browser retain focus, scroll position,
// and other view state that exists only on the browser.
//
// A key must be nonempty,
// stable (any given item should be given the same key every time),
// and unique within the enclosing view.
//
// If f returns a Group,
// the key is assigned to its first member.
func ForEach[T any, S ~[]T](items S, key func(T) string, f func(T) View) View {
	var vs []View
	for _, it := range items {
		v := f(it)
		if key != nil {
			v = withKey(key(it), v)
		}
		vs = append(vs, v)
	}
	return Group(vs...)
}

// withKey assigns key to the first node in v.
func withKey(key string, v View) base {
	return func(renv resenv) []node {
		ns := v.resolve(renv)
		if len(ns) > 0 {
			ns = slices.Clone(ns)
			ns[0] = modKey(key)(ns[0])
		}
		return ns
	}
}

func modKey(key string) modifier {
	return modBox(func(b box) box {
		b.node = domi.WithKey(key, b.node)
		return b
	})
}
