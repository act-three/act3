package storage

import (
	"crypto/rand"
	"errors"
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

// ErrBadKey reports a key that is not a well-formed storage key, and
// so could never name a stored blob.
var ErrBadKey = errors.New("bad key")

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

func (d *Dir) CopyFile(name string) (key string, err error) {
	fr, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer fr.Close()
	return d.Copy(fr)
}

func (d *Dir) Copy(r io.Reader) (key string, err error) {
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
	key = newID()
	path, _ := keyPath(key, filepath.Join)
	err = d.root.MkdirAll(path[:5], 0755)
	if err != nil {
		return "", err
	}
	err = d.root.Rename(tmp, path)
	if err != nil {
		return "", err
	}
	return key, nil
}

func (d *Dir) CreateFunc(f func(*os.File) error) (key string, err error) {
	tmp := rand.Text()[:8]
	w, err := d.root.Create(tmp)
	if err != nil {
		return "", err
	}
	defer d.root.Remove(tmp)

	err = f(w)
	if err != nil {
		return "", err
	}
	err = w.Close()
	if err != nil && !errors.Is(err, os.ErrClosed) {
		return "", err
	}

	key = newID()
	path, _ := keyPath(key, filepath.Join)
	err = d.root.MkdirAll(path[:5], 0755)
	if err != nil {
		return "", err
	}
	err = d.root.Rename(tmp, path)
	if err != nil {
		return "", err
	}
	return key, nil
}

func (d *Dir) Remove(key string) error {
	p, err := keyPath(key, filepath.Join)
	if err != nil {
		return err
	}
	return d.root.Remove(p)
}

func (d *Dir) Open(key string) (*os.File, error) {
	p, err := keyPath(key, filepath.Join)
	if err != nil {
		return nil, err
	}
	slog.Debug("open", "path", p)
	return d.root.Open(p)
}

type fanoutFS struct{ fs fs.FS }

func (f *fanoutFS) Open(key string) (fs.File, error) {
	p, err := keyPath(key, path.Join)
	if err != nil {
		return nil, &fs.PathError{
			Op:   "Open",
			Path: key,
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

func keyPath(key string, join func(...string) string) (string, error) {
	_, err := base32c.DecodeString(key)
	if len(key) != b32Size || err != nil {
		return "", ErrBadKey
	}
	return join(key[:2], key[2:4], key[4:]), nil
}
