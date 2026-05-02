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
	"sync"
	"time"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/log/logcontext"
	"ily.dev/act3/priority"
	"ily.dev/act3/tlog"
	"kr.dev/errorfmt"
)

const (
	taskAddDownloadToTransmission = "add-download-to-transmission"
	taskFetchEpisodes             = "fetch-episodes"
	taskFetchEpisodeThumbnail     = "fetch-episode-thumbnail"
	taskFetchSeriesPoster         = "fetch-series-poster"
	taskFetchMoviePoster          = "fetch-movie-poster"
	taskIngest                    = "ingest"
	taskIngestEncodeAudio         = "ingest-encode-audio"
	taskIngestEncodeDownloadRend  = "ingest-encode-dl-rend"
	taskIngestEncodeRend          = "ingest-encode-rend"
	taskIngestExtractSubs         = "ingest-extract-subs"
	taskIngestPass1               = "ingest-pass1"
	taskReimport                  = "reimport"
	taskReencode                  = "reencode"
)

const (
	queueIO  = "io"
	queueCPU = "cpu"
)

type taskFunc func(*TxR, Context, []string) error

var taskTab = map[string]taskFunc{
	taskAddDownloadToTransmission: (*TxR).taskAddDownloadToTransmission,
	taskFetchEpisodes:             (*TxR).taskFetchEpisodes,
	taskFetchEpisodeThumbnail:     (*TxR).taskFetchEpisodeThumbnail,
	taskFetchSeriesPoster:         (*TxR).taskFetchSeriesPoster,
	taskFetchMoviePoster:          (*TxR).taskFetchMoviePoster,
	taskIngest:                    (*TxR).taskIngest,
	taskIngestEncodeAudio:         (*TxR).taskIngestEncodeAudio,
	taskIngestEncodeDownloadRend:  (*TxR).taskIngestEncodeDownloadRend,
	taskIngestEncodeRend:          (*TxR).taskIngestEncodeRend,
	taskIngestExtractSubs:         (*TxR).taskIngestExtractSubs,
	taskIngestPass1:               (*TxR).taskIngestPass1,
	taskReimport:                  (*TxR).taskReimport,
	taskReencode:                  (*TxR).taskReencode,
}

var queueTab = map[string]string{
	taskAddDownloadToTransmission: queueIO,
	taskFetchEpisodes:             queueIO,
	taskFetchEpisodeThumbnail:     queueIO,
	taskFetchSeriesPoster:         queueIO,
	taskFetchMoviePoster:          queueIO,
	taskIngest:                    queueIO,
	taskIngestEncodeAudio:         queueCPU,
	taskIngestEncodeDownloadRend:  queueCPU,
	taskIngestEncodeRend:          queueCPU,
	taskIngestExtractSubs:         queueIO,
	taskIngestPass1:               queueCPU,
	taskReimport:                  queueIO,
	taskReencode:                  queueIO,
}

var taskQueues = map[string]int{
	queueIO:  2,
	queueCPU: 1,
}

// ErrPermanent marks a task error as not retriable.
// Task functions wrap an error with [Permanent] to signal this;
// the task framework records the failure and stops scheduling retries.
var ErrPermanent = errors.New("permanent failure")

// Permanent wraps err so the task framework treats it as a permanent
// failure rather than rescheduling for retry.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrPermanent, err)
}

type Task struct {
	t schema.Task
}

func (t *Task) ID() string         { return t.t.ID }
func (t *Task) Type() string       { return t.t.Type }
func (t *Task) Args() string       { return t.t.Args }
func (t *Task) Failures() int64    { return t.t.Failures }
func (t *Task) NextRun() time.Time { return time.UnixMilli(t.t.NextRun) }
func (t *Task) State() string      { return t.t.State }
func (t *Task) Failed() bool       { return t.t.State == "failed" }

type RunningTask struct {
	id    string
	ttype string
	args  string
}

func (t *RunningTask) ID() string   { return t.id }
func (t *RunningTask) Type() string { return t.ttype }
func (t *RunningTask) Args() string { return t.args }
func (t *Task) FailureDesc() string {
	if s := t.t.FailureDesc; s != nil {
		return *s
	}
	return ""
}

type taskQueue struct {
	name string
	m    *Model

	signal chan struct{} // capacity 1; wakes a worker

	runningMu sync.Mutex
	running   map[string]runEntry
}

type runEntry struct {
	key    string
	ttype  string
	args   string
	cancel context.CancelFunc
}

func newTaskQueue(name string, m *Model) *taskQueue {
	return &taskQueue{
		name:    name,
		m:       m,
		signal:  make(chan struct{}, 1),
		running: map[string]runEntry{},
	}
}

func (tq *taskQueue) runTasks() {
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-tq.signal:
		case <-tick.C:
		}
		for {
			task, err := tq.next()
			if err != nil {
				slog.Error("task-next-error", "error", err)
			}
			if task == nil {
				break
			}
			tq.run(*task)
		}
	}
}

// wake sends a non-blocking signal to unblock one worker.
func (tq *taskQueue) wake() {
	select {
	case tq.signal <- struct{}{}:
	default:
	}
}

