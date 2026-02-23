package model

import (
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
	"kr.dev/errorfmt"
)

const (
	taskAddDownloadToTransmission = "add-download-to-transmission"
	taskFetchEpisodes             = "fetch-episodes"
	taskIngest                    = "ingest"
	taskIngestPass1               = "ingest-pass1"
	taskIngestEncodeRend          = "ingest-encode-rend"
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
}

var queueTab = map[string]string{
	taskAddDownloadToTransmission: queueIO,
	taskFetchEpisodes:             queueIO,
	taskIngest:                    queueIO,
	taskIngestPass1:               queueCPU,
	taskIngestEncodeRend:          queueCPU,
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
	in   chan<- schema.Task

	runningMu sync.Mutex
	running   map[string]string
}

func newTaskQueue(name string, m *Model, in chan<- schema.Task) *taskQueue {
	return &taskQueue{
		name:    name,
		m:       m,
		in:      in,
		running: map[string]string{},
	}
}

func (tq *taskQueue) startTasks(ctx Context, n int, tasks <-chan schema.Task) error {
	a, err := schema.New(tq.m.dbr).TaskList(ctx)
	if err != nil {
		return err
	}
	for _, task := range a {
		if queueTab[task.Type] != tq.name {
			continue
		}
		tq.submit(task)
	}
	for range n {
		go tq.runTasks(tasks)
	}
	return nil
}

func (tq *taskQueue) runTasks(tasks <-chan schema.Task) {
	for task := range tasks {
		if d := time.Until(time.UnixMilli(task.NextRun)); d > 0 {
			time.AfterFunc(d, func() { tq.submit(task) })
		} else {
			tq.run(task)
		}
	}
}

func (tq *taskQueue) submit(tasks ...schema.Task) {
	go func() {
		for _, task := range tasks {
			tq.in <- task
		}
	}()
}

func (tq *taskQueue) lock(id string) string {
	tq.runningMu.Lock()
	defer tq.runningMu.Unlock()
	if _, ok := tq.running[id]; ok {
		return ""
	}
	key := cryptorand.Text()
	tq.running[id] = key
	return key
}

func (tq *taskQueue) unlock(id, key string) {
	tq.runningMu.Lock()
	defer tq.runningMu.Unlock()
	if tq.running[id] != key {
		panic("bad lock structure: key mismatch")
	}
	delete(tq.running, id)
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
	if key := tq.lock(task.ID); key == "" {
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
	b, err := json.Marshal(args)
	task, err := t.q.TaskCreate(ctx, schema.TaskCreateParams{
		Type: ttype,
		Args: string(b),
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
