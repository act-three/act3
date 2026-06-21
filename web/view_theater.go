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
	sed := tx.SeriesEdition(sedID)
	editions := tx.SeriesEditionList(sed.SeriesHead())
	return view.BrowseSeriesEdition(sed, editions, tx.Uploads())
}

func viewTheaterMovie(tx *model.TxR, medID string) (title string, n node) {
	med := tx.MovieEdition(medID)
	editions := tx.MovieEditionList(med.MovieHead())
	dls := tx.MovieDownloadList(med)
	var audioOpts []model.AudioOption
	var subOpts []model.SubtitleOption
	if v := med.ActiveVideo(); v != nil {
		audioOpts = tx.AudioOptions(v)
		subOpts = tx.SubtitleOptions(v)
	}
	return view.BrowseMovieEdition(med, editions, dls, audioOpts, subOpts, tx.Uploads())
}

func viewTheaterEpisode(tx *model.TxR, sedID, epID string) (title string, n node) {
	ep := tx.EpisodeInEdition(epID, sedID)
	dls := tx.EpisodeDownloadList(ep)
	var audioOpts []model.AudioOption
	var subOpts []model.SubtitleOption
	if v := ep.ActiveVideo(); v != nil {
		audioOpts = tx.AudioOptions(v)
		subOpts = tx.SubtitleOptions(v)
	}
	return view.BrowseEpisode(ep, dls, audioOpts, subOpts, tx.Uploads())
}

// viewTheaterCollection renders a collection overview or playlist.
func viewTheaterCollection(tx *model.TxR, colID string, playlist bool) (title string, n node) {
	col := tx.Collection(colID)
	itemCount, runtimeMinutes := tx.CollectionStats(col.ID())
	var ps []model.Playable
	if playlist {
		ps = tx.CollectionPlayables(col.ID())
	}
	return view.TheaterCollection(col, itemCount, runtimeMinutes, playlist, ps, tx.Uploads())
}
