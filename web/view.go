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

func (a *app) View(ctx context.Context) (title string, n node) {
	var dlg node
	err := a.model.WithTxR(func(tx *model.TxR) error {
		title, n = a.view(ctx, tx, splitPath(a.path))
		dlg = a.viewDialog(ctx, tx)
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
func (a *app) viewDialog(ctx context.Context, tx *model.TxR) node {
	switch d := a.dialog.(type) {
	case *seriesAddDialog:
		return view.AppSeriesAddDialog(d.query, d.searching, d.results)
	case *movieAddDialog:
		return view.AppMovieAddDialog(d.query, d.searching, d.results)
	case *collectionMovieAddDialog:
		results, err := tx.CollectionMovieSearch(ctx, d.colID, d.query)
		if err != nil {
			return viewError(err)
		}
		return view.AppCollectionMovieAddDialog(d.colID, d.query, results)
	case *collectionSeriesAddDialog:
		results, err := tx.CollectionSeriesSearch(ctx, d.colID, d.query)
		if err != nil {
			return viewError(err)
		}
		return view.AppCollectionSeriesAddDialog(d.colID, d.query, results)
	case *imageDialog:
		return viewImageDialog(ctx, tx, d.id)
	case *downloadFileAttachPopover:
		return viewDownloadFileAttach(ctx, tx, d)
	}
	return nil
}

// viewDownloadFileAttach renders the episode picker for attaching a
// downloaded file, once the attached-set snapshot has arrived.
func viewDownloadFileAttach(ctx context.Context, tx *model.TxR, d *downloadFileAttachPopover) node {
	if !d.ready {
		return nil
	}
	dl, err := tx.Download(ctx, d.infoHash)
	if err != nil {
		return viewError(err)
	}
	sed := dl.SeriesEdition()
	if sed == nil {
		return viewError(fmt.Errorf("download %s is not planned for a series", d.infoHash))
	}
	vid, err := tx.VideoGetByName(ctx, d.infoHash, d.path)
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
func downloadAttachedEpisodes(ctx context.Context, tx *model.TxR, infoHash, path string) ([]string, error) {
	dl, err := tx.Download(ctx, infoHash)
	if err != nil {
		return nil, err
	}
	vid, err := tx.VideoGetByName(ctx, infoHash, path)
	if err != nil {
		return nil, err
	}
	return dl.EpisodeIDsByVideoID(vid.ID()), nil
}

// viewImageDialog renders the image-edit dialog for the item the ID
// identifies.
func viewImageDialog(ctx context.Context, tx *model.TxR, id string) node {
	switch model.KindOf(id) {
	case model.TrashKindMovieEdition:
		med, err := tx.MovieEdition(ctx, id)
		if err != nil {
			return viewError(err)
		}
		return view.AppMoviePosterDialog(med)
	case model.TrashKindSeriesEdition:
		sed, err := tx.SeriesEdition(ctx, id)
		if err != nil {
			return viewError(err)
		}
		return view.AppSeriesEditionPosterDialog(sed)
	case model.TrashKindEpisode:
		ep, err := tx.EpisodeHead(ctx, id)
		if err != nil {
			return viewError(err)
		}
		return view.AppEpisodeThumbnailDialog(ep)
	case model.TrashKindCollection:
		col, err := tx.CollectionHead(ctx, id)
		if err != nil {
			return viewError(err)
		}
		return view.AppCollectionBannerDialog(col)
	}
	return nil
}

func (a *app) Preview(ctx context.Context, u *url.URL) (dest, title string, n node) {
	// TODO: render previews. For now, we just deny all preview requests.
	return "", "", nil
}

func (a *app) view(ctx context.Context, tx *model.TxR, path []string) (title string, n node) {
	if a.odesc != nil {
		return viewObject(ctx, tx, a.path, a.odesc)
	}
	m := &matcher{path: path}
	switch {
	case m.match(""):
		return viewHome(ctx, tx)
	case m.match("collections"):
		return viewCollections(ctx, tx)
	case path[0] == "app":
		title, body := viewEditorPage(ctx, tx, path[1:])
		return title, viewEditor(ctx, tx, a.path, body)
	}
	return "Not Found", notFound
}

// viewObject renders the editor or theater page for odesc.
func viewObject(ctx context.Context, tx *model.TxR, current string, odesc map[string]string) (title string, n node) {
	switch odesc["section"] {
	case sectionEditor:
		title, body := viewEditorObject(ctx, tx, odesc)
		return title, viewEditor(ctx, tx, current, body)
	case sectionTheater:
		return viewTheater(ctx, tx, odesc)
	}
	return "Not Found", notFound
}

func viewHome(ctx context.Context, tx *model.TxR) (title string, n node) {
	works, err := tx.WorkList(ctx)
	if err != nil {
		return "", viewError(err)
	}
	cols, err := tx.CollectionHeadList(ctx)
	if err != nil {
		return "", viewError(err)
	}
	return view.Home(works, cols, tx.Uploads())
}

func viewCollections(ctx context.Context, tx *model.TxR) (title string, n node) {
	cols, err := tx.CollectionHeadList(ctx)
	if err != nil {
		return "", viewError(err)
	}
	return view.Collections(cols, tx.Uploads())
}

func viewError(err error) node {
	return domi.Text("Error: " + err.Error())
}
