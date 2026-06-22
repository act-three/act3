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
// It derives the correct URL from a.path, a.odesc,
// and the current slug set in the database,
// and returns a non-nil cmd if the new path differs.
func (a *app) replaceURL(ctx context.Context) (c cmd) {
	if a.odesc == nil {
		return nil // non-slug urls never update
	}
	a.doR(ctx, func(tx *model.TxR) error {
		path := splitPath(a.path)
		_, slugPath, _ := slugs(path)
		dest := replaceSlugSuffix(path, slugPath, tx.SlugPath(a.odesc))
		if dest != "" && dest != a.path {
			c = domi.ReplaceURL[msg.Msg](dest)
		}
		return nil
	})
	return c
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
