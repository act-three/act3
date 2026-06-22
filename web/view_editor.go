package web

import (
	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func viewEditor(tx *model.TxR, path string, body node) node {
	stats := tx.TaskStats()
	return view.Editor(body, view.AppConfig{
		Path:           path,
		TaskCount:      stats.Queued + stats.Running,
		TaskCountError: stats.CountError,
		Uploads:        tx.Uploads(),
	})
}

func viewEditorPage(tx *model.TxR, path []string, odesc map[string]string) (title string, n node) {
	m := &matcher{path: path}
	switch {
	case m.match(""):
		return "", notFound
	case m.match("profile"):
		return view.AppProfile()
	case m.match("security"):
		return view.AppSecurity()
	case m.match("collections"):
		return viewEditorCollection(tx, "", false)
	case path[0] == "collections":
		return viewEditorCollection(tx, odesc["col"], odesc == nil)
	case m.match("downloads"):
		return viewEditorDownloads(tx, "")
	case m.match("downloads/{id}"):
		return viewEditorDownloads(tx, m.get("id"))

	case m.match("movies"):
		return viewEditorMovie(tx, "", false)
	case path[0] == "movies":
		return viewEditorMovie(tx, odesc["med"], odesc == nil)
	case m.match("series"):
		return viewEditorSeries(tx, "", false)
	case path[0] == "series":
		switch odesc["kind"] {
		case model.KindSeriesEdition:
			return viewEditorSeries(tx, odesc["sed"], false)
		case model.KindEpisode:
			return viewEditorEpisode(tx, odesc["sed"], odesc["ep"])
		}
		return viewEditorSeries(tx, "", true)

	case m.match("storage"):
		return viewEditorStorage(tx)
	case m.match("tasks"):
		return viewEditorTasks(tx)
	case m.match("tmdb"):
		return view.AppTMDB(tx.SettingGetByGroup("tmdb"))
	case m.match("transmission"):
		return view.AppTransmission(tx.SettingGetByGroup("transmission"))
	case m.match("trash"):
		return viewEditorTrash(tx, "")
	case m.match("trash/{id}"):
		return viewEditorTrash(tx, m.get("id"))
	}
	return "", notFound
}

func viewEditorCollection(tx *model.TxR, id string, notFound bool) (title string, n node) {
	cols := tx.CollectionHeadList()
	var selected *model.Collection
	if id != "" {
		selected = tx.Collection(id)
	}
	return view.AppCollections(cols, selected, notFound)
}

func viewEditorDownloads(tx *model.TxR, id string) (title string, n node) {
	dls := tx.DownloadInfoList()
	var selected *model.Download
	found := true
	if id != "" {
		selected, found = tx.FindDownload(id)
	}
	return view.AppDownloads(dls, selected, !found)
}

func viewEditorTrash(tx *model.TxR, id string) (title string, n node) {
	items := tx.TrashList()
	var selected *model.TrashItem
	found := true
	if id != "" {
		selected, found = tx.FindTrashItem(id)
	}
	return view.AppTrash(items, selected, !found)
}

func viewEditorTasks(tx *model.TxR) (title string, n node) {
	running := tx.RunningTasks()
	tasks := tx.TaskList()
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

func viewEditorMovie(tx *model.TxR, medID string, notFound bool) (title string, n node) {
	movies := tx.MovieWorkList()
	var selected *model.MovieEdition
	var editions []*model.MovieWork
	var dls []*model.DownloadHead
	if medID != "" {
		med := tx.MovieEdition(medID)
		editions = tx.MovieEditionList(med.MovieHead())
		dls = tx.DownloadHeadListByMovieEditionID(med.ID())
		selected = med
	}
	return view.AppMovies(movies, selected, editions, dls, tx.Uploads(), notFound)
}

func viewEditorSeries(tx *model.TxR, sedID string, notFound bool) (title string, n node) {
	series := tx.SeriesWorkList()
	var selected *model.SeriesEdition
	var editions []*model.SeriesWork
	var dls []*model.DownloadHead
	if sedID != "" {
		sed := tx.SeriesEdition(sedID)
		editions = tx.SeriesEditionList(sed.SeriesHead())
		dls = tx.DownloadHeadListBySeriesEditionID(sed.ID())
		selected = sed
	}
	return view.AppSeries(series, selected, editions, dls, notFound)
}

func viewEditorEpisode(tx *model.TxR, sedID, epID string) (title string, n node) {
	series := tx.SeriesWorkList()
	ep := tx.EpisodeInEdition(epID, sedID)
	renditions := tx.RenditionListStreamingByEpisodeID(ep.ID())
	episodeEditions := tx.EpisodeEditions(ep.ID())
	return view.AppSeriesEpisode(series, ep, episodeEditions, renditions, tx.Uploads())
}
