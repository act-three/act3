package model

import "iter"

// Event reports that model state changed.
// Emitters include the database (after each read-write transaction),
// the upload tracker (periodically as bytes are received),
// and the task queue (when a task begins or ends).
//
// Details contains additional information a client might need.
type Event struct {
	Details []Detail
}

// Detail is extra information about an Event
// that a client might find useful.
type Detail struct {
	// SlugChangeID, when set, is the ID of an object whose slug
	// changed, so a client viewing that object can follow the rename.
	SlugChangeID string
}

// emit broadcasts an Event carrying details (which may be nil) to every
// subscriber, dropping it for any subscriber not keeping up.
func (m *Model) emit(details []Detail) {
	ev := &Event{Details: details}
	m.subMu.Lock()
	defer m.subMu.Unlock()
	for ch := range m.sub {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (m *Model) Events(ctx Context) iter.Seq[*Event] {
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