func (tq *taskQueue) next() (*schema.Task, error) {
	ctx := context.Background()
	now := time.Now().UnixMilli()
	task, err := schema.New(tq.m.dbr).TaskNext(ctx, schema.TaskNextParams{
		Queue:   tq.name,
		NextRun: now,
	})
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (tq *taskQueue) lock(id, ttype, args string, cancel context.CancelFunc) (string, error) {
	ctx := context.Background()
	_, err := schema.New(tq.m.dbw).TaskLock(ctx, id)
	if err == sql.ErrNoRows {
		return "", nil // someone else locked it
	}
	if err != nil {
		return "", err
	}
	tq.runningMu.Lock()
	defer tq.runningMu.Unlock()
	key := cryptorand.Text()
	tq.running[id] = runEntry{key: key, ttype: ttype, args: args, cancel: cancel}
	return key, nil
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
	ctx := context.Background()
	err := schema.New(tq.m.dbw).TaskUnlock(ctx, id)
	if err != nil {
		slog.Error("task-unlock", "id", id, "error", err)
	}
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

func (tq *taskQueue) run(task schema.Task) {
	ctx := context.Background()
	ctx = logcontext.With(ctx, slog.Group("task", "id", task.ID))
	tq.m.emitEvent(&Event{Type: EventTaskStatsChange})
	defer tq.m.emitEvent(&Event{Type: EventTaskStatsChange})
	err := tq.run1(ctx, task)
	if err != nil {
		slog.ErrorContext(ctx, "error", "error", err)
		if errors.Is(err, ErrPermanent) {
			err = tq.markFailed(ctx, task, err.Error())
		} else {
			err = tq.reschedule(ctx, task, err.Error())
		}
		if err != nil {
			slog.ErrorContext(ctx, "error", "error", err)
		}
	}
}

func (tq *taskQueue) run1(ctx Context, task schema.Task) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if key, err := tq.lock(task.ID, task.Type, task.Args, cancel); err != nil {
		return err
	} else if key == "" {
		return nil
	} else {
		defer tq.unlock(task.ID, key)
	}
	defer func() {
		// run1's only ctx-cancel source is tq.kill, so if ctx.Err() is
		// non-nil here the user explicitly killed the task.
		err = errors.Join(err, Permanent(ctx.Err()))
	}()

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
	delay := taskFailDelay(task.Failures)
	delay += time.Duration(rand.Int64N(int64(delay)))
	_, err := schema.New(tq.m.dbw).TaskReschedule(ctx, schema.TaskRescheduleParams{
		ID:          task.ID,
		Failures:    task.Failures + 1,
		NextRun:     time.Now().Add(delay).UnixMilli(),
		FailureDesc: &failure,
	})
	return err
}

func (tq *taskQueue) markFailed(ctx Context, task schema.Task, failure string) error {
	return schema.New(tq.m.dbw).TaskMarkFailed(ctx, schema.TaskMarkFailedParams{
		ID:          task.ID,
		FailureDesc: &failure,
	})
}

func (m *Model) RunTaskNow(ctx Context, id string) (err error) {
	defer errorfmt.Handlef("run task %s now: %w", id, &err)
	err = schema.New(m.dbw).TaskRunNow(ctx, schema.TaskRunNowParams{
		NextRun: time.Now().UnixMilli(),
		ID:      id,
	})
	if err != nil {
		return err
	}
	for _, tq := range m.tasks {
		tq.wake()
	}
	return nil
}

func (m *Model) KillTask(id string) bool {
	for _, tq := range m.tasks {
		if tq.kill(id) {
			return true
		}
	}
	return false
}

func (tq *taskQueue) runningTasks() []*RunningTask {
	tq.runningMu.Lock()
	defer tq.runningMu.Unlock()
	var out []*RunningTask
	for id, e := range tq.running {
		out = append(out, &RunningTask{id: id, ttype: e.ttype, args: e.args})
	}
	return out
}

func (m *Model) RunningTasks() []*RunningTask {
	var out []*RunningTask
	for _, tq := range m.tasks {
		out = append(out, tq.runningTasks()...)
	}
	return out
}

func (m *Model) wake(ttype string) {
	tq := m.tasks[queueTab[ttype]]
	if tq != nil {
		tq.wake()
	}
}

type TaskStats struct {
	Queued     int
	Running    int
	CountError int
}

func (tx *TxR) TaskStats(ctx Context) (TaskStats, error) {
	queued, err := tx.q.TaskCountQueued(ctx)
	if err != nil {
		return TaskStats{}, err
	}
	countError, err := tx.q.TaskCountError(ctx)
	if err != nil {
		return TaskStats{}, err
	}
	return TaskStats{
		Queued:     int(queued),
		Running:    len(tx.m.RunningTasks()),
		CountError: int(countError),
	}, nil
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

func (tx *TxRW) TaskDelete(ctx Context, id string) error {
	tx.onCommit(func() {
		tx.m.emitEvent(&Event{Type: EventTaskStatsChange})
	})
	return tx.q.TaskDelete(ctx, id)
}

func (t *TxRW) addTask(ctx Context, ttype string, args ...string) error {
	return t.addTaskOpts(ctx, ttype, 0, priority.Default, args...)
}

func (t *TxRW) addTaskAfter(ctx Context, delay time.Duration, ttype string, args ...string) error {
	return t.addTaskOpts(ctx, ttype, delay, priority.Default, args...)
}

func (t *TxRW) addTaskWithPriority(ctx Context, priority int64, ttype string, args ...string) error {
	return t.addTaskOpts(ctx, ttype, 0, priority, args...)
}

func (t *TxRW) addTaskOpts(ctx Context, ttype string, delay time.Duration, pri int64, args ...string) error {
	b, err := json.Marshal(args)
	if err != nil {
		return err
	}
	var nextRun int64
	if delay > 0 {
		nextRun = time.Now().Add(delay).UnixMilli()
	}
	_, err = t.q.TaskCreate(ctx, schema.TaskCreateParams{
		Type:     ttype,
		Args:     string(b),
		Priority: pri,
		Queue:    queueTab[ttype],
		NextRun:  nextRun,
	})
	if err != nil {
		return err
	}
	t.onCommit(func() {
		t.m.wake(ttype)
		t.m.emitEvent(&Event{Type: EventTaskStatsChange})
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
