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

	"lukechampine.com/blake3"

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

func (d *Dir) Link(name string) (hash string, err error) {
	tmp := rand.Text()[:8]
	err = os.Link(name, filepath.Join(d.root.Name(), tmp))
	if err != nil {
		return "", err
	}
	defer d.root.Remove(tmp)
	f, err := d.root.Open(tmp)
	if err != nil {
		return "", err
	}
	hash, err = digest(f)
	if err != nil {
		return "", err
	}
	path := filepath.Join(hash[:2], hash[2:4], hash[4:])
	err = d.root.MkdirAll(path[:5], 0755)
	if err != nil {
		return "", err
	}
	err = d.root.Rename(tmp, path)
	if err != nil {
		return "", err
	}
	return hash, nil
}

func (d *Dir) CreateFunc(f func(*os.File) error) (hash string, err error) {
	tmp := rand.Text()[:8]
	slog.Debug("create", "path", tmp)
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

	slog.Debug("open", "path", tmp)
	r, err := d.root.Open(tmp)
	if err != nil {
		return "", err
	}
	hash, err = digest(r)
	if err != nil {
		return "", err
	}
	path := filepath.Join(hash[:2], hash[2:4], hash[4:])
	err = d.root.MkdirAll(path[:5], 0755)
	if err != nil {
		return "", err
	}
	err = d.root.Rename(tmp, path)
	if err != nil {
		return "", err
	}
	return hash, nil
}

func (d *Dir) Open(hash string) (*os.File, error) {
	p, err := keyPath(hash, true)
	if err != nil {
		return nil, err
	}
	slog.Debug("open", "path", p)
	return d.root.Open(p)
}

type fanoutFS struct{ fs fs.FS }

func (f *fanoutFS) Open(hash string) (fs.File, error) {
	p, err := keyPath(hash, false)
	if err != nil {
		return nil, &fs.PathError{
			Op:   "Open",
			Path: hash,
			Err:  fs.ErrNotExist,
		}
	}
	return f.fs.Open(p)
}

func digest(r io.Reader) (string, error) {
	s := blake3.New(byteSize, nil)
	_, err := io.Copy(s, r)
	if err != nil {
		return "", err
	}
	p := s.Sum(nil)
	return base32c.EncodeToString(p), nil
}

func keyPath(hash string, forFile bool) (string, error) {
	_, err := base32c.DecodeString(hash)
	if len(hash) != b32Size || err != nil {
		return "", fmt.Errorf("bad hash")
	}
	if forFile {
		return filepath.Join(hash[:2], hash[2:4], hash[4:]), nil
	} else {
		return path.Join(hash[:2], hash[2:4], hash[4:]), nil
	}
}
