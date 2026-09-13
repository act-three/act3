package web

import (
	"strings"

	"ily.dev/act3/buildinfo"
	"ily.dev/act3/hi"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func viewEditor(tx *model.TxR, path hi.RequestPath, odesc map[string]string) hi.View {
	stats := tx.TaskStats()
	return view.Editor(
		viewEditorPage(tx, path, odesc).Class("v-app-page"),
		view.AppConfig{
			Path:           "/" + strings.Join(path, "/"),
			TaskCount:      stats.Queued + stats.Running,
			TaskCountError: stats.CountError,
			Uploads:        tx.Uploads(),
		},
	)
}

func viewEditorPage(tx *model.TxR, path []string, odesc map[string]string) hi.View {
	return hi.First(
		hi.Path("/app/about", viewHTML(viewEditorAbout)),
		hi.Path("/app/profile", viewHTML(view.AppProfile)),
		hi.Path("/app/security", viewHTML(view.AppSecurity)),
		hi.PathPrefix("/app/collections", viewHTML(func() (string, node) {
			return viewEditorCollection(tx, odesc["col"], len(path) > 2 && odesc == nil)
		})),
		hi.PathPrefix("/app/downloads", viewEditorItem(tx, path, viewEditorDownloads)),
		hi.PathPrefix("/app/movies", viewHTML(func() (string, node) {
			return viewEditorMovie(tx, odesc["med"], len(path) > 2 && odesc == nil)
		})),
		hi.PathPrefix("/app/series", viewHTML(func() (string, node) {
			switch odesc["kind"] {
			case model.KindSeriesEdition:
				return viewEditorSeries(tx, odesc["sed"], false)
			case model.KindEpisode:
				return viewEditorEpisode(tx, odesc["sed"], odesc["ep"])
			}
			return viewEditorSeries(tx, "", len(path) > 2)
		})),
		hi.Path("/app/storage", viewHTML(func() (string, node) {
			return viewEditorStorage(tx)
		})),
		hi.Path("/app/tasks", viewHTML(func() (string, node) {
			return viewEditorTasks(tx)
		})),
		hi.Path("/app/tmdb", viewHTML(func() (string, node) {
			return view.AppTMDB(tx.SettingGetByGroup("tmdb"))
		})),
		hi.Path("/app/transmission", viewHTML(func() (string, node) {
			return view.AppTransmission(tx.SettingGetByGroup("transmission"))
		})),
		hi.PathPrefix("/app/trash", viewEditorItem(tx, path, viewEditorTrash)),
		hi.HTML(notFound),
	)
}

// Downloads and trash accept a list path or one item ID, never a subtree.
func viewEditorItem(tx *model.TxR, path []string, f func(*model.TxR, string) (string, node)) hi.View {
	return viewHTML(func() (string, node) {
		switch len(path) {
		case 2:
			return f(tx, "")
		case 3:
			return f(tx, path[2])
		}
		return "", notFound
	})
}

func viewEditorAbout() (title string, n node) {
	return view.AppAbout(buildinfo.Get())
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
