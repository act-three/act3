package model

import (
	"context"
	"iter"
)

// An Event reports that model state has changed.
// Emitters include the database (after each read-write transaction),
// the upload tracker (periodically as bytes are received),
// and the task queue (when a task begins or ends).
type Event struct{}

// emit broadcasts an Event to every subscriber, dropping it for any
// subscriber not keeping up.
func (m *Model) emit() {
	ev := &Event{}
	m.subMu.Lock()
	defer m.subMu.Unlock()
	for ch := range m.sub {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (m *Model) Events(ctx context.Context) iter.Seq[*Event] {
	ch := make(chan *Event, 64)
	m.subMu.Lock()
	m.sub[ch] = struct{}{}
	m.subMu.Unlock()
	return func(yield func(*Event) bool) {
		defer func() {
			m.subMu.Lock()
			delete(m.sub, ch)
			m.subMu.Unlock()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-ch:
				if !yield(ev) {
					return
				}
			}
		}
	}
}
