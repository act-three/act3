package web

import (
	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

// viewTheater renders a theater object page from its descriptor.
func viewTheater(tx *model.TxR, odesc map[string]string) (title string, n node) {
	switch odesc["kind"] {
	case model.KindMovieEdition:
		return viewTheaterMovie(tx, odesc["med"])
	case model.KindSeriesEdition:
		return viewTheaterSeries(tx, odesc["sed"])
	case model.KindEpisode:
		return viewTheaterEpisode(tx, odesc["sed"], odesc["ep"])
	case model.KindCollectionOverview:
		return viewTheaterCollection(tx, odesc["col"], false)
	case model.KindCollectionPlaylist:
		return viewTheaterCollection(tx, odesc["col"], true)
	}
	return "Not Found", notFound
}

func viewTheaterSeries(tx *model.TxR, sedID string) (title string, n node) {
	sed, err := tx.SeriesEdition(sedID)
	if err != nil {
		return "Not Found", notFound
	}
	editions, err := tx.SeriesEditionList(sed.SeriesHead())
	if err != nil {
		return "", viewError(err)
	}
	return view.BrowseSeriesEdition(sed, editions, tx.Uploads())
}

func viewTheaterMovie(tx *model.TxR, medID string) (title string, n node) {
	med, err := tx.MovieEdition(medID)
	if err != nil {
		return "Not Found", notFound
	}
	editions, err := tx.MovieEditionList(med.MovieHead())
	if err != nil {
		return "", viewError(err)
	}
	dls, err := tx.MovieDownloadList(med)
	if err != nil {
		return "", viewError(err)
	}
	var audioOpts []model.AudioOption
	var subOpts []model.SubtitleOption
	if v := med.ActiveVideo(); v != nil {
		audioOpts, err = tx.AudioOptions(v)
		if err != nil {
			return "", viewError(err)
		}
		subOpts, err = tx.SubtitleOptions(v)
		if err != nil {
			return "", viewError(err)
		}
	}
	return view.BrowseMovieEdition(med, editions, dls, audioOpts, subOpts, tx.Uploads())
}

func viewTheaterEpisode(tx *model.TxR, sedID, epID string) (title string, n node) {
	ep, err := tx.EpisodeInEdition(epID, sedID)
	if err != nil {
		return "Not Found", notFound
	}
	dls, err := tx.EpisodeDownloadList(ep)
	if err != nil {
		return "", viewError(err)
	}
	var audioOpts []model.AudioOption
	var subOpts []model.SubtitleOption
	if v := ep.ActiveVideo(); v != nil {
		audioOpts, err = tx.AudioOptions(v)
		if err != nil {
			return "", viewError(err)
		}
		subOpts, err = tx.SubtitleOptions(v)
		if err != nil {
			return "", viewError(err)
		}
	}
	return view.BrowseEpisode(ep, dls, audioOpts, subOpts, tx.Uploads())
}

// viewTheaterCollection renders a collection overview or playlist.
func viewTheaterCollection(tx *model.TxR, colID string, playlist bool) (title string, n node) {
	col, err := tx.Collection(colID)
	if err != nil {
		return "Not Found", notFound
	}
	itemCount, runtimeMinutes, err := tx.CollectionStats(col.ID())
	if err != nil {
		return "", viewError(err)
	}
	var ps []model.Playable
	if playlist {
		ps, err = tx.CollectionPlayables(col.ID())
		if err != nil {
			return "", viewError(err)
		}
	}
	return view.TheaterCollection(col, itemCount, runtimeMinutes, playlist, ps, tx.Uploads())
}
