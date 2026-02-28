package model

import (
	"cmp"
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"math/rand/v2"
	"runtime/debug"
	"sync"
	"time"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/log/logcontext"
	"ily.dev/act3/tlog"
	"ily.dev/act3/xheap"
	"kr.dev/errorfmt"
)

const (
	taskAddDownloadToTransmission = "add-download-to-transmission"
	taskFetchEpisodes             = "fetch-episodes"
	taskIngest                    = "ingest"
	taskIngestPass1               = "ingest-pass1"
	taskIngestEncodeRend          = "ingest-encode-rend"
	taskReingest                  = "reingest"
)

const (
	queueIO  = "io"
	queueCPU = "cpu"
)

type taskFunc func(*TxR, Context, []string) error

var taskTab = map[string]taskFunc{
	taskAddDownloadToTransmission: (*TxR).taskAddDownloadToTransmission,
	taskFetchEpisodes:             (*TxR).taskFetchEpisodes,
	taskIngest:                    (*TxR).taskIngest,
	taskIngestPass1:               (*TxR).taskIngestPass1,
	taskIngestEncodeRend:          (*TxR).taskIngestEncodeRend,
	taskReingest:                  (*TxR).taskReingest,
}

var queueTab = map[string]string{
	taskAddDownloadToTransmission: queueIO,
	taskFetchEpisodes:             queueIO,
	taskIngest:                    queueIO,
	taskIngestPass1:               queueCPU,
	taskIngestEncodeRend:          queueCPU,
	taskReingest:                  queueIO,
}

var taskQueues = map[string]int{
	queueIO:  2,
	queueCPU: 1,
}

type Task struct {
	t         schema.Task
	isRunning bool
}

func (t *Task) ID() string {
	return t.t.ID
}

func (t *Task) Type() string       { return t.t.Type }
func (t *Task) Args() string       { return t.t.Args }
func (t *Task) Failures() int64    { return t.t.Failures }
func (t *Task) IsRunning() bool    { return t.isRunning }
func (t *Task) NextRun() time.Time { return time.UnixMilli(t.t.NextRun) }
func (t *Task) FailureDesc() string {
	if s := t.t.FailureDesc; s != nil {
		return *s
	}
	return ""
}

type taskQueue struct {
	name string
	m    *Model

	mu     sync.Mutex
	heap   *xheap.Heap[schema.Task]
	signal chan struct{} // capacity 1; wakes a worker

	runningMu sync.Mutex
	running   map[string]runEntry
}

type runEntry struct {
	key    string
	cancel context.CancelFunc
}

func newTaskQueue(name string, m *Model) *taskQueue {
	return &taskQueue{
		name: name,
		m:    m,
		heap: xheap.New(func(a, b schema.Task) int {
			return cmp.Or(
				cmp.Compare(a.Priority, b.Priority),
				cmp.Compare(a.ID, b.ID),
			)
		}),
		signal:  make(chan struct{}, 1),
		running: map[string]runEntry{},
	}
}

func (tq *taskQueue) startTasks(ctx Context, n int) error {
	a, err := schema.New(tq.m.dbr).TaskList(ctx)
	if err != nil {
		return err
	}
	for _, task := range a {
		if queueTab[task.Type] != tq.name {
			continue
		}
		tq.push(task)
	}
	tq.wake()
	for range n {
		go tq.runTasks()
	}
	return nil
}

func (tq *taskQueue) runTasks() {
	for range tq.signal {
		for {
			tq.mu.Lock()
			if tq.heap.Len() == 0 {
				tq.mu.Unlock()
				break
			}
			task := tq.heap.Pop()
			more := tq.heap.Len() > 0
			tq.mu.Unlock()
			if more {
				tq.wake()
			}
			if d := time.Until(time.UnixMilli(task.NextRun)); d > 0 {
				time.AfterFunc(d, func() { tq.submit(task) })
			} else {
				tq.run(task)
			}
		}
	}
}

// push adds a task to the heap under the lock.
func (tq *taskQueue) push(task schema.Task) {
	tq.mu.Lock()
	tq.heap.Push(task)
	tq.mu.Unlock()
}

// wake sends a non-blocking signal to unblock one worker.
func (tq *taskQueue) wake() {
	select {
	case tq.signal <- struct{}{}:
	default:
	}
}

func (tq *taskQueue) submit(tasks ...schema.Task) {
	for _, task := range tasks {
		tq.push(task)
	}
	tq.wake()
}

func (tq *taskQueue) lock(id string, cancel context.CancelFunc) string {
	tq.runningMu.Lock()
	defer tq.runningMu.Unlock()
	if _, ok := tq.running[id]; ok {
		return ""
	}
	key := cryptorand.Text()
	tq.running[id] = runEntry{key: key, cancel: cancel}
	return key
}

func (tq *taskQueue) unlock(id, key string) {
	tq.runningMu.Lock()
	defer tq.runningMu.Unlock()
	e := tq.running[id]
	if e.key != key {
		panic("bad lock structure: key mismatch")
	}
	e.cancel()
	delete(tq.running, id)
}

func (tq *taskQueue) kill(id string) bool {
	tq.runningMu.Lock()
	defer tq.runningMu.Unlock()
	e, ok := tq.running[id]
	if ok {
		e.cancel()
	}
	return ok
}

