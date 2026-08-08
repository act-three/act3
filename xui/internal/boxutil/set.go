package boxutil

import (
	"maps"
	"slices"

	"ily.dev/domi"
	"ily.dev/domi/attr"
)

type ClassSet struct {
	m map[any]string
}

func (s *ClassSet) Set[K comparable](key K, value string) {
	m := maps.Clone(s.m)
	if m == nil {
		m = make(map[any]string, 1)
	}
	m[key] = value
	s.m = m
}

func (s *ClassSet) Attr() domi.Attr {
	if len(s.m) == 0 {
		return nil
	}
	return attr.Class(slices.Sorted(maps.Values(s.m))...)
}
