package xiter

// Contains returns whether v is present in s.
func Contains[S ~func(yield func(E) bool), E comparable](s S, v E) bool {
	for vv := range s {
		if v == vv {
			return true
		}
	}
	return false
}
