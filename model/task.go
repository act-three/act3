package model

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"runtime/debug"
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
	taskIngestEncode              = "ingest-demo"
)

type taskFunc func(*TxR, Context, []string) func(*TxRW) error

var taskTab = map[string]taskFunc{
	taskAddDownloadToTransmission: (*TxR).taskAddDownloadToTransmission,
	taskFetchEpisodes:             (*TxR).taskFetchEpisodes,
	taskIngest:                    (*TxR).taskIngest,
	taskIngestEncode:              (*TxR).taskIngestEncode,
}

func taskError(err error) func(*TxRW) error {
	return func(*TxRW) error { return err }
}

type Task struct {
	t schema.Task
}

func (t *Task) ID() string {
	return t.t.ID
}

func (t *Task) Type() string       { return t.t.Type }
func (t *Task) Args() string       { return t.t.Args }
func (t *Task) Failures() int64    { return t.t.Failures }
func (t *Task) NextRun() time.Time { return time.UnixMilli(t.t.NextRun) }
func (t *Task) FailureDesc() string {
	if s := t.t.FailureDesc; s != nil {
		return *s
	}
	return ""
}

func (tx *TxR) TaskList(ctx Context) ([]*Task, error) {
	list, err := tx.q.TaskList(ctx)
	if err != nil {
		return nil, err
	}
	var tasks []*Task
	for _, t := range list {
		tasks = append(tasks, &Task{t})
	}
	return tasks, nil
}

func (m *Model) RunTaskNow(ctx Context, id string) (err error) {
	defer errorfmt.Handlef("run task %s now: %w", id, &err)
	task, err := schema.New(m.dbr).TaskGet(ctx, id)
	if err != nil {
		return err
	}
	task.NextRun = -1 // Negative means don't reschedule.
	m.start(task)
	return nil
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
	t.onCommit(func() { t.m.start(task) })
	return nil
}

func (m *Model) startTasks(ctx Context, tasks <-chan schema.Task) error {
	err := m.loadTasks(ctx)
	if err != nil {
		return err
	}
	go func() {
		for task := range tasks {
			if d := time.Until(time.UnixMilli(task.NextRun)); d > 0 {
				time.AfterFunc(d, func() { m.start(task) })
			} else {
				m.run(task)
			}
		}
	}()
	return nil
}

func (m *Model) loadTasks(ctx Context) error {
	tasks, err := schema.New(m.dbr).TaskList(ctx)
	if err != nil {
		return err
	}
	m.start(tasks...)
	return nil
}

func (m *Model) start(tasks ...schema.Task) {
	go func() {
		for _, task := range tasks {
			m.tasks <- task
		}
	}()
}

func (m *Model) taskLock(id string) string {
	m.runningTasksMu.Lock()
	defer m.runningTasksMu.Unlock()
	if _, ok := m.runningTasks[id]; ok {
		return ""
	}
	key := cryptorand.Text()
	m.runningTasks[id] = key
	return key
}

func (m *Model) taskUnlock(id, key string) {
	m.runningTasksMu.Lock()
	defer m.runningTasksMu.Unlock()
	if m.runningTasks[id] != key {
		panic("bad lock structure: key mismatch")
	}
	delete(m.runningTasks, id)
}

func (m *Model) run(task schema.Task) {
	ctx := context.Background()
	ctx = logcontext.With(ctx, slog.Group("task", "id", task.ID))
	err := m.run1(ctx, task)
	if err != nil {
		slog.ErrorContext(ctx, "error", "error", err)
		err = m.reschedule(ctx, task, err.Error())
		if err != nil {
			slog.ErrorContext(ctx, "error", "error", err)
		}
	}
}

func (md *Model) run1(ctx Context, task schema.Task) (err error) {
	if key := md.taskLock(task.ID); key == "" {
		time.AfterFunc(time.Minute, func() { md.start(task) })
		return nil
	} else {
		defer md.taskUnlock(task.ID, key)
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

	return md.WithTxR(func(tx *TxR) error {
		task, err = schema.New(md.dbr).TaskGet(ctx, task.ID)
		if err == sql.ErrNoRows {
			return nil
		} else if err != nil {
			return err
		}

		frw := f(tx, ctx, args)
		return md.WithTxRW(func(t *TxRW) error {
			err = frw(t)
			if err != nil {
				return err
			}
			return t.q.TaskDelete(ctx, task.ID)
		})
	})
}

func (m *Model) reschedule(ctx Context, task schema.Task, failure string) error {
	if task.NextRun < 0 {
		// This was a one-off run initiated by the operator.
		// As a special case, don't update the next run time,
		// and don't requeue the task (there's already one in the queue).
		return schema.New(m.dbw).TaskSaveOneOff(ctx, schema.TaskSaveOneOffParams{
			ID:          task.ID,
			Failures:    task.Failures + 1,
			FailureDesc: &failure,
		})
	}
	delay := taskFailDelay(task.Failures)
	delay += time.Duration(rand.Int64N(int64(delay)))
	task, err := schema.New(m.dbw).TaskReschedule(ctx, schema.TaskRescheduleParams{
		ID:          task.ID,
		Failures:    task.Failures + 1,
		NextRun:     time.Now().Add(delay).UnixMilli(),
		FailureDesc: &failure,
	})
	if err != nil {
		return err
	}
	m.start(task)
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
