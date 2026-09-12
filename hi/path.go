package hi

import (
	"path"
	"slices"
	"strings"
)

// Path displays v
// if path is the current request path.
// Otherwise, the view is empty.
//
// The path is cleaned before matching.
func Path(path string, v View) View {
	return PathReader(func(rp RequestPath) View {
		if rp.Equal(path) {
			return v
		}
		return Empty()
	})
}

// PathPrefix displays v
// if prefix is a prefix of the current request path.
// Otherwise, the view is empty.
//
// The prefix is cleaned before matching.
// It matches complete path segments.
// The prefix "/app" does not match a request for "/apple".
func PathPrefix(prefix string, v View) View {
	return PathReader(func(rp RequestPath) View {
		if rp.HasPrefix(prefix) {
			return v
		}
		return Empty()
	})
}

// PathReader calls f with the current request path
// and displays its result.
func PathReader(f func(RequestPath) View) View {
	return base(func(renv resenv) []node {
		return f(slices.Clone(renv.path)).resolve(renv)
	})
}

// A RequestPath is a list of HTTP request path segments.
// See [PathReader].
//
// A request for "/settings/profile"
// produces RequestPath{"settings", "profile"}.
// A request for "/" produces an empty RequestPath.
type RequestPath []string

// Equal reports whether path is the same path as p.
//
// The path is cleaned before matching.
func (p RequestPath) Equal(path string) bool {
	return slices.Equal(p, splitPath(path))
}

// HasPrefix reports whether prefix is a prefix of p.
//
// The prefix is cleaned before matching.
// It matches complete path segments.
// The prefix "/app" does not match a request for "/apple".
func (p RequestPath) HasPrefix(prefix string) bool {
	parts := splitPath(prefix)
	return len(p) >= len(parts) && slices.Equal(parts, p[:len(parts)])
}

func splitPath(p string) []string {
	p = strings.TrimPrefix(path.Clean("/"+p), "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}
