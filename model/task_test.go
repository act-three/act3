package model

import (
	"context"
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

// TestPanicTaskStaysFailed checks that a task whose function panics is
// recorded as permanently failed and not released back to the queue,
// over the full dispatch flow: run records the failure, then the
// goroutine's deferred unlock releases the slot without disturbing it.
func TestPanicTaskStaysFailed(t *testing.T) {
	const taskPanic = "test-panic"
	taskTab[taskPanic] = func(*TxR, []string) error { panic("boom") }
	queueTab[taskPanic] = queueNet
	t.Cleanup(func() {
		delete(taskTab, taskPanic)
		delete(queueTab, taskPanic)
	})

	m := newTestModel(t)
	ctx := t.Context()
	tq := newTaskQueue(queueNet, 1, m)

	var task schema.Task
	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		var err error
		task, err = tx.q.TaskCreate(schema.TaskCreateParams{
			Type:     taskPanic,
			Args:     "[]",
			Priority: priority.FetchPoster,
			Queue:    queueNet,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	// Mirror dispatch's goroutine body: lock, run, then unlock via
	// the deferred release.
	_, cancel := context.WithCancel(ctx)
	key, err := tq.lock(task.ID, task.Type, task.Args, cancel)
	if err != nil {
		t.Fatal(err)
	}
	if key == "" {
		t.Fatal("lock returned empty key")
	}
	func() {
		defer tq.unlock(task.ID, key)
		tq.run(ctx, task)
	}()

	if err := m.WithTxR(ctx, func(tx *TxR) error {
		var err error
		task, err = tx.q.TaskGet(task.ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if task.State != "failed" {
		t.Errorf("State = %q, want %q", task.State, "failed")
	}
	if task.FailureDesc == nil {
		t.Error("FailureDesc = nil, want recorded panic")
	}
}

// TestKilledTaskRecordsFailed checks that a task killed mid-run still
// records its terminal 'failed' state. A kill cancels the run ctx, so
// run must persist the failure with cancellation detached; otherwise
// the write fails on the canceled ctx and the task is stranded in
// 'running'.
func TestKilledTaskRecordsFailed(t *testing.T) {
	m := newTestModel(t)
	tq := newTaskQueue(queueNet, 1, m)

	var task schema.Task
	if err := m.WithTxRW(t.Context(), func(tx *TxRW) error {
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

	// Mirror dispatch's goroutine body, but with an already-canceled
	// ctx, as after tq.kill canceled the running task's context.
	ctx, cancel := context.WithCancel(t.Context())
	key, err := tq.lock(task.ID, task.Type, task.Args, cancel)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	func() {
		defer tq.unlock(task.ID, key)
		tq.run(ctx, task)
	}()

	if err := m.WithTxR(t.Context(), func(tx *TxR) error {
		var err error
		task, err = tx.q.TaskGet(task.ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if task.State != "failed" {
		t.Errorf("State = %q, want %q", task.State, "failed")
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
