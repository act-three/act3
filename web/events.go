package web

import (
	"context"
	"net/http"
	"strconv"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/model/progress"
	"ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/view"
	"ily.dev/act3/view/sidebar"
)

func (c *Config) events(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	h := w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
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
	case model.EventLiveUpdate:
		return ui.LiveTextUpdate(ev.NewText, ev.Addr)
	case progress.EventOpen:
		return view.ProgressItemAppend(ev.Progress)
	case progress.EventUpdate:
		return view.ProgressItemUpdate(ev.Progress)
	case progress.EventClose:
		return view.ProgressItemRemove(ev.Progress)
	case model.EventSeriesSetSlug:
		return c.eventSeriesSetSlug(ctx, ev.ID, ev.OldText, ev.NewText)
	case model.EventSeriesEditionSetSlug:
		return c.eventSeriesEditionSetSlug(ctx, ev.ID, ev.OldText, ev.NewText)
	case model.EventMovieSetSlug:
		return c.eventMovieSetSlug(ctx, ev.ID, ev.OldText, ev.NewText)
	case model.EventMovieEditionSetSlug:
		return c.eventMovieEditionSetSlug(ctx, ev.ID, ev.OldText, ev.NewText)
	case model.EventCollectionMovieAdd:
		return c.eventCollectionMovieAdd(ctx, ev.ID, ev.NewText)
	case model.EventCollectionMovieRemove:
		return c.eventCollectionMovieRemove(ctx, ev.ID, ev.NewText)
	case model.EventCollectionSeriesAdd:
		return c.eventCollectionSeriesAdd(ctx, ev.ID, ev.NewText)
	case model.EventCollectionSeriesRemove:
		return c.eventCollectionSeriesRemove(ctx, ev.ID, ev.NewText)
	case model.EventCollectionChangeBanner:
		return c.eventCollectionChangeBanner(ctx, ev.ID)
	case model.EventCollectionSetSlug:
		return c.eventCollectionSetSlug(ctx, ev.ID, ev.OldText, ev.NewText)
	case model.EventMovieEditionChangePoster:
		return c.eventMovieEditionChangePoster(ctx, ev.ID)
	case model.EventSeriesEditionChangePoster:
		return c.eventSeriesEditionChangePoster(ctx, ev.ID)
	case model.EventEpisodeChangeThumbnail:
		return c.eventEpisodeChangeThumbnail(ctx, ev.ID)
	case model.EventDownloadFileAttach:
		return c.eventDownloadFileAttach(ctx, ev.ID, ev.NewText)
	case model.EventSeasonAdd:
		return c.eventSeasonAdd(ctx, ev.ID)
	case model.EventSeasonRenumber:
		return c.eventSeasonRenumber(ctx, ev.ID)
	case model.EventSeasonEpisodeAdd:
		return view.EpisodeEditionButtonUpdate(ev.ID, ev.NewText, true, 0)
	case model.EventSeasonEpisodeRemove:
		sortKey, _ := strconv.ParseInt(ev.OldText, 10, 64)
		return view.EpisodeEditionButtonUpdate(ev.ID, ev.NewText, false, sortKey)
	case model.EventTaskStatsChange:
		return c.eventTaskStatsChange(ctx)
	case model.EventTrash:
		return c.eventTrash(ctx, ev.TrashKind, ev.ID)
	case model.EventTrashCascade:
		return c.eventTrashCascade(ev.TrashItems)
	case model.EventRestore:
		return c.eventRestore(ctx, ev.TrashKind, ev.ID, ev.TrashItems)
	case model.EventPurge:
		return c.eventPurge(ev.ID)
	}
	return nil
}

func (c *Config) eventTrash(ctx context.Context, kind model.TrashKind, id string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		it, err := tx.TrashItem(ctx, id)
		if err != nil {
			return view.MediaListRemove(kind, id), nil
		}
		return html.Group(
			view.MediaListRemove(kind, id),
			view.TrashListAppend(&it),
		), nil
	})
	return n
}

func (c *Config) eventTrashCascade(items []model.TrashItem) html.Node {
	var nodes []html.Node
	for _, it := range items {
		if n := view.MediaListRemove(it.Kind, it.ID); n != nil {
			nodes = append(nodes, n)
		}
	}
	if len(nodes) == 0 {
		return nil
	}
	return html.Group(nodes...)
}

func (c *Config) eventRestore(ctx context.Context, kind model.TrashKind, id string, ancestors []model.TrashItem) html.Node {
	nodes := []html.Node{view.TrashListRemove(id)}
	for _, it := range ancestors {
		nodes = append(nodes, view.TrashListRemove(it.ID))
	}
	if n := c.mediaListAppend(ctx, kind, id); n != nil {
		nodes = append(nodes, n)
	}
	return html.Group(nodes...)
}

