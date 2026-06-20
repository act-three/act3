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
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/log/logcontext"
	"ily.dev/act3/priority"
	"ily.dev/act3/tlog"
	"ily.dev/act3/video/ffmpeg"
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
	queueNet  = "net"
	queueDisk = "disk"
	queueCPU  = "cpu"
)

type taskFunc func(*TxR, []string) error

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
	taskAddDownloadToTransmission: queueNet,
	taskFetchEpisodes:             queueNet,
	taskFetchEpisodeThumbnail:     queueNet,
	taskFetchSeriesPoster:         queueNet,
	taskFetchMoviePoster:          queueNet,
	taskIngest:                    queueDisk,
	taskIngestEncodeAudio:         queueCPU,
	taskIngestEncodeDownloadRend:  queueCPU,
	taskIngestEncodeRend:          queueCPU,
	taskIngestExtractSubs:         queueDisk,
	taskIngestPass1:               queueCPU,
	taskReimport:                  queueDisk,
	taskReencode:                  queueDisk,
}

// taskQueues maps each queue to its admission capacity. The net
// queue bounds in-flight HTTP work (the API rate limiters in
// service/ are the real throughput guard); the disk queue bounds
// concurrent heavy file work, where two sequential copies already
// saturate disk bandwidth. Both count tasks (all weigh 1). The cpu
// queue counts cores: GOMAXPROCS rather than NumCPU so container
// CPU quotas and an explicit GOMAXPROCS setting are respected.
var taskQueues = map[string]int{
	queueNet:  4,
	queueDisk: 2,
	queueCPU:  runtime.GOMAXPROCS(0),
}

// taskWeight maps a task type to its admission weight; absent types
// weigh 1. cpu-queue weights estimate the cores a task keeps busy:
// a rendition encode runs one ffmpeg encoder, and pass 1 bundles
// all of a video's encoders (typically five) into one process.
// These are nominal: encode tasks call reweighTask once they know
// which rendition they drew.
var taskWeight = map[string]int{
	taskIngestPass1:              5 * ffmpeg.EncoderThreads,
	taskIngestEncodeRend:         ffmpeg.EncoderThreads,
	taskIngestEncodeDownloadRend: ffmpeg.EncoderThreads,
}

func weightFor(ttype string) int {
	if w := taskWeight[ttype]; w > 0 {
		return w
	}
	return 1
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
	name     string
	capacity int // admission budget; see taskQueues
	m        *Model

	signal chan struct{} // capacity 1; wakes the dispatcher

	runningMu sync.Mutex
	used      int // total weight of running tasks
	running   map[string]runEntry
}

type runEntry struct {
	key    string
	ttype  string
	args   string
	weight int
	cancel context.CancelFunc
}

func newTaskQueue(name string, capacity int, m *Model) *taskQueue {
	return &taskQueue{
		name:     name,
		capacity: capacity,
		m:        m,
		signal:   make(chan struct{}, 1),
		running:  map[string]runEntry{},
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
		for tq.canAdmit() {
			task, err := tq.next()
			if err != nil {
				slog.Error("task-next-error", "error", err)
			}
			if task == nil {
				break
			}
			tq.dispatch(*task)
		}
	}
}

// canAdmit reports whether the queue has budget to start another
// task. The test deliberately ignores the candidate's own weight:
// a task wider than the remaining budget still starts, overbooking
// the queue by at most one task, so that wide tasks can't starve
// and a nearly-full queue never idles.
func (tq *taskQueue) canAdmit() bool {
	tq.runningMu.Lock()
	defer tq.runningMu.Unlock()
	return tq.used < tq.capacity
}

// dispatch locks task and, if the lock is won, runs it on its own
// goroutine. Locking happens synchronously so that by the time
// dispatch returns, TaskNext no longer offers the task.
func (tq *taskQueue) dispatch(task schema.Task) {
	ctx := context.Background()
	ctx = logcontext.With(ctx, slog.Group("task", "id", task.ID))
	ctx = context.WithValue(ctx, taskIDKey{}, task.ID)
	ctx, cancel := context.WithCancel(ctx)
	key, err := tq.lock(task.ID, task.Type, task.Args, cancel)
	if err != nil {
		cancel()
		slog.Error("task-lock-error", "id", task.ID, "error", err)
		return
	}
	if key == "" {
		cancel()
		return // someone else locked it
	}
	go func() {
		defer tq.unlock(task.ID, key)
		tq.run(ctx, task)
	}()
}

