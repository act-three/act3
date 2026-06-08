package web

import (
	"context"
	"database/sql"

	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func viewEditor(ctx context.Context, tx *model.TxR, current string, body node) node {
	stats, _ := tx.TaskStats(ctx)
	return view.Editor(body, view.AppConfig{
		Path:           current,
		TaskCount:      stats.Queued + stats.Running,
		TaskCountError: stats.CountError,
		Uploads:        tx.Uploads(),
	})
}

func viewEditorPage(ctx context.Context, tx *model.TxR, path []string) (title string, n node) {
	m := &matcher{path: path}
	switch {
	case m.match(""):
		return "", notFound
	case m.match("profile"):
		return view.AppProfile()
	case m.match("security"):
		return view.AppSecurity()
	case m.match("collections"):
		return viewEditorCollection(ctx, tx, "")
	case m.match("downloads"):
		return viewEditorDownloads(ctx, tx, "")
	case m.match("downloads/{id}"):
		return viewEditorDownloads(ctx, tx, m.get("id"))

	case m.match("movies"):
		return viewEditorMovie(ctx, tx, "")
	case m.match("series"):
		return viewEditorSeries(ctx, tx, "")

	case m.match("storage"):
		return viewEditorStorage(ctx, tx)
	case m.match("tasks"):
		return viewEditorTasks(ctx, tx)
	case m.match("tmdb"):
		settings, err := tx.SettingGetByGroup(ctx, "tmdb")
		if err != nil {
			return "", viewError(err)
		}
		return view.AppTMDB(settings)
	case m.match("transmission"):
		settings, err := tx.SettingGetByGroup(ctx, "transmission")
		if err != nil {
			return "", viewError(err)
		}
		return view.AppTransmission(settings)
	case m.match("trash"):
		return viewEditorTrash(ctx, tx, "")
	case m.match("trash/{id}"):
		return viewEditorTrash(ctx, tx, m.get("id"))
	}
	return "", notFound
}

// viewEditorObject renders the editor page for odesc.
func viewEditorObject(ctx context.Context, tx *model.TxR, odesc map[string]string) (title string, n node) {
	switch odesc["kind"] {
	case model.KindMovieEdition:
		return viewEditorMovie(ctx, tx, odesc["med"])
	case model.KindSeriesEdition:
		return viewEditorSeries(ctx, tx, odesc["sed"])
	case model.KindEpisode:
		return viewEditorEpisode(ctx, tx, odesc["sed"], odesc["ep"])
	case model.KindCollectionOverview:
		return viewEditorCollection(ctx, tx, odesc["col"])
	}
	return "", notFound
}

func viewEditorCollection(ctx context.Context, tx *model.TxR, colID string) (title string, n node) {
	var selected *model.Collection
	if colID != "" {
		col, err := tx.Collection(ctx, colID)
		if err == sql.ErrNoRows {
			return "Collections", notFound
		} else if err != nil {
			return "", viewError(err)
		}
		selected = col
	}

	all, err := tx.CollectionHeadList(ctx)
	if err != nil {
		return "", viewError(err)
	}
	return view.AppCollections(all, selected)
}

func viewEditorDownloads(ctx context.Context, tx *model.TxR, id string) (title string, n node) {
	var selected *model.Download
	if id != "" {
		dl, err := tx.Download(ctx, id)
		if err == sql.ErrNoRows {
			return "Downloads", notFound
		} else if err != nil {
			return "", viewError(err)
		}
		selected = dl
	}
	dls, err := tx.DownloadInfoList(ctx)
	if err != nil {
		return "", viewError(err)
	}
	return view.AppDownloads(dls, selected)
}

func viewEditorTrash(ctx context.Context, tx *model.TxR, id string) (title string, n node) {
	var selected *model.TrashItem
	if id != "" {
		it, err := tx.TrashItem(ctx, id)
		if err == sql.ErrNoRows {
			return "Trash", notFound
		} else if err != nil {
			return "", viewError(err)
		}
		selected = &it
	}
	items, err := tx.TrashList(ctx)
	if err != nil {
		return "", viewError(err)
	}
	return view.AppTrash(items, selected)
}

func viewEditorTasks(ctx context.Context, tx *model.TxR) (title string, n node) {
	running := tx.RunningTasks()
	tasks, err := tx.TaskList(ctx)
	if err != nil {
		return "", viewError(err)
	}
	var queued, failed []*model.Task
	for _, t := range tasks {
		if t.Failed() {
			failed = append(failed, t)
		} else {
			queued = append(queued, t)
		}
	}
	return view.AppTasks(running, queued, failed)
}

func viewEditorMovie(ctx context.Context, tx *model.TxR, medID string) (title string, n node) {
	var selected *model.MovieEdition
	var editions []*model.MovieWork
	var dls []*model.DownloadHead
	if medID != "" {
		med, err := tx.MovieEdition(ctx, medID)
		if err == sql.ErrNoRows {
			return "All Movies", notFound
		} else if err != nil {
			return "", viewError(err)
		}
		editions, err = tx.MovieEditionList(ctx, med.MovieHead())
		if err != nil {
			return "", viewError(err)
		}
		dls, err = tx.DownloadHeadListByMovieEditionID(ctx, med.ID())
		if err != nil {
			return "", viewError(err)
		}
		selected = med
	}
	all, err := tx.MovieWorkList(ctx)
	if err != nil {
		return "", viewError(err)
	}
	return view.AppMovies(all, selected, editions, dls, tx.Uploads())
}

func viewEditorSeries(ctx context.Context, tx *model.TxR, sedID string) (title string, n node) {
	var selected *model.SeriesEdition
	var editions []*model.SeriesWork
	var dls []*model.DownloadHead
	if sedID != "" {
		sed, err := tx.SeriesEdition(ctx, sedID)
		if err == sql.ErrNoRows {
			return "Edit Series", notFound
		} else if err != nil {
			return "", viewError(err)
		}
		editions, err = tx.SeriesEditionList(ctx, sed.SeriesHead())
		if err != nil {
			return "", viewError(err)
		}
		dls, err = tx.DownloadHeadListBySeriesEditionID(ctx, sed.ID())
		if err != nil {
			return "", viewError(err)
		}
		selected = sed
	}
	all, err := tx.SeriesWorkList(ctx)
	if err != nil {
		return "", viewError(err)
	}
	return view.AppSeries(all, selected, editions, dls)
}

func viewEditorEpisode(ctx context.Context, tx *model.TxR, sedID, epID string) (title string, n node) {
	ep, err := tx.EpisodeInEdition(ctx, epID, sedID)
	if err == sql.ErrNoRows {
		return "Edit Series", notFound
	} else if err != nil {
		return "", viewError(err)
	}
	renditions, err := tx.RenditionListStreamingByEpisodeID(ctx, ep.ID())
	if err != nil {
		return "", viewError(err)
	}
	episodeEditions, err := tx.EpisodeEditions(ctx, ep.ID())
	if err != nil {
		return "", viewError(err)
	}
	all, err := tx.SeriesWorkList(ctx)
	if err != nil {
		return "", viewError(err)
	}
	return view.AppSeriesEpisode(all, ep, episodeEditions, renditions, tx.Uploads())
}
