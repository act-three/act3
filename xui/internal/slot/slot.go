// Package slot provides a container for a value that is consumed at most once.
package slot

// A Slot holds at most one value.
// The value can only be observed by taking it,
// which empties the slot.
//
// The zero Slot is empty.
type Slot[T any] struct {
	value T
	valid bool
}

// Set fills s with v.
// It replaces any value already present.
func (s *Slot[T]) Set(v T) { s.value, s.valid = v, true }

// Take empties s.
// It returns the value s previously held
// and reports whether there was one.
func (s *Slot[T]) Take() (T, bool) {
	v, ok := s.value, s.valid
	*s = Slot[T]{}
	return v, ok
}
