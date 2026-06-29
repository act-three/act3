package progress

import (
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"
)

func TestOpenAndList(t *testing.T) {
	var tr Tracker
	tr.Open("v1", "Encoding", "pass 1")
	items := tr.List("v1")
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].Description() != "Encoding" {
		t.Errorf("desc = %q, want %q", items[0].Description(), "Encoding")
	}
	if items[0].Status() != "pass 1" {
		t.Errorf("status = %q, want %q", items[0].Status(), "pass 1")
	}
	if items[0].Progress() != 0 {
		t.Errorf("progress = %f, want 0", items[0].Progress())
	}
	if items[0].Error() != nil {
		t.Errorf("error = %v, want nil", items[0].Error())
	}
}

func TestUpdate(t *testing.T) {
	var tr Tracker
	tr.Open("v1", "Encoding", "pass 1")
	tr.Update("v1", 0.5)
	items := tr.List("v1")
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].Progress() != 0.5 {
		t.Errorf("progress = %f, want 0.5", items[0].Progress())
	}
}

func TestUpdateStatus(t *testing.T) {
	var tr Tracker
	tr.Open("v1", "Encoding", "pass 1")
	tr.UpdateStatus("v1", "pass 2")
	items := tr.List("v1")
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].Status() != "pass 2" {
		t.Errorf("status = %q, want %q", items[0].Status(), "pass 2")
	}
}

func TestCloseSuccess(t *testing.T) {
	var tr Tracker
	tr.Open("v1", "Encoding", "pass 1")
	tr.Close("v1", nil)
	items := tr.List("v1")
	if len(items) != 0 {
		t.Fatalf("got %d items after close, want 0", len(items))
	}
}

func TestCloseError(t *testing.T) {
	var tr Tracker
	tr.Open("v1", "Encoding", "pass 1")
	e := errors.New("encode failed")
	tr.Close("v1", e)
	items := tr.List("v1")
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1 (error tombstone)", len(items))
	}
	if !errors.Is(items[0].Error(), e) {
		t.Errorf("error = %v, want %v", items[0].Error(), e)
	}
}

func TestUpdateOnClosedIsNoop(t *testing.T) {
	var tr Tracker
	tr.Open("v1", "Encoding", "pass 1")
	tr.Close("v1", errors.New("fail"))
	tr.Update("v1", 0.9)
	tr.UpdateStatus("v1", "should not appear")
	items := tr.List("v1")
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].Progress() != 0 {
		t.Errorf("progress = %f, want 0 (update after close should be noop)", items[0].Progress())
	}
	if items[0].Status() != "pass 1" {
		t.Errorf("status = %q, want %q (update after close should be noop)", items[0].Status(), "pass 1")
	}
}

func TestUpdateOnUnknownKeyIsNoop(t *testing.T) {
	var tr Tracker
	tr.Update("nonexistent", 0.5)
	tr.UpdateStatus("nonexistent", "hello")
	// Should not panic or create items.
	items := tr.List("nonexistent")
	if len(items) != 0 {
		t.Fatalf("got %d items, want 0", len(items))
	}
}

func TestCloseOnUnknownKeyIsNoop(t *testing.T) {
	var tr Tracker
	tr.Close("nonexistent", errors.New("fail"))
	items := tr.List("nonexistent")
	if len(items) != 0 {
		t.Fatalf("got %d items, want 0", len(items))
	}
}

func TestEdges(t *testing.T) {
	var tr Tracker
	tr.AddEdge("ep1", "v1")
	tr.AddEdge("ep1", "v2")
	tr.Open("v1", "Encoding v1", "")
	tr.Open("v2", "Encoding v2", "")

	items := tr.List("ep1")
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
}

func TestEdgeTransitive(t *testing.T) {
	var tr Tracker
	tr.AddEdge("series", "ep1")
	tr.AddEdge("ep1", "v1")
	tr.Open("v1", "Encoding", "")

	items := tr.List("series")
	if len(items) != 1 {
		t.Fatalf("got %d items via transitive edge, want 1", len(items))
	}
	if items[0].Description() != "Encoding" {
		t.Errorf("desc = %q, want %q", items[0].Description(), "Encoding")
	}
}

func TestOpenIdempotent(t *testing.T) {
	var tr Tracker
	tr.Open("v1", "first", "")
	tr.Update("v1", 0.5)
	tr.Open("v1", "second", "")
	items := tr.List("v1")
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	// Second Open should have no effect.
	if items[0].Description() != "first" {
		t.Errorf("desc = %q, want %q (Open should be idempotent)", items[0].Description(), "first")
	}
	if items[0].Progress() != 0.5 {
		t.Errorf("progress = %f, want 0.5 (Open should not reset progress)", items[0].Progress())
	}
}

func TestReopenAfterSuccessClose(t *testing.T) {
	var tr Tracker
	tr.Open("v1", "first", "")
	tr.Close("v1", nil)
	// After successful close, key should be deletable from the map,
	// so a new Open should work.
	tr.Open("v1", "retry", "")
	items := tr.List("v1")
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].Description() != "retry" {
		t.Errorf("desc = %q, want %q", items[0].Description(), "retry")
	}
}

