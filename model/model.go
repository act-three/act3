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
	"ily.dev/act3/service/tvmaze"
	"ily.dev/act3/storage"
)

type Context = context.Context

type Config struct {
	Store  *storage.Dir
	TVmaze *tvmaze.Client
}

type Model struct {
	store  *storage.Dir
	tvmaze *tvmaze.Client

	dbr   *sql.DB
	dbw   *sql.DB
	prog  progress
	tasks map[string]*taskQueue

	transmission atomic.Pointer[transmissionrpc.Client]

	activeInfoHashMu sync.Mutex
	activeInfoHash   map[string]bool

	fileProgressMu sync.RWMutex
	fileProgress   map[string]map[string]float64 // infoHash -> filePath -> fraction [0,1]
}

type model Config

func New(dbr, dbw *sql.DB, c Config) (m *Model, err error) {
	ctx := context.Background()
	defer errorfmt.Handlef("model init: %w", &err)
	m = &Model{
		store:          c.Store,
		tvmaze:         c.TVmaze,
		dbr:            dbr,
		dbw:            dbw,
		tasks:          map[string]*taskQueue{},
		activeInfoHash: map[string]bool{},
		fileProgress:   map[string]map[string]float64{},
	}
	err = m.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	for name, n := range taskQueues {
		ch := make(chan schema.Task)
		tq := newTaskQueue(name, m, ch)
		err = tq.startTasks(ctx, n, ch)
		if err != nil {
			return nil, err
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
