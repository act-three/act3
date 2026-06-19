package model

import (
	"testing"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/priority"
)

func TestTaskFailDelay(t *testing.T) {
	for _, tt := range []struct {
		n int64
		d string
	}{
		{0, "100ms"},
		{1, "200ms"},
		{2, "400ms"},
		{5, "3.2s"},
		{10, "1m42.4s"},
		{15, "54m36.8s"},
		{16, "1h49m13.6s"},
		{17, "2h0m0s"},
		{18, "2h0m0s"},
	} {
		d := taskFailDelay(tt.n)
		got := d.String()
		if got != tt.d {
			t.Errorf("taskFailDelay(%d) = %v, want %v", tt.n, got, tt.d)
		}
	}
}

// TestRescheduleLowersPriority checks that each failed retry drops the
// task one priority level, so a repeatedly-failing task interleaves
// once per level instead of monopolizing its own.
func TestRescheduleLowersPriority(t *testing.T) {
	m := newTestModel(t)
	ctx := t.Context()
	tq := newTaskQueue(queueNet, 1, m)

	var task schema.Task
	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		var err error
		task, err = tx.q.TaskCreate(schema.TaskCreateParams{
			Type:     taskFetchSeriesPoster,
			Args:     "[]",
			Priority: priority.FetchPoster,
			Queue:    queueNet,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	for n := int64(1); n <= 3; n++ {
		if err := tq.reschedule(ctx, task, "boom"); err != nil {
			t.Fatal(err)
		}
		if err := m.WithTxR(ctx, func(tx *TxR) error {
			var err error
			task, err = tx.q.TaskGet(task.ID)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		if want := int64(priority.FetchPoster) + n; task.Priority != want {
			t.Errorf("after %d failures: Priority = %d, want %d", n, task.Priority, want)
		}
		if task.Failures != n {
			t.Errorf("after %d failures: Failures = %d, want %d", n, task.Failures, n)
		}
	}
}
