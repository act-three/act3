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

type Context = context.Context

type Config struct {
	Store         *storage.Dir
	PersistentTmp string
	TMDB          *tmdb.Client
	TVmaze        *tvmaze.Client
}

type Model struct {
	store         *storage.Dir
	persistentTmp string
	tmdb          *tmdb.Client
	tvmaze        *tvmaze.Client

	dbr   *sql.DB
	dbw   *sql.DB
	prog  progress.Tracker
	tasks map[string]*taskQueue

	transmission atomic.Pointer[transmissionrpc.Client]

	activeInfoHashMu sync.Mutex
	activeInfoHash   map[string]bool

	torrentMu sync.Mutex
	torrent   map[string]*transmissionrpc.Torrent

	subMu sync.Mutex
	sub   map[chan *Event]struct{}
}

type model Config

func New(dbr, dbw *sql.DB, c Config) (m *Model, err error) {
	ctx := context.Background()
	defer errorfmt.Handlef("model init: %w", &err)
	m = &Model{
		store:          c.Store,
		persistentTmp:  c.PersistentTmp,
		tmdb:           c.TMDB,
		tvmaze:         c.TVmaze,
		dbr:            dbr,
		dbw:            dbw,
		tasks:          map[string]*taskQueue{},
		activeInfoHash: map[string]bool{},
		torrent:        map[string]*transmissionrpc.Torrent{},
		sub:            map[chan *Event]struct{}{},
	}
	m.prog.SetHook(func(event string, it *progress.Item) {
		m.addEvent(&Event{Type: event, Progress: it})
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
	err = schema.New(dbw).TaskResetRunning(ctx)
	if err != nil {
		return nil, err
	}
	for name, n := range taskQueues {
		tq := newTaskQueue(name, m)
		for range n {
			go tq.runTasks()
		}
		m.tasks[name] = tq
	}
	hashes, err := schema.New(dbr).DownloadListInfoHashesActive(ctx)
	if err != nil {
		return nil, err
	}
	for _, h := range hashes {
		m.setInfoHashActive(h, true)
	}
	go m.pollTransission()
	return m, nil
}

var configLoaders = []func(*TxR, Context) error{
	(*TxR).loadTMDBConfig,
	(*TxR).loadTransmissionConfig,
}

func (m *Model) loadConfig(ctx Context) (err error) {
	defer errorfmt.Handlef("load config: %w", &err)
	return m.WithTxR(func(t *TxR) error {
		for _, f := range configLoaders {
			err = f(t, ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *Model) WithTxR(f func(*TxR) error) error {
	tx, err := m.dbr.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return f(&TxR{
		m:  m,
		tx: tx,
		q:  schema.New(tx),
	})
}

// Non-nil error will roll back the transation.
func (m *Model) WithTxRW(f func(*TxRW) error) error {
	tx, err := m.dbw.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	mt := &TxRW{TxR: TxR{
		m:  m,
		tx: tx,
		q:  schema.New(tx),
	}}
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
	return nil
}

type TxR struct {
	m  *Model
	tx *sql.Tx
	q  *schema.Queries
}

type TxRW struct {
	TxR
	commitHook []func()
}

func (mt *TxRW) onCommit(f func()) {
	mt.commitHook = append(mt.commitHook, f)
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