func TestReopenAfterErrorClose(t *testing.T) {
	var tr Tracker
	tr.Open("v1", "first", "")
	tr.Close("v1", errors.New("fail"))
	// Reopening after error close should replace the tombstone.
	tr.Open("v1", "retry", "")
	items := tr.List("v1")
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	// The error tombstone should be replaced by the new item.
	if items[0].Error() != nil {
		t.Errorf("error = %v, want nil (reopen should clear error)", items[0].Error())
	}
	if items[0].Description() != "retry" {
		t.Errorf("desc = %q, want %q", items[0].Description(), "retry")
	}
}

func TestETANoUpdates(t *testing.T) {
	var tr Tracker
	tr.Open("v1", "Encoding", "pass 1")
	items := tr.List("v1")
	if items[0].ETA() != 0 {
		t.Errorf("ETA = %v, want 0 (no updates yet)", items[0].ETA())
	}
}

func TestETACalculation(t *testing.T) {
	// Test ETA directly via Item fields (pure math, no time dependency).
	m := &Item{value: 0.5, rate: 0.1}
	// remaining=0.5, rate=0.1 → ETA = 5s
	got := m.ETA()
	want := 5 * time.Second
	if diff := got - want; diff < -time.Millisecond || diff > time.Millisecond {
		t.Errorf("ETA = %v, want ~%v", got, want)
	}
}

func TestETAComplete(t *testing.T) {
	m := &Item{value: 1.0, rate: 0.1}
	if m.ETA() != 0 {
		t.Errorf("ETA = %v, want 0 (progress complete)", m.ETA())
	}
}

func TestETAZeroRate(t *testing.T) {
	m := &Item{value: 0.5, rate: 0}
	if m.ETA() != 0 {
		t.Errorf("ETA = %v, want 0 (zero rate)", m.ETA())
	}
}

func TestEMARateConverges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var tr Tracker
		tr.Open("v1", "Encoding", "")

		// Constant rate: 10% per second.
		time.Sleep(1 * time.Second)
		tr.Update("v1", 0.1)
		time.Sleep(1 * time.Second)
		tr.Update("v1", 0.2)

		// Rate is steady at 0.1/s → remaining 0.8 → ETA = 8s.
		got := tr.List("v1")[0].ETA()
		want := 8 * time.Second
		if diff := got - want; diff < -time.Millisecond || diff > time.Millisecond {
			t.Errorf("ETA = %v, want ~%v", got, want)
		}

		// Speed doubles to 0.2/s for one interval.
		// EMA = 0.1*0.2 + 0.9*0.1 = 0.11
		// remaining 0.6 / 0.11 ≈ 5.4545s
		time.Sleep(1 * time.Second)
		tr.Update("v1", 0.4)
		got = tr.List("v1")[0].ETA()
		want = 60 * time.Second / 11
		if diff := got - want; diff < -time.Millisecond || diff > time.Millisecond {
			t.Errorf("ETA = %v, want ~%v", got, want)
		}
	})
}

func TestHookOnOpen(t *testing.T) {
	var tr Tracker
	var got []*Item
	tr.SetHook(func(_ string, m *Item) { got = append(got, m) })
	tr.Open("v1", "Encoding", "pass 1")
	if len(got) != 1 {
		t.Fatalf("hook called %d times, want 1", len(got))
	}
	if got[0].Description() != "Encoding" {
		t.Errorf("desc = %q, want %q", got[0].Description(), "Encoding")
	}
}

func TestHookOnUpdate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var tr Tracker
		var got []*Item
		tr.SetHook(func(_ string, m *Item) { got = append(got, m) })
		tr.Open("v1", "Encoding", "")
		time.Sleep(1 * time.Second)
		tr.Update("v1", 0.5)
		if len(got) != 2 {
			t.Fatalf("hook called %d times, want 2", len(got))
		}
		if got[1].Progress() != 0.5 {
			t.Errorf("progress = %f, want 0.5", got[1].Progress())
		}
	})
}

func TestHookOnUpdateStatus(t *testing.T) {
	var tr Tracker
	var got []*Item
	tr.SetHook(func(_ string, m *Item) { got = append(got, m) })
	tr.Open("v1", "Encoding", "pass 1")
	tr.UpdateStatus("v1", "pass 2")
	if len(got) != 2 {
		t.Fatalf("hook called %d times, want 2", len(got))
	}
	if got[1].Status() != "pass 2" {
		t.Errorf("status = %q, want %q", got[1].Status(), "pass 2")
	}
}

func TestHookOnCloseSuccess(t *testing.T) {
	var tr Tracker
	var got []*Item
	tr.SetHook(func(_ string, m *Item) { got = append(got, m) })
	tr.Open("v1", "Encoding", "")
	tr.Close("v1", nil)
	if len(got) != 2 {
		t.Fatalf("hook called %d times, want 2", len(got))
	}
}

