package digest

import (
	"crypto/sha3"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"kr.dev/walk"

	"ily.dev/act3/encoding/base32c"
)

type Handler struct {
	fs     fs.FS
	toHash map[string]string
	toBare map[string]string
}

func New(fsys fs.FS) (*Handler, error) {
	h := &Handler{
		fs:     fsys,
		toHash: map[string]string{},
		toBare: map[string]string{},
	}

	w := walk.New(fsys, ".")
	for w.Next() {
		if w.Err() != nil {
			return nil, w.Err()
		}
		ent := w.Entry()
		if ent.IsDir() {
			continue
		}
		hashPath, err := digest(fsys, w.Path())
		if err != nil {
			return nil, err
		}
		h.toHash[w.Path()] = hashPath
		h.toBare[hashPath] = w.Path()
	}

	return h, nil
}

// NameToDigest returns the digested path for name.
// If name is not found, it returns the empty string.
func (h *Handler) NameToDigest(name string) string {
	s, ok := h.toHash[strings.TrimPrefix(name, "/")]
	if !ok {
		return ""
	}
	return path.Join("/", s)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/")
	w.Header().Set("Cache-Control", "max-age=31536000, immutable")
	http.ServeFileFS(w, req, h.fs, h.toBare[path])
}

func digest(fsys fs.FS, name string) (string, error) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return "", err
	}
	sum := sha3.Sum256(b)
	s := strings.ToLower(base32c.EncodeToString(sum[:])[:6])
	ext := path.Ext(name)
	return strings.TrimSuffix(name, ext) + "." + s + ext, nil
}
