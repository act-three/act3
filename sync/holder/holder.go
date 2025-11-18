package holder

import "sync"

type Holder[T any] struct {
	mu  sync.Mutex
	v   T
	err error
}

func New[T any](v T, err error) *Holder[T] {
	return &Holder[T]{v: v, err: err}
}

func (h *Holder[T]) Set(v T, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.v, h.err = v, err
}

func (h *Holder[T]) Get() (T, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.v, h.err
}
