package web

import (
	"context"
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/model/progress"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/view"
)

func (c *Config) events(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	rc := http.NewResponseController(w)
	for ev := range c.Model.Events(ctx) {
		if node := c.eventView(ctx, ev); node != nil {
			turbo.EncodeSSE(w, node)
			rc.Flush()
		}
	}
}

func (c *Config) eventView(ctx context.Context, ev *model.Event) html.Node {
	switch ev.Type {
	case progress.EventOpen:
		return view.ProgressItemAppend(ev.Progress)
	case progress.EventUpdate:
		return view.ProgressItemUpdate(ev.Progress)
	case progress.EventClose:
		return view.ProgressItemRemove(ev.Progress)
	case model.EventSeriesSetTitle:
		return view.SeriesSetTitle(ev.ID, ev.NewText)
	case model.EventMovieEditionSetTitle:
		return view.MovieEditionSetTitle(ev.ID, ev.NewText)
	case model.EventMovieEditionSetLabel:
		return view.MovieEditionSetLabel(ev.ID, ev.NewText)
	case model.EventSeriesEditionSetLabel:
		return view.SeriesEditionSetLabel(ev.ID, ev.NewText)
	case model.EventSeriesSetSlug:
		return c.eventSeriesSetSlug(ctx, ev.ID, ev.OldText, ev.NewText)
	case model.EventSeriesEditionSetSlug:
		return c.eventSeriesEditionSetSlug(ctx, ev.ID, ev.OldText, ev.NewText)
	case model.EventMovieSetSlug:
		return c.eventMovieSetSlug(ctx, ev.ID, ev.OldText, ev.NewText)
	case model.EventMovieEditionSetSlug:
		return c.eventMovieEditionSetSlug(ctx, ev.ID, ev.OldText, ev.NewText)
	}
	return nil
}

func (c *Config) eventMovieSetSlug(ctx context.Context, movieID, oldSlug, newSlug string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		mo, err := tx.MovieHead(ctx, movieID)
		if err != nil {
			return nil, err
		}
		editions, err := tx.MovieEditionList(ctx, mo)
		if err != nil {
			return nil, err
		}
		return view.MovieSetSlug(movieID, oldSlug, newSlug, editions), nil
	})
	return n
}

func (c *Config) eventSeriesSetSlug(ctx context.Context, seriesID, oldSlug, newSlug string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		sr, err := tx.SeriesHead(ctx, seriesID)
		if err != nil {
			return nil, err
		}
		editions, err := tx.SeriesEditionList(ctx, sr)
		if err != nil {
			return nil, err
		}
		return view.SeriesSetSlug(seriesID, oldSlug, newSlug, editions), nil
	})
	return n
}

func (c *Config) eventMovieEditionSetSlug(ctx context.Context, editionID, oldSlug, newSlug string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		mo, err := tx.MovieHeadByEditionID(ctx, editionID)
		if err != nil {
			return nil, err
		}
		med, err := tx.MovieEditionHead(ctx, editionID)
		if err != nil {
			return nil, err
		}
		ed := &model.MovieWork{MovieHead: *mo, MovieEditionHead: *med}
		return view.MovieEditionSetSlug(ed, oldSlug), nil
	})
	return n
}

func (c *Config) eventSeriesEditionSetSlug(ctx context.Context, editionID, oldSlug, newSlug string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		sed, err := tx.SeriesEditionHead(ctx, editionID)
		if err != nil {
			return nil, err
		}
		sr, err := tx.SeriesHead(ctx, sed.SeriesID())
		if err != nil {
			return nil, err
		}
		ed := &model.SeriesWork{SeriesHead: *sr, SeriesEditionHead: *sed}
		return view.SeriesEditionSetSlug(ed, oldSlug), nil
	})
	return n
}
