package web

import (
	"path"
	"strings"
)

type matcher struct {
	path []string
	tab  map[string]string
}

func (m *matcher) match(pattern string) bool {
	parts := splitPath(pattern)
	if len(m.path) != len(parts) {
		return false
	}
	tab := map[string]string{}
	for i, part := range parts {
		part, foundPre := strings.CutPrefix(part, "{")
		part, foundSuf := strings.CutSuffix(part, "}")
		if foundPre && foundSuf {
			tab[part] = m.path[i]
		} else if part != m.path[i] {
			return false
		}
	}
	m.tab = tab
	return true
}

func (m *matcher) get(name string) string {
	return m.tab[name]
}

// splitPath splits path into a list of segments.
// if path is "/" or "", splitPath returns an empty slice.
func splitPath(p string) []string {
	a := strings.Split(strings.Trim(path.Join("/", p), "/"), "/")
	if len(a) == 1 && a[0] == "" {
		a = nil
	}
	return a
}