func TestHookOnCloseError(t *testing.T) {
	var tr Tracker
	var got []*Item
	tr.SetHook(func(_ string, m *Item) { got = append(got, m) })
	tr.Open("v1", "Encoding", "")
	e := errors.New("fail")
	tr.Close("v1", e)
	if len(got) != 2 {
		t.Fatalf("hook called %d times, want 2", len(got))
	}
	if !errors.Is(got[1].Error(), e) {
		t.Errorf("error = %v, want %v", got[1].Error(), e)
	}
}

func TestHookNotCalledForNoops(t *testing.T) {
	var tr Tracker
	var n int
	tr.SetHook(func(_ string, m *Item) { n++ })
	tr.Update("nonexistent", 0.5)
	tr.UpdateStatus("nonexistent", "x")
	tr.Close("nonexistent", nil)
	if n != 0 {
		t.Errorf("hook called %d times, want 0 (noop operations)", n)
	}
}

func TestAddEdgeUpdatesParents(t *testing.T) {
	var tr Tracker
	tr.Open("v1", "Encoding", "")
	tr.AddEdge("ep1", "v1")
	items := tr.List("ep1")
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	parents := slices.Collect(items[0].Parents())
	if len(parents) != 1 || parents[0] != "ep1" {
		t.Errorf("parents = %v, want [ep1]", parents)
	}
}

func TestAddEdgeTransitiveParents(t *testing.T) {
	var tr Tracker
	tr.Open("v1", "Encoding", "")
	tr.AddEdge("ep1", "v1")
	tr.AddEdge("series", "ep1")
	items := tr.List("series")
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	parents := slices.Collect(items[0].Parents())
	slices.Sort(parents)
	if len(parents) != 2 || parents[0] != "ep1" || parents[1] != "series" {
		t.Errorf("parents = %v, want [ep1 series]", parents)
	}
}

func TestKeyReturnsStringKey(t *testing.T) {
	var tr Tracker
	tr.Open("v1", "Encoding", "")
	items := tr.List("v1")
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].Key() != "v1" {
		t.Errorf("Key() = %v, want %q", items[0].Key(), "v1")
	}
}

func TestOpenAfterAddEdgeSetsParents(t *testing.T) {
	var tr Tracker
	tr.AddEdge("ep1", "v1")
	tr.Open("v1", "Encoding", "")
	items := tr.List("ep1")
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	parents := slices.Collect(items[0].Parents())
	if len(parents) != 1 || parents[0] != "ep1" {
		t.Errorf("parents = %v, want [ep1]", parents)
	}
}

func TestOpenAfterTransitiveAddEdgeSetsParents(t *testing.T) {
	var tr Tracker
	tr.AddEdge("series", "ep1")
	tr.AddEdge("ep1", "v1")
	tr.Open("v1", "Encoding", "")
	items := tr.List("series")
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	parents := slices.Collect(items[0].Parents())
	slices.Sort(parents)
	if len(parents) != 2 || parents[0] != "ep1" || parents[1] != "series" {
		t.Errorf("parents = %v, want [ep1 series]", parents)
	}
}

func TestValidKeyAcceptsAlphanumHyphenUnderscore(t *testing.T) {
	var tr Tracker
	// Should not panic.
	tr.Open("abc-123_XYZ", "desc", "")
	tr.AddEdge("parent-1", "abc-123_XYZ")
}

func TestValidKeyRejectsSlash(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for key with /")
		}
	}()
	var tr Tracker
	tr.Open("video/123", "desc", "")
}

func TestValidKeyRejectsDot(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for key with .")
		}
	}()
	var tr Tracker
	tr.Open("video.123", "desc", "")
}

func TestValidKeyRejectsSpace(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for key with space")
		}
	}()
	var tr Tracker
	tr.Open("video 123", "desc", "")
}

func TestValidKeyRejectsEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for empty key")
		}
	}()
	var tr Tracker
	tr.Open("", "desc", "")
}

func TestValidKeyAddEdgeRejectsInvalidParent(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid parent key")
		}
	}()
	var tr Tracker
	tr.AddEdge("parent/1", "child")
}

func TestValidKeyAddEdgeRejectsInvalidChild(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid child key")
		}
	}()
	var tr Tracker
	tr.AddEdge("parent", "child:1")
}

func TestListSortedByCreationTime(t *testing.T) {
	var tr Tracker
	tr.AddEdge("parent", "a")
	tr.AddEdge("parent", "b")
	tr.AddEdge("parent", "c")
	// Open in order; times should be monotonically increasing.
	tr.Open("a", "first", "")
	tr.Open("b", "second", "")
	tr.Open("c", "third", "")

	items := tr.List("parent")
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}
	for i := 1; i < len(items); i++ {
		if items[i].Opened().Before(items[i-1].Opened()) {
			t.Errorf("items[%d].Opened (%v) before items[%d].Opened (%v)",
				i, items[i].Opened(), i-1, items[i-1].Opened())
		}
	}
}
