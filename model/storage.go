package model

import "ily.dev/act3/database/schema"

type Storage struct {
	s schema.Storage
}

func newStorageList(list []schema.Storage, err error) ([]*Storage, error) {
	if err != nil {
		return nil, err
	}
	sl := make([]*Storage, len(list))
	for i := range sl {
		sl[i] = &Storage{list[i]}
	}
	return sl, nil
}

func (tx *TxR) StorageList(ctx Context) ([]*Storage, error) {
	return newStorageList(tx.q.StorageList(ctx))
}

func (s *Storage) Path() string     { return s.s.Path }
func (s *Storage) Contents() string { return s.s.Contents }
