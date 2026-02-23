package xheap

import (
	"cmp"
	"testing"
)

func TestMinHeap(t *testing.T) {
	h := NewOrdered[int]()
	for _, v := range []int{5, 3, 7, 1, 4, 2, 6} {
		h.Push(v)
	}
	if h.Len() != 7 {
		t.Fatalf("Len = %d, want 7", h.Len())
	}
	for want := 1; want <= 7; want++ {
		got := h.Pop()
		if got != want {
			t.Errorf("Pop = %d, want %d", got, want)
		}
	}
	if h.Len() != 0 {
		t.Fatalf("Len = %d, want 0", h.Len())
	}
}

func TestCustomCompare(t *testing.T) {
	type item struct {
		pri  int
		name string
	}
	h := New(func(a, b item) int {
		return cmp.Or(
			cmp.Compare(a.pri, b.pri),
			cmp.Compare(a.name, b.name),
		)
	})
	h.Push(item{2, "b"})
	h.Push(item{1, "c"})
	h.Push(item{1, "a"})
	h.Push(item{3, "d"})

	got := h.Pop()
	if got.pri != 1 || got.name != "a" {
		t.Errorf("first = %v, want {1, a}", got)
	}
	got = h.Pop()
	if got.pri != 1 || got.name != "c" {
		t.Errorf("second = %v, want {1, c}", got)
	}
	got = h.Pop()
	if got.pri != 2 || got.name != "b" {
		t.Errorf("third = %v, want {2, b}", got)
	}
	got = h.Pop()
	if got.pri != 3 || got.name != "d" {
		t.Errorf("fourth = %v, want {3, d}", got)
	}
}