func (c *Config) mediaListAppend(ctx context.Context, kind model.TrashKind, id string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		switch kind {
		case model.TrashKindMovie:
			mo, err := tx.MovieHead(ctx, id)
			if err != nil {
				return nil, err
			}
			meds, err := tx.MovieEditionList(ctx, mo)
			if err != nil {
				return nil, err
			}
			for _, mw := range meds {
				if mw.MovieEditionHead.Slug() == "" {
					return view.MoviesListAppend(mw), nil
				}
			}
			return nil, nil
		case model.TrashKindSeries:
			sr, err := tx.SeriesHead(ctx, id)
			if err != nil {
				return nil, err
			}
			sws, err := tx.SeriesEditionList(ctx, sr)
			if err != nil {
				return nil, err
			}
			for _, sw := range sws {
				if sw.SeriesEditionHead.Slug() == "" {
					return view.SeriesListAppend(sw), nil
				}
			}
			return nil, nil
		case model.TrashKindCollection:
			col, err := tx.CollectionHead(ctx, id)
			if err != nil {
				return nil, err
			}
			return view.CollectionsListAppend(col), nil
		}
		return nil, nil
	})
	return n
}

func (c *Config) eventPurge(id string) html.Node {
	return view.TrashListRemove(id)
}

func (c *Config) eventTaskStatsChange(ctx context.Context) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		stats, err := tx.TaskStats(ctx)
		if err != nil {
			return nil, err
		}
		return sidebar.TaskStats(stats.Queued+stats.Running, stats.CountError), nil
	})
	return n
}

func (c *Config) eventSeasonAdd(ctx context.Context, seasonID string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		sn, err := tx.SeasonInEdition(ctx, seasonID)
		if err != nil {
			return nil, err
		}
		return view.SeasonAppend(sn), nil
	})
	return n
}

func (c *Config) eventSeasonRenumber(ctx context.Context, seasonID string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		sn, err := tx.SeasonInEdition(ctx, seasonID)
		if err != nil {
			return nil, err
		}
		return view.SeasonEpisodesUpdate(sn), nil
	})
	return n
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
		return view.MovieSetSlug(mo, oldSlug, editions), nil
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
		return view.SeriesSetSlug(sr, oldSlug, editions), nil
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

func (c *Config) eventCollectionMovieAdd(ctx context.Context, colID, movieID string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		col, err := tx.Collection(ctx, colID)
		if err != nil {
			return nil, err
		}
		return view.CollectionMovieAppend(col), nil
	})
	return n
}

func (c *Config) eventCollectionMovieRemove(ctx context.Context, colID, movieID string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		col, err := tx.Collection(ctx, colID)
		if err != nil {
			return nil, err
		}
		return view.CollectionMovieRemove(col, movieID), nil
	})
	return n
}

func (c *Config) eventCollectionSeriesAdd(ctx context.Context, colID, seriesID string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		col, err := tx.Collection(ctx, colID)
		if err != nil {
			return nil, err
		}
		return view.CollectionSeriesAppend(col), nil
	})
	return n
}

func (c *Config) eventCollectionSeriesRemove(ctx context.Context, colID, seriesID string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		col, err := tx.Collection(ctx, colID)
		if err != nil {
			return nil, err
		}
		return view.CollectionSeriesRemove(col, seriesID), nil
	})
	return n
}

func (c *Config) eventCollectionSetSlug(ctx context.Context, colID, oldSlug, newSlug string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		col, err := tx.CollectionHead(ctx, colID)
		if err != nil {
			return nil, err
		}
		return view.CollectionSetSlug(col, oldSlug), nil
	})
	return n
}

func (c *Config) eventCollectionChangeBanner(ctx context.Context, colID string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		col, err := tx.CollectionHead(ctx, colID)
		if err != nil {
			return nil, err
		}
		return view.CollectionChangeBanner(col), nil
	})
	return n
}

func (c *Config) eventMovieEditionChangePoster(ctx context.Context, editionID string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		med, err := tx.MovieEditionHead(ctx, editionID)
		if err != nil {
			return nil, err
		}
		return view.MovieEditionChangePoster(med), nil
	})
	return n
}

func (c *Config) eventSeriesEditionChangePoster(ctx context.Context, editionID string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		sed, err := tx.SeriesEditionHead(ctx, editionID)
		if err != nil {
			return nil, err
		}
		return view.SeriesEditionChangePoster(sed), nil
	})
	return n
}

func (c *Config) eventEpisodeChangeThumbnail(ctx context.Context, episodeID string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ep, err := tx.EpisodeHead(ctx, episodeID)
		if err != nil {
			return nil, err
		}
		return view.EpisodeChangeThumbnail(ep), nil
	})
	return n
}

func (c *Config) eventDownloadFileAttach(ctx context.Context, infoHash, filePath string) html.Node {
	n, _ := c.withTxR(func(tx *model.TxR) (html.Node, error) {
		dl, err := tx.Download(ctx, infoHash)
		if err != nil {
			return nil, err
		}
		for _, df := range dl.Files() {
			if df.Path() == filePath {
				return view.DownloadFileEpisodesUpdate(df), nil
			}
		}
		return nil, nil
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