func (tq *taskQueue) runningSet() map[string]bool {
	tq.runningMu.Lock()
	defer tq.runningMu.Unlock()
	m := map[string]bool{}
	for k := range tq.running {
		m[k] = true
	}
	return m
}

func (tq *taskQueue) run(task schema.Task) {
	ctx := context.Background()
	ctx = logcontext.With(ctx, slog.Group("task", "id", task.ID))
	err := tq.run1(ctx, task)
	if err != nil {
		slog.ErrorContext(ctx, "error", "error", err)
		err = tq.reschedule(ctx, task, err.Error())
		if err != nil {
			slog.ErrorContext(ctx, "error", "error", err)
		}
	}
}

func (tq *taskQueue) run1(ctx Context, task schema.Task) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if key := tq.lock(task.ID, cancel); key == "" {
		time.AfterFunc(time.Minute, func() { tq.submit(task) })
		return nil
	} else {
		defer tq.unlock(task.ID, key)
	}

	defer tlog.Elapsed(ctx, "task", "type", task.Type, "args", task.Args)()

	defer func() {
		r := recover()
		if e1, ok := r.(error); ok {
			slog.ErrorContext(ctx, "panic", "error", e1)
			debug.PrintStack()
			err = nil
		} else if r != nil {
			slog.ErrorContext(ctx, "panic", "value", r)
			debug.PrintStack()
			err = nil
		}
	}()

	f := taskTab[task.Type]
	if f == nil {
		return errors.New("unknown task type: " + task.Type)
	}
	var args []string
	err = json.Unmarshal([]byte(task.Args), &args)
	if err != nil {
		return fmt.Errorf("task %s %s: bad args: %w", task.Type, task.ID, err)
	}

	return tq.m.WithTxR(func(tx *TxR) error {
		task, err = tx.q.TaskGet(ctx, task.ID)
		if err == sql.ErrNoRows {
			return nil
		} else if err != nil {
			return err
		}

		err = f(tx, ctx, args)
		if err != nil {
			return err
		}
		return tq.m.WithTxRW(func(t *TxRW) error {
			return t.q.TaskDelete(ctx, task.ID)
		})
	})
}

func (tq *taskQueue) reschedule(ctx Context, task schema.Task, failure string) error {
	if task.NextRun < 0 {
		// This was a one-off run initiated by the operator.
		// As a special case, don't update the next run time,
		// and don't requeue the task (there's already one in the queue).
		return schema.New(tq.m.dbw).TaskSaveOneOff(ctx, schema.TaskSaveOneOffParams{
			ID:          task.ID,
			Failures:    task.Failures + 1,
			FailureDesc: &failure,
		})
	}
	delay := taskFailDelay(task.Failures)
	delay += time.Duration(rand.Int64N(int64(delay)))
	task, err := schema.New(tq.m.dbw).TaskReschedule(ctx, schema.TaskRescheduleParams{
		ID:          task.ID,
		Failures:    task.Failures + 1,
		NextRun:     time.Now().Add(delay).UnixMilli(),
		FailureDesc: &failure,
	})
	if err != nil {
		return err
	}
	tq.submit(task)
	return nil
}

func (m *Model) RunTaskNow(ctx Context, id string) (err error) {
	defer errorfmt.Handlef("run task %s now: %w", id, &err)
	task, err := schema.New(m.dbr).TaskGet(ctx, id)
	if err != nil {
		return err
	}
	task.NextRun = -1 // Negative means don't reschedule.
	return m.submit(task)
}

func (m *Model) KillTask(id string) bool {
	for _, tq := range m.tasks {
		if tq.kill(id) {
			return true
		}
	}
	return false
}

func (m *Model) submit(task schema.Task) (err error) {
	tq := m.tasks[queueTab[task.Type]]
	if tq == nil {
		return errors.New("unknown task type: " + task.Type)
	}
	tq.submit(task)
	return nil
}

func (tx *TxR) TaskList(ctx Context) ([]*Task, error) {
	list, err := tx.q.TaskList(ctx)
	if err != nil {
		return nil, err
	}
	isRunning := map[string]bool{}
	for _, tq := range tx.m.tasks {
		maps.Copy(isRunning, tq.runningSet())
	}
	var tasks []*Task
	for _, t := range list {
		task := &Task{t, isRunning[t.ID]}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (tx *TxRW) TaskDelete(ctx Context, id string) error {
	return tx.q.TaskDelete(ctx, id)
}

func (t *TxRW) addTask(ctx Context, ttype string, args ...string) error {
	return t.addTaskWithPriority(ctx, 0, ttype, args...)
}

func (t *TxRW) addTaskWithPriority(ctx Context, priority int64, ttype string, args ...string) error {
	b, err := json.Marshal(args)
	task, err := t.q.TaskCreate(ctx, schema.TaskCreateParams{
		Type:     ttype,
		Args:     string(b),
		Priority: priority,
	})
	if err != nil {
		return err
	}
	t.onCommit(func() {
		err := t.m.submit(task)
		if err != nil {
			slog.ErrorContext(ctx, "submit-task-error",
				"task-id", task.ID,
				"error", err.Error(),
			)
		}
	})
	return nil
}

func taskFailDelay(n int64) time.Duration {
	const maxShift = 20
	if n > maxShift {
		n = maxShift
	}
	base := 100 * time.Millisecond * (1 << n)
	const maxDelay = 2 * time.Hour
	if base > maxDelay {
		base = maxDelay
	}
	return base
}

