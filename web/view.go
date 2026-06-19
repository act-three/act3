package web

import (
	"context"
	"fmt"
	"net/url"

	"ily.dev/act3/model"
	"ily.dev/act3/ui"
	"ily.dev/act3/view"
	"ily.dev/domi"
)

type node = domi.Node

var notFound = domi.Text("Not Found")

func (a *app) Preview(ctx context.Context, u *url.URL) (dest, title string, n node) {
	// TODO: render previews. For now, we just deny all preview requests.
	return "", "", nil
}

func (a *app) View(ctx context.Context) (title string, n node) {
	var dlg node
	err := a.model.WithTxR(ctx, func(tx *model.TxR) error {
		title, n = viewRoot(tx, a.path, a.odesc)
		dlg = viewDialog(tx, a.dialog)
		return nil
	})
	if err != nil {
		n = viewError(err)
	}
	n = domi.Fragment(n,
		dlg,
		ui.NotePort(a.notes),
		view.PlayerContainer(a.viewPlayer(a.player)),
	)
	if title == "" {
		return "Act Three", n
	}
	return title + " — Act Three", n
}

func viewRoot(tx *model.TxR, path string, odesc map[string]string) (title string, n node) {
	if odesc != nil {
		return viewObject(tx, path, odesc)
	}
	m := &matcher{path: splitPath(path)}
	switch {
	case m.match(""):
		return viewHome(tx)
	case m.match("collections"):
		return viewCollections(tx)
	case m.path[0] == "app":
		title, body := viewEditorPage(tx, m.path[1:])
		return title, viewEditor(tx, path, body)
	}
	return "Not Found", notFound
}

// viewPlayer renders the open player, or an empty slot when none is open.
func (a *app) viewPlayer(ps *player) (string, node) {
	if ps == nil {
		return "", nil
	}
	var inner node
	if ps.episode != nil {
		inner = view.PlayerForEpisode(ps.video, ps.episode, ps.qualityOpts, ps.captionsOpts, ps.audioOpts, ps.audio, ps.subtitle, ps.pinAudio)
	} else {
		inner = view.PlayerForMovie(ps.video, ps.movie, ps.qualityOpts, ps.captionsOpts, ps.audioOpts, ps.audio, ps.subtitle, ps.pinAudio)
	}
	return ps.video.ID(), inner
}

// viewDialog renders the open dialog, if any.
func viewDialog(tx *model.TxR, d dialog) node {
	switch d := d.(type) {
	case *seriesAddDialog:
		return view.AppSeriesAddDialog(d.query, d.searching, d.results)
	case *movieAddDialog:
		return view.AppMovieAddDialog(d.query, d.searching, d.results)
	case *collectionMovieAddDialog:
		results, err := tx.CollectionMovieSearch(d.colID, d.query)
		if err != nil {
			return viewError(err)
		}
		return view.AppCollectionMovieAddDialog(d.colID, d.query, results)
	case *collectionSeriesAddDialog:
		results, err := tx.CollectionSeriesSearch(d.colID, d.query)
		if err != nil {
			return viewError(err)
		}
		return view.AppCollectionSeriesAddDialog(d.colID, d.query, results)
	case *imageDialog:
		return viewImageDialog(tx, d.id)
	case *downloadFileAttachPopover:
		return viewDownloadFileAttach(tx, d)
	}
	return nil
}

// viewDownloadFileAttach renders the episode picker for attaching a
// downloaded file, once the attached-set snapshot has arrived.
func viewDownloadFileAttach(tx *model.TxR, d *downloadFileAttachPopover) node {
	if !d.ready {
		return nil
	}
	dl, err := tx.Download(d.infoHash)
	if err != nil {
		return viewError(err)
	}
	sed := dl.SeriesEdition()
	if sed == nil {
		return viewError(fmt.Errorf("download %s is not planned for a series", d.infoHash))
	}
	vid, err := tx.VideoGetByName(d.infoHash, d.path)
	if err != nil {
		return viewError(err)
	}
	linked := map[string]bool{}
	for _, epID := range dl.EpisodeIDsByVideoID(vid.ID()) {
		linked[epID] = true
	}
	return view.AppDownloadFileAttachPopover(sed, d.infoHash, d.path, vid.ID(), d.attached, linked)
}

// downloadAttachedEpisodes lists the episodes the downloaded file is
// currently attached to.
func downloadAttachedEpisodes(tx *model.TxR, infoHash, path string) ([]string, error) {
	dl, err := tx.Download(infoHash)
	if err != nil {
		return nil, err
	}
	vid, err := tx.VideoGetByName(infoHash, path)
	if err != nil {
		return nil, err
	}
	return dl.EpisodeIDsByVideoID(vid.ID()), nil
}

// viewImageDialog renders the image-edit dialog for the item the ID
// identifies.
func viewImageDialog(tx *model.TxR, id string) node {
	switch model.KindOf(id) {
	case model.TrashKindMovieEdition:
		med, err := tx.MovieEdition(id)
		if err != nil {
			return viewError(err)
		}
		return view.AppMoviePosterDialog(med)
	case model.TrashKindSeriesEdition:
		sed, err := tx.SeriesEdition(id)
		if err != nil {
			return viewError(err)
		}
		return view.AppSeriesEditionPosterDialog(sed)
	case model.TrashKindEpisode:
		ep, err := tx.EpisodeHead(id)
		if err != nil {
			return viewError(err)
		}
		return view.AppEpisodeThumbnailDialog(ep)
	case model.TrashKindCollection:
		col, err := tx.CollectionHead(id)
		if err != nil {
			return viewError(err)
		}
		return view.AppCollectionBannerDialog(col)
	}
	return nil
}

// viewObject renders the editor or theater page for odesc.
func viewObject(tx *model.TxR, current string, odesc map[string]string) (title string, n node) {
	switch odesc["section"] {
	case sectionEditor:
		title, body := viewEditorObject(tx, odesc)
		return title, viewEditor(tx, current, body)
	case sectionTheater:
		return viewTheater(tx, odesc)
	}
	return "Not Found", notFound
}

func viewHome(tx *model.TxR) (title string, n node) {
	works, err := tx.WorkList()
	if err != nil {
		return "", viewError(err)
	}
	cols, err := tx.CollectionHeadList()
	if err != nil {
		return "", viewError(err)
	}
	return view.Home(works, cols, tx.Uploads())
}

func viewCollections(tx *model.TxR) (title string, n node) {
	cols, err := tx.CollectionHeadList()
	if err != nil {
		return "", viewError(err)
	}
	return view.Collections(cols, tx.Uploads())
}

func viewError(err error) node {
	return domi.Text("Error: " + err.Error())
}
