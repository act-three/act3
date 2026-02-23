// Package xheap implements a generic min-heap.
package xheap

import "cmp"

// Heap is a min-heap of elements ordered by a comparison function.
type Heap[E any] struct {
	s    []E
	less func(a, b E) int
}

// New returns an empty heap that orders elements using the given
// comparison function. The function should return a negative value
// when a < b, zero when a == b, and a positive value when a > b
// (like [cmp.Compare]).
func New[E any](less func(a, b E) int) *Heap[E] {
	return &Heap[E]{less: less}
}

// NewOrdered returns an empty heap for ordered types, using the
// natural ordering.
func NewOrdered[E cmp.Ordered]() *Heap[E] {
	return &Heap[E]{less: cmp.Compare[E]}
}

// Len returns the number of elements in the heap.
func (h *Heap[E]) Len() int { return len(h.s) }

// Push adds an element to the heap.
func (h *Heap[E]) Push(e E) {
	h.s = append(h.s, e)
	h.up(len(h.s) - 1)
}

// Pop removes and returns the minimum element.
// It panics if the heap is empty.
func (h *Heap[E]) Pop() E {
	n := len(h.s) - 1
	h.s[0], h.s[n] = h.s[n], h.s[0]
	e := h.s[n]
	h.s = h.s[:n]
	if n > 0 {
		h.down(0)
	}
	return e
}

func (h *Heap[E]) up(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if h.less(h.s[i], h.s[parent]) >= 0 {
			break
		}
		h.s[i], h.s[parent] = h.s[parent], h.s[i]
		i = parent
	}
}

func (h *Heap[E]) down(i int) {
	n := len(h.s)
	for {
		left := 2*i + 1
		if left >= n {
			break
		}
		j := left
		if right := left + 1; right < n && h.less(h.s[right], h.s[left]) < 0 {
			j = right
		}
		if h.less(h.s[j], h.s[i]) >= 0 {
			break
		}
		h.s[i], h.s[j] = h.s[j], h.s[i]
		i = j
	}
}