// taskIDKey carries the running task's ID in its context, so task
// funcs can find their own queue entry (see reweighTask).
type taskIDKey struct{}

// reweighTask replaces the admission weight of the running task
// that owns ctx, for use once a task learns its true width (e.g.
// which rendition it drew). No-op when ctx carries no task ID.
func (m *Model) reweighTask(ctx context.Context, weight int) {
	id, ok := ctx.Value(taskIDKey{}).(string)
	if !ok {
		return
	}
	for _, tq := range m.tasks {
		if tq.reweigh(id, weight) {
			return
		}
	}
}

// reweigh sets the weight of running task id, reporting whether the
// task was found. Shrinking frees budget, so it wakes the
// dispatcher.
func (tq *taskQueue) reweigh(id string, weight int) bool {
	tq.runningMu.Lock()
	e, ok := tq.running[id]
	if !ok {
		tq.runningMu.Unlock()
		return false
	}
	shrunk := weight < e.weight
	tq.used += weight - e.weight
	e.weight = weight
	tq.running[id] = e
	tq.runningMu.Unlock()
	if shrunk {
		tq.wake()
	}
	return true
}

// wake sends a non-blocking signal to unblock the dispatcher.
func (tq *taskQueue) wake() {
	select {
	case tq.signal <- struct{}{}:
	default:
	}
}

func (tq *taskQueue) next() (*schema.Task, error) {
	ctx := context.Background()
	now := time.Now().UnixMilli()
	task, err := schema.New(ctx, tq.m.dbr).TaskNext(schema.TaskNextParams{
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
	_, err := schema.New(ctx, tq.m.dbw).TaskLock(id)
	if err == sql.ErrNoRows {
		return "", nil // someone else locked it
	}
	if err != nil {
		return "", err
	}
	tq.runningMu.Lock()
	defer tq.runningMu.Unlock()
	key := cryptorand.Text()
	w := weightFor(ttype)
	tq.running[id] = runEntry{key: key, ttype: ttype, args: args, weight: w, cancel: cancel}
	tq.used += w
	return key, nil
}

// unlock releases the in-memory admission slot held by a finished
// task. It writes no DB state: run already moved the task to its
// terminal state (deleted, failed, or requeued), and any task left in
// 'running' by a crash is requeued by TaskResetRunning at startup.
func (tq *taskQueue) unlock(id, key string) {
	tq.runningMu.Lock()
	defer tq.runningMu.Unlock()
	e := tq.running[id]
	if e.key != key {
		panic("bad lock structure: key mismatch")
	}
	e.cancel()
	delete(tq.running, id)
	tq.used -= e.weight
	tq.wake()
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

func (tq *taskQueue) run(ctx context.Context, task schema.Task) {
	tq.m.emit(nil)
	defer tq.m.emit(nil)
	err, stack := tq.run1(ctx, task)
	if err != nil {
		slog.ErrorContext(ctx, "error", "error", err)
		// A killed task arrives here with ctx already canceled.
		// Detach so the failure write does not itself fail.
		ctx := context.WithoutCancel(ctx)
		if errors.Is(err, ErrPermanent) {
			err = tq.markFailed(ctx, task, err.Error(), stack)
		} else {
			err = tq.reschedule(ctx, task, err.Error())
		}
		if err != nil {
			slog.ErrorContext(ctx, "error", "error", err)
		}
	}
}

func (tq *taskQueue) run1(ctx context.Context, task schema.Task) (err error, stack []byte) {
	defer func() {
		// run1's only ctx-cancel source is tq.kill, so if ctx.Err() is
		// non-nil here the user explicitly killed the task.
		err = errors.Join(err, Permanent(ctx.Err()))
	}()

	defer tlog.Elapsed(ctx, "task", "type", task.Type, "args", task.Args)()

	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "panic", "error", r)
			stack = debug.Stack()
			os.Stderr.Write(stack)
			e1, ok := r.(error)
			if !ok {
				e1 = fmt.Errorf("%v", r)
			}
			err = Permanent(fmt.Errorf("task recovered panic: %w", e1))
		}
	}()

	f := taskTab[task.Type]
	if f == nil {
		return errors.New("unknown task type: " + task.Type), nil
	}
	var args []string
	err = json.Unmarshal([]byte(task.Args), &args)
	if err != nil {
		return fmt.Errorf("task %s %s: bad args: %w", task.Type, task.ID, err), nil
	}

	return tq.m.WithTxR(ctx, func(tx *TxR) error {
		task, err = tx.q.TaskGet(task.ID)
		if err == sql.ErrNoRows {
			return nil
		} else if err != nil {
			return err
		}

		err = f(tx, args)
		if err != nil {
			return err
		}
		return tq.m.WithTxRW(ctx, func(t *TxRW) error {
			return t.q.TaskDelete(task.ID)
		})
	}), nil
}

