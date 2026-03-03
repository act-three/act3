// Package progress contains helper types to keep track of progress
// through sets of work items.
package progress

import (
	"iter"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"
)

// validKey panics if key contains characters that are unsafe
// for use in CSS class names. Keys are used as CSS selectors
// for Turbo Stream targeting, so they must only contain
// letters, digits, hyphens, and underscores.
func validKey(key string) {
	if key == "" {
		panic("progress: empty key")
	}
	if i := strings.IndexFunc(key, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' ||
			r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' ||
			r == '-' || r == '_')
	}); i >= 0 {
		panic("progress: invalid character " +
			string(rune(key[i])) + " in key " + key)
	}
}

// emaAlpha controls how much weight is given to the most recent
// sample when computing the exponential moving average of the
// progress rate. A value of 0.2 gives an effective window of
// roughly 5 samples.
const emaAlpha = 0.2

const (
	EventOpen   = "progress-open"
	EventUpdate = "progress-update"
	EventClose  = "progress-close"
)

// Item represents progress through a work item.
// Its methods are safe to call concurrently.
type Item struct {
	// Item values are not synchronized, so keep them immutable.
	key     string
	parents map[string]bool
	opened  time.Time
	desc    string
	status  string
	value   float64
	err     error
	closed  bool
	rate    float64   // EMA-smoothed progress per second
	updated time.Time // time of last Update call
}

// Key returns the unique key for m.
func (m *Item) Key() string { return m.key }

// Parents yields the keys from which m is reachable.
func (m *Item) Parents() iter.Seq[string] { return maps.Keys(m.parents) }

// Opened returns the time m was opened.
func (m *Item) Opened() time.Time { return m.opened }

// Description returns the description given to Open.
// It is suitable for display to the user.
func (m *Item) Description() string { return m.desc }

// Status returns the most recent status given to Open or UpdateStatus.
// It is suitable for display to the user.
func (m *Item) Status() string { return m.status }

// Progress returns m's current progress in the range [0,1].
func (m *Item) Progress() float64 { return m.value }

// Error returns the error, if any, given to Close.
func (m *Item) Error() error { return m.err }

// ETA returns the estimated time remaining based on the
// exponentially weighted moving average of the progress rate.
// It returns 0 if no estimate is available.
func (m *Item) ETA() time.Duration {
	if m.rate <= 0 {
		return 0
	}
	remaining := 1.0 - m.value
	if remaining <= 0 {
		return 0
	}
	return time.Duration(remaining / m.rate * float64(time.Second))
}

type hookCall struct {
	hook func(string, *Item)
	Item *Item
}

func (h *hookCall) fire(event string) {
	if h.hook != nil && h.Item != nil {
		h.hook(event, h.Item)
	}
}

// A Tracker keeps track of progress through a set of work items.
// Clients open items, update progress as work is done,
// and close them when finished.
// Items can be arranged in a tree by adding parent-child associations.
//
// A Tracker's methods are safe to call concurrently.
type Tracker struct {
	mu   sync.Mutex
	hook func(string, *Item)
	item map[string]*Item           // missing entry is implicitly "closed"
	edge map[string]map[string]bool // parent -> child edges
	back map[string]map[string]bool // child -> parent edges
	// Note that we never delete edges. They only accumulate over time.
	// If it ever becomes a problem, we can figure something out,
	// but I doubt it will.
}

// SetHook arranges for t to call f upon each update of each Item.
// Parameter event is one of the Event* constants.
func (t *Tracker) SetHook(f func(event string, m *Item)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.hook = f
}

// AddEdge associates child with parent,
// so that t.List(parent) returns all of child's items
// (including child itself, if it is an item key).
//
// Parent does not have to be a key opened by Open;
// likewise, child does not have to be a key opened by Open.
// However, adding an edge is useful
// only if a key opened by Open can eventually be reached
// by following edges.
func (t *Tracker) AddEdge(parent, child string) {
	validKey(parent)
	validKey(child)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.edge == nil {
		t.edge = map[string]map[string]bool{}
	}
	if t.edge[parent] == nil {
		t.edge[parent] = map[string]bool{}
	}
	t.edge[parent][child] = true
	if t.back == nil {
		t.back = map[string]map[string]bool{}
	}
	if t.back[child] == nil {
		t.back[child] = map[string]bool{}
	}
	t.back[child][parent] = true
	var a []*Item
	t.list(&a, parent)
	for _, m := range a {
		mm := &Item{}
		*mm = *m
		mm.parents = map[string]bool{}
		maps.Copy(mm.parents, m.parents)
		mm.parents[parent] = true
		t.item[mm.key] = mm
	}
}

