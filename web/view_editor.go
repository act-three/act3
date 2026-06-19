package web

import (
	"database/sql"

	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func viewEditor(tx *model.TxR, current string, body node) node {
	stats, _ := tx.TaskStats()
	return view.Editor(body, view.AppConfig{
		Path:           current,
		TaskCount:      stats.Queued + stats.Running,
		TaskCountError: stats.CountError,
		Uploads:        tx.Uploads(),
	})
}

func viewEditorPage(tx *model.TxR, path []string) (title string, n node) {
	m := &matcher{path: path}
	switch {
	case m.match(""):
		return "", notFound
	case m.match("profile"):
		return view.AppProfile()
	case m.match("security"):
		return view.AppSecurity()
	case m.match("collections"):
		return viewEditorCollection(tx, "")
	case m.match("downloads"):
		return viewEditorDownloads(tx, "")
	case m.match("downloads/{id}"):
		return viewEditorDownloads(tx, m.get("id"))

	case m.match("movies"):
		return viewEditorMovie(tx, "")
	case m.match("series"):
		return viewEditorSeries(tx, "")

	case m.match("storage"):
		return viewEditorStorage(tx)
	case m.match("tasks"):
		return viewEditorTasks(tx)
	case m.match("tmdb"):
		settings, err := tx.SettingGetByGroup("tmdb")
		if err != nil {
			return "", viewError(err)
		}
		return view.AppTMDB(settings)
	case m.match("transmission"):
		settings, err := tx.SettingGetByGroup("transmission")
		if err != nil {
			return "", viewError(err)
		}
		return view.AppTransmission(settings)
	case m.match("trash"):
		return viewEditorTrash(tx, "")
	case m.match("trash/{id}"):
		return viewEditorTrash(tx, m.get("id"))
	}
	return "", notFound
}

// viewEditorObject renders the editor page for odesc.
func viewEditorObject(tx *model.TxR, odesc map[string]string) (title string, n node) {
	switch odesc["kind"] {
	case model.KindMovieEdition:
		return viewEditorMovie(tx, odesc["med"])
	case model.KindSeriesEdition:
		return viewEditorSeries(tx, odesc["sed"])
	case model.KindEpisode:
		return viewEditorEpisode(tx, odesc["sed"], odesc["ep"])
	case model.KindCollectionOverview:
		return viewEditorCollection(tx, odesc["col"])
	}
	return "", notFound
}

func viewEditorCollection(tx *model.TxR, colID string) (title string, n node) {
	var selected *model.Collection
	if colID != "" {
		col, err := tx.Collection(colID)
		if err == sql.ErrNoRows {
			return "Collections", notFound
		} else if err != nil {
			return "", viewError(err)
		}
		selected = col
	}

	all, err := tx.CollectionHeadList()
	if err != nil {
		return "", viewError(err)
	}
	return view.AppCollections(all, selected)
}

func viewEditorDownloads(tx *model.TxR, id string) (title string, n node) {
	var selected *model.Download
	if id != "" {
		dl, err := tx.Download(id)
		if err == sql.ErrNoRows {
			return "Downloads", notFound
		} else if err != nil {
			return "", viewError(err)
		}
		selected = dl
	}
	dls, err := tx.DownloadInfoList()
	if err != nil {
		return "", viewError(err)
	}
	return view.AppDownloads(dls, selected)
}

func viewEditorTrash(tx *model.TxR, id string) (title string, n node) {
	var selected *model.TrashItem
	if id != "" {
		it, err := tx.TrashItem(id)
		if err == sql.ErrNoRows {
			return "Trash", notFound
		} else if err != nil {
			return "", viewError(err)
		}
		selected = &it
	}
	items, err := tx.TrashList()
	if err != nil {
		return "", viewError(err)
	}
	return view.AppTrash(items, selected)
}

func viewEditorTasks(tx *model.TxR) (title string, n node) {
	running := tx.RunningTasks()
	tasks, err := tx.TaskList()
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

func viewEditorMovie(tx *model.TxR, medID string) (title string, n node) {
	var selected *model.MovieEdition
	var editions []*model.MovieWork
	var dls []*model.DownloadHead
	if medID != "" {
		med, err := tx.MovieEdition(medID)
		if err == sql.ErrNoRows {
			return "All Movies", notFound
		} else if err != nil {
			return "", viewError(err)
		}
		editions, err = tx.MovieEditionList(med.MovieHead())
		if err != nil {
			return "", viewError(err)
		}
		dls, err = tx.DownloadHeadListByMovieEditionID(med.ID())
		if err != nil {
			return "", viewError(err)
		}
		selected = med
	}
	all, err := tx.MovieWorkList()
	if err != nil {
		return "", viewError(err)
	}
	return view.AppMovies(all, selected, editions, dls, tx.Uploads())
}

func viewEditorSeries(tx *model.TxR, sedID string) (title string, n node) {
	var selected *model.SeriesEdition
	var editions []*model.SeriesWork
	var dls []*model.DownloadHead
	if sedID != "" {
		sed, err := tx.SeriesEdition(sedID)
		if err == sql.ErrNoRows {
			return "Edit Series", notFound
		} else if err != nil {
			return "", viewError(err)
		}
		editions, err = tx.SeriesEditionList(sed.SeriesHead())
		if err != nil {
			return "", viewError(err)
		}
		dls, err = tx.DownloadHeadListBySeriesEditionID(sed.ID())
		if err != nil {
			return "", viewError(err)
		}
		selected = sed
	}
	all, err := tx.SeriesWorkList()
	if err != nil {
		return "", viewError(err)
	}
	return view.AppSeries(all, selected, editions, dls)
}

func viewEditorEpisode(tx *model.TxR, sedID, epID string) (title string, n node) {
	ep, err := tx.EpisodeInEdition(epID, sedID)
	if err == sql.ErrNoRows {
		return "Edit Series", notFound
	} else if err != nil {
		return "", viewError(err)
	}
	renditions, err := tx.RenditionListStreamingByEpisodeID(ep.ID())
	if err != nil {
		return "", viewError(err)
	}
	episodeEditions, err := tx.EpisodeEditions(ep.ID())
	if err != nil {
		return "", viewError(err)
	}
	all, err := tx.SeriesWorkList()
	if err != nil {
		return "", viewError(err)
	}
	return view.AppSeriesEpisode(all, ep, episodeEditions, renditions, tx.Uploads())
}