func (tq *taskQueue) reschedule(ctx context.Context, task schema.Task, failure string) error {
	delay := taskFailDelay(task.Failures)
	delay += time.Duration(rand.Int64N(int64(delay)))
	_, err := schema.New(ctx, tq.m.dbw).TaskReschedule(schema.TaskRescheduleParams{
		ID:          task.ID,
		Failures:    task.Failures + 1,
		NextRun:     time.Now().Add(delay).UnixMilli(),
		Priority:    task.Priority + 1, // minimize interference with other work
		FailureDesc: &failure,
	})
	return err
}

func (tq *taskQueue) markFailed(ctx context.Context, task schema.Task, failure string, stack []byte) error {
	if len(stack) > 0 {
		failure = failure + "\n\n" + string(stack)
	}
	return schema.New(ctx, tq.m.dbw).TaskMarkFailed(schema.TaskMarkFailedParams{
		ID:          task.ID,
		FailureDesc: &failure,
	})
}

func (m *Model) RunTaskNow(ctx context.Context, id string) (err error) {
	defer errorfmt.Handlef("run task %s now: %w", id, &err)
	err = schema.New(ctx, m.dbw).TaskRunNow(schema.TaskRunNowParams{
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

// RunningTasks reports the tasks currently executing.
// It reflects live queue state at the time of the call,
// not state as of the transaction's snapshot.
func (tx *TxR) RunningTasks() []*RunningTask {
	var out []*RunningTask
	for _, tq := range tx.m.tasks {
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

func (tx *TxR) TaskStats() (TaskStats, error) {
	queued, err := tx.q.TaskCountQueued()
	if err != nil {
		return TaskStats{}, err
	}
	countError, err := tx.q.TaskCountError()
	if err != nil {
		return TaskStats{}, err
	}
	return TaskStats{
		Queued:     int(queued),
		Running:    len(tx.RunningTasks()),
		CountError: int(countError),
	}, nil
}

func (tx *TxR) TaskList() ([]*Task, error) {
	list, err := tx.q.TaskList()
	if err != nil {
		return nil, err
	}
	var tasks []*Task
	for _, t := range list {
		tasks = append(tasks, &Task{t})
	}
	return tasks, nil
}

func (tx *TxRW) TaskDelete(id string) error {
	return tx.q.TaskDelete(id)
}

func (t *TxRW) addTask(ttype string, args ...string) error {
	return t.addTaskOpts(ttype, 0, priority.Default, args...)
}

func (t *TxRW) addTaskAfter(delay time.Duration, ttype string, args ...string) error {
	return t.addTaskOpts(ttype, delay, priority.Default, args...)
}

func (t *TxRW) addTaskWithPriority(priority int64, ttype string, args ...string) error {
	return t.addTaskOpts(ttype, 0, priority, args...)
}

func (t *TxRW) addTaskOpts(ttype string, delay time.Duration, pri int64, args ...string) error {
	b, err := json.Marshal(args)
	if err != nil {
		return err
	}
	var nextRun int64
	if delay > 0 {
		nextRun = time.Now().Add(delay).UnixMilli()
	}
	_, err = t.q.TaskCreate(schema.TaskCreateParams{
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
