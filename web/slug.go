package web

import (
	"context"
	"path"

	"ily.dev/domi"

	"ily.dev/act3/model"
	"ily.dev/act3/msg"
)

// Sections a resolved object page can live in.
const (
	sectionTheater = "theater"
	sectionEditor  = "editor"
)

// replaceURL returns a ReplaceURL cmd, if necessary,
// to update the current page's path.
// It derives the correct URL from a.path
// and the current slug set in the database,
// and returns a non-nil cmd if the canonical path differs.
func (a *app) replaceURL(ctx context.Context) (c cmd) {
	a.doR(ctx, func(tx *model.TxR) error {
		path := splitPath(a.path)
		odesc, slugPath := resolve(tx, path)
		if odesc == nil {
			return nil // non-slug urls never update
		}
		dest := replaceSlugSuffix(path, slugPath, tx.SlugPath(odesc))
		if dest != "" && dest != a.path {
			c = domi.ReplaceURL[msg.Msg](dest)
		}
		return nil
	})
	return c
}

// resolve resolves the slug suffix of path to an object descriptor,
// also returning the suffix itself. The odesc is nil when the path
// has no slug section or its slugs don't resolve.
func resolve(tx *model.TxR, path []string) (odesc map[string]string, slugPath []string) {
	section, slugPath, allowed := slugs(path)
	if section == "" {
		return nil, nil
	}
	return slugResolve(tx, slugPath, allowed), slugPath
}

func replaceSlugSuffix(p, oldsuf []string, newsuf string) string {
	if newsuf == "" {
		return ""
	}
	n := len(p) - len(oldsuf)
	if n < 0 {
		return ""
	}
	return path.Join("/", path.Join(p[:n]...), newsuf)
}
