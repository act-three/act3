package progress

import (
	"errors"
	"testing"
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
	if items[0].Error() != e {
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
