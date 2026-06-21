// Package model contains the core logic of Act Three.
// Code is organized into database-access (Model, TxR, TxRW)
// and data objects (Movie, Series, Episode, etc).
//
// The database access interface is responsible
// for coordinating I/O, running background tasks, measuring progress, etc.
// Methods on TxR return data objects
// for use by the rest of the system (e.g. view code);
// methods on TxRW modify the database and return data objects.
//
// Data objects carry readonly data.
// They're constructed by package model from database entries
// and other sources.
// They don't carry database handles, they perform no I/O,
// and they don't mutate.
package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/hekmon/transmissionrpc/v3"
	"kr.dev/errorfmt"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/model/progress"
	"ily.dev/act3/service/tmdb"
	"ily.dev/act3/service/tvmaze"
	"ily.dev/act3/storage"
)

type Config struct {
	Store    *storage.Dir
	Pass1Dir string
	TMDB     *tmdb.Client
	TVmaze   *tvmaze.Client
}

type Model struct {
	store     *storage.Dir
	pass1Root string
	tmdb      *tmdb.Client
	tvmaze    *tvmaze.Client

	dbr   *sql.DB
	dbw   *sql.DB
	prog  progress.Tracker
	tasks map[string]*taskQueue

	transmission atomic.Pointer[transmissionrpc.Client]

	torrentMu sync.Mutex
	torrent   map[string]*transmissionrpc.Torrent

	downloadDirMu sync.Mutex
	downloadDir   map[string]string // Transmission DownloadDir → local path

	uploadMu sync.Mutex
	uploads  []*upload

	subMu sync.Mutex
	sub   map[chan *Event]struct{}
}

type model Config

func New(dbr, dbw *sql.DB, c Config) (m *Model, err error) {
	ctx := context.Background()
	defer errorfmt.Handlef("model init: %w", &err)
	m = &Model{
		store:       c.Store,
		pass1Root:   c.Pass1Dir,
		tmdb:        c.TMDB,
		tvmaze:      c.TVmaze,
		dbr:         dbr,
		dbw:         dbw,
		tasks:       map[string]*taskQueue{},
		torrent:     map[string]*transmissionrpc.Torrent{},
		downloadDir: map[string]string{},
		sub:         map[chan *Event]struct{}{},
	}
	m.prog.SetHook(func(string, *progress.Item) {
		m.emit(nil)
	})
	m.registerTMDBSettingHooks()
	m.registerTransmissionSettingHooks()
	err = m.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	err = m.insertPlaceholderImages(ctx)
	if err != nil {
		return nil, err
	}
	err = schema.New(ctx, dbw).TaskResetRunning()
	if err != nil {
		return nil, err
	}
	for name, capacity := range taskQueues {
		tq := newTaskQueue(name, capacity, m)
		go tq.runTasks()
		m.tasks[name] = tq
	}
	go m.pollTransission()
	go m.purgeTrashLoop()
	go m.autoTrashDownloadsLoop()
	return m, nil
}

var configLoaders = []func(*TxR) error{
	(*TxR).loadTMDBConfig,
	(*TxR).loadTransmissionConfig,
}

func (m *Model) loadConfig(ctx context.Context) (err error) {
	defer errorfmt.Handlef("load config: %w", &err)
	return m.WithTxR(ctx, func(t *TxR) error {
		for _, f := range configLoaders {
			err = f(t)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// WithTxR calls f with a read-only transaction object.
// Most TxR methods abort the tx on error (including lookup failures).
// WithTxR returns any such aborts. Otherwise, it returns the error from f.
func (m *Model) WithTxR(ctx context.Context, f func(*TxR) error) (err error) {
	defer txrecover(&err)
	tx, err := m.dbr.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return f(&TxR{
		m:   m,
		tx:  tx,
		q:   schema.New(ctx, tx),
		ctx: ctx,
	})
}

// WithTxRW calls f with a read-write transaction object.
// Most read-only TxR methods abort the tx on error (including lookup failures).
// WithTxRW returns any such aborts. Otherwise, it returns the error from f.
// If f returns a non-nil error, WithTxRW rolls back the transaction.
func (m *Model) WithTxRW(ctx context.Context, f func(*TxRW) error) (err error) {
	defer txrecover(&err)
	tx, err := m.dbw.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	mt := &TxRW{
		m:   m,
		tx:  tx,
		q:   schema.New(ctx, tx),
		ctx: ctx,
	}
	err = f(mt)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	for _, f := range mt.commitHook {
		f()
	}
	m.emit(mt.details)
	return nil
}

type TxR struct {
	m   *Model
	tx  *sql.Tx
	q   *schema.Queries
	ctx context.Context
}

type TxRW struct {
	TxR
	commitHook []func()
	details    []Detail
}

func (mt *TxRW) onCommit(f func()) {
	mt.commitHook = append(mt.commitHook, f)
}

// emitDetail adds d to the Event emitted when this transaction commits.
func (mt *TxRW) emitDetail(d Detail) {
	mt.details = append(mt.details, d)
}

type ValidationError struct {
	Op  string
	Err error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

type txError struct{ err error }

func (e *txError) Error() string { return e.err.Error() }
func (e *txError) Unwrap() error { return e.err }

func txrecover(err *error) {
	r := recover()
	e, _ := r.(error)
	if _, ok := errors.AsType[*txError](e); ok {
		*err = e
		return
	}
	if r != nil {
		panic(r)
	}
}

func txmust(err error) {
	if err != nil {
		panic(&txError{err})
	}
}

func txmust1[T any](v T, err error) T {
	txmust(err)
	return v
}

func txfind1[T any](v T, err error) (z T, found bool) {
	if errors.Is(err, sql.ErrNoRows) {
		return z, false
	}
	txmust(err)
	return v, true
}