// Open opens key for updates,
// initializing it with desc, an initial status, and the current time.
// The given desc should be a human-readable description,
// sufficiently specific to distinguish this item from others in UI.
//
// Callers must call AddEdge for all parents before Open to ensure
// that the UI updates correctly.
// An event hook fires during Open, and the progress item contains
// a snapshot of the parents that exist at that time.
//
// If key is already open, Open has no effect.
func (t *Tracker) Open(key, desc, status string) {
	validKey(key)
	t.mu.Lock()
	notify := hookCall{hook: t.hook}
	defer notify.fire(EventOpen)
	defer t.mu.Unlock()
	if t.item == nil {
		t.item = map[string]*Item{}
	}
	if m, ok := t.item[key]; ok && !m.closed {
		return
	}
	now := time.Now()
	m := &Item{
		key:     key,
		parents: map[string]bool{},
		opened:  now,
		updated: now,
		desc:    desc,
		status:  status,
	}
	t.parents(m.parents, key)
	t.item[key] = m
	notify.Item = m
}

// Update updates the progress value for key and recomputes the
// EMA-smoothed progress rate used by [Item.ETA].
// If key is not open, Update has no effect.
func (t *Tracker) Update(key string, value float64) {
	t.mu.Lock()
	notify := hookCall{hook: t.hook}
	defer notify.fire(EventUpdate)
	defer t.mu.Unlock()
	m := t.item[key]
	if m == nil || m.closed {
		return
	}
	mm := &Item{}
	*mm = *m
	mm.value = value

	now := time.Now()
	var instant float64
	if dt := now.Sub(m.updated).Seconds(); dt > 0 {
		instant = (value - m.value) / dt
	}
	if m.updated.Equal(m.opened) {
		mm.rate = instant
	} else {
		mm.rate = emaAlpha*instant + (1-emaAlpha)*m.rate
	}
	mm.updated = now

	t.item[key] = mm
	notify.Item = mm
}

// Update updates the status for key.
// If key is not open, UpdateStatus has no effect.
func (t *Tracker) UpdateStatus(key, status string) {
	t.mu.Lock()
	notify := hookCall{hook: t.hook}
	defer notify.fire(EventUpdate)
	defer t.mu.Unlock()
	m := t.item[key]
	if m == nil || m.closed {
		return
	}
	mm := &Item{}
	*mm = *m
	mm.status = status
	t.item[key] = mm
	notify.Item = mm
}

// Close closes key, and stores err, which may be nil, as its error.
// If key is already closed, Close has no effect.
func (t *Tracker) Close(key string, err error) {
	t.mu.Lock()
	notify := hookCall{hook: t.hook}
	defer notify.fire(EventClose)
	defer t.mu.Unlock()
	if m, ok := t.item[key]; !ok {
		return
	} else if err != nil {
		// Leave tombstone if there's an error.
		mm := &Item{}
		*mm = *m
		mm.err = err
		mm.closed = true
		t.item[key] = mm
		notify.Item = mm
		return
	}
	delete(t.item, key)
	notify.Item = &Item{
		key:    key,
		closed: true,
	}
}

// List returns all items associated with key via Open or AddEdge.
// The returned slice is sorted by creation time.
func (t *Tracker) List(key string) []*Item {
	t.mu.Lock()
	defer t.mu.Unlock()
	var a []*Item
	t.list(&a, key)
	slices.SortFunc(a, func(a, b *Item) int {
		return a.opened.Compare(b.opened)
	})
	return a
}

// list collects items reachable from key. Must be called with t.mu held.
func (t *Tracker) list(a *[]*Item, key string) {
	if it := t.item[key]; it != nil {
		*a = append(*a, it)
	}
	for child := range t.edge[key] {
		t.list(a, child)
	}
}

// parents collects parents reachable from key. Must be called with t.mu held.
func (t *Tracker) parents(m map[string]bool, key string) {
	for p := range t.back[key] {
		m[p] = true
		t.parents(m, p)
	}
}
