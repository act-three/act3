package storage

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"

	"ily.dev/act3/encoding/base32c"
)

const (
	byteSize = 15
	b32Size  = byteSize * 8 / 5
)

type Dir struct {
	root *os.Root
}

func Open(name string) (*Dir, error) {
	root, err := os.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	return &Dir{root}, nil
}

func (d *Dir) FS() fs.FS {
	return &fanoutFS{d.root.FS()}
}

func (d *Dir) CopyFile(name string) (id string, err error) {
	fr, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer fr.Close()
	tmp := rand.Text()[:8]
	fw, err := d.root.Create(tmp)
	if err != nil {
		return "", err
	}
	defer fw.Close()
	defer d.root.Remove(tmp)
	_, err = io.Copy(fw, fr)
	if err != nil {
		return "", err
	}
	id = newID()
	path := filepath.Join(id[:2], id[2:4], id[4:])
	err = d.root.MkdirAll(path[:5], 0755)
	if err != nil {
		return "", err
	}
	err = d.root.Rename(tmp, path)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (d *Dir) Copy(r io.Reader) (id string, err error) {
	tmp := rand.Text()[:8]
	fw, err := d.root.Create(tmp)
	if err != nil {
		return "", err
	}
	defer fw.Close()
	defer d.root.Remove(tmp)
	_, err = io.Copy(fw, r)
	if err != nil {
		return "", err
	}
	id = newID()
	path := filepath.Join(id[:2], id[2:4], id[4:])
	err = d.root.MkdirAll(path[:5], 0755)
	if err != nil {
		return "", err
	}
	err = d.root.Rename(tmp, path)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (d *Dir) CreateNFunc(n int, f func([]*os.File) error) (ids []string, err error) {
	var tmps []string
	for range n {
		tmps = append(tmps, rand.Text()[:8])
	}
	var ws []*os.File
	for _, tmp := range tmps {
		w, err := d.root.Create(tmp)
		if err != nil {
			return nil, err
		}
		defer d.root.Remove(tmp)
		ws = append(ws, w)
	}

	err = f(ws)
	if err != nil {
		return nil, err
	}
	for _, w := range ws {
		err = w.Close()
		if err != nil && !errors.Is(err, os.ErrClosed) {
			return nil, err
		}
	}

	for _, tmp := range tmps {
		slog.Debug("open", "path", tmp)
		id := newID()
		path := filepath.Join(id[:2], id[2:4], id[4:])
		err = d.root.MkdirAll(path[:5], 0755)
		if err != nil {
			return nil, err
		}
		err = d.root.Rename(tmp, path)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (d *Dir) Remove(id string) error {
	p, err := keyPath(id, true)
	if err != nil {
		return err
	}
	return d.root.Remove(p)
}

func (d *Dir) Open(id string) (*os.File, error) {
	p, err := keyPath(id, true)
	if err != nil {
		return nil, err
	}
	slog.Debug("open", "path", p)
	return d.root.Open(p)
}

type fanoutFS struct{ fs fs.FS }

func (f *fanoutFS) Open(id string) (fs.File, error) {
	p, err := keyPath(id, false)
	if err != nil {
		return nil, &fs.PathError{
			Op:   "Open",
			Path: id,
			Err:  fs.ErrNotExist,
		}
	}
	return f.fs.Open(p)
}

func newID() string {
	p := make([]byte, byteSize)
	rand.Read(p)
	return base32c.EncodeToString(p)
}

func keyPath(id string, forFile bool) (string, error) {
	_, err := base32c.DecodeString(id)
	if len(id) != b32Size || err != nil {
		return "", fmt.Errorf("bad id")
	}
	if forFile {
		return filepath.Join(id[:2], id[2:4], id[4:]), nil
	} else {
		return path.Join(id[:2], id[2:4], id[4:]), nil
	}
}
