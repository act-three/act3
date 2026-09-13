package web

import (
	"context"
	"fmt"
	"net/url"

	"ily.dev/act3/hi"
	"ily.dev/act3/model"
	"ily.dev/act3/model/kind"
	"ily.dev/act3/ui"
	"ily.dev/act3/view"
	"ily.dev/domi"
)

type node = domi.Node

var notFound = domi.Text("Not Found")

func (a *app) Preview(ctx context.Context, u *url.URL, render hi.PreviewRenderer) (p hi.Preview) {
	a = new(*a)
	u = u.Clone()
	u.Path = redirect(u.Path)
	a.notes = nil
	a.setPath(ctx, u)
	a.render(ctx, func(v hi.View) {
		p = render(u.String(), v)
	})
	return p
}

func (a *app) View(ctx context.Context, render hi.PageRenderer) (p hi.Page) {
	a.render(ctx, func(v hi.View) {
		p = render(v)
	})
	return p
}

// Rendering resolves lazy views, which may still need the transaction.
func (a *app) render(ctx context.Context, render func(hi.View)) {
	var dlg node
	err := a.model.WithTxR(ctx, func(tx *model.TxR) error {
		v := viewRoot(tx)
		dlg = viewDialog(tx, a.dialog)
		render(a.view(v, dlg))
		return nil
	})
	if err != nil {
		render(a.view(
			hi.ScrollView(hi.Vertical,
				hi.HTML(viewError(err)).Class("v-domi-root"),
			).
				Title("Error"),
			dlg,
		))
	}
}

func (a *app) view(v hi.View, dlg node) hi.View {
	n := domi.Fragment(
		dlg,
		ui.NotePort(a.notes),
		view.PlayerContainer(a.viewPlayer(a.player)),
	)
	// A root overlay preserves the theater's document scrolling and
	// keeps these fixed-position containers out of the page's layout.
	return v.Overlay(hi.Center,
		hi.HTML(n).
			Class("v-domi-root").
			FixedSize(),
	)
}

func viewRoot(tx *model.TxR) hi.View {
	return hi.PathReader(func(path hi.RequestPath) hi.View {
		odesc, _ := resolve(tx, path)
		return hi.First(
			hi.PathPrefix("/app", hi.Lazy(func() hi.View {
				return viewEditor(tx, path, odesc)
			})),
			hi.ScrollView(hi.Vertical, hi.First(
				hi.Path("/", viewHTML(func() (string, node) {
					return viewHome(tx)
				})),
				hi.Path("/collections", viewHTML(func() (string, node) {
					return viewCollections(tx)
				})),
				viewHTML(func() (string, node) {
					return viewTheater(tx, odesc)
				}),
			).Class("v-domi-root")),
		)
	})
}

// viewHTML adapts legacy pages, keeping their queries lazy and their
// titles attached to the selected view during the migration to hi.
func viewHTML(f func() (string, node)) hi.View {
	return hi.Lazy(func() hi.View {
		title, n := f()
		return hi.HTML(n).Title(title)
	})
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
		results := tx.CollectionMovieSearch(d.colID, d.query)
		return view.AppCollectionMovieAddDialog(d.colID, d.query, results)
	case *collectionSeriesAddDialog:
		results := tx.CollectionSeriesSearch(d.colID, d.query)
		return view.AppCollectionSeriesAddDialog(d.colID, d.query, results)
	case *imageDialog:
		return view.AppImageDialog(d.kind, d.id, dialogImage(tx, d.kind, d.id))
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
	dl := tx.Download(d.infoHash)
	sed := dl.SeriesEdition()
	if sed == nil {
		// TODO: viewError doesn't help in this context,
		// we need an actual dialog or nothing.
		// Ideally we use a TxR method to abort the tx
		// if there's no SeriesEdition here.
		return viewError(fmt.Errorf("download %s is not planned for a series", d.infoHash))
	}
	vid := tx.VideoGetByName(d.infoHash, d.path)
	linked := map[string]bool{}
	for _, epID := range dl.EpisodeIDsByVideoID(vid.ID()) {
		linked[epID] = true
	}
	return view.AppDownloadFileAttachPopover(sed, d.infoHash, d.path, vid.ID(), d.attached, linked)
}

// dialogImage returns the current image for the item the ID
// identifies; k selects how to resolve it.
func dialogImage(tx *model.TxR, k kind.ImageOwner, id string) model.Image {
	switch k.(type) {
	case kind.MovieEdition:
		return tx.MovieEdition(id).Poster()
	case kind.SeriesEdition:
		return tx.SeriesEdition(id).Poster()
	case kind.Episode:
		return tx.EpisodeHead(id).Thumbnail()
	case kind.Collection:
		return tx.CollectionHead(id).Banner()
	}
	panic(fmt.Sprintf("unknown image dialog kind %v", k))
}

func viewHome(tx *model.TxR) (title string, n node) {
	works := tx.WorkList()
	cols := tx.CollectionHeadList()
	return view.Home(works, cols, tx.Uploads())
}

func viewCollections(tx *model.TxR) (title string, n node) {
	cols := tx.CollectionHeadList()
	return view.Collections(cols, tx.Uploads())
}

func viewError(err error) node {
	return domi.Text("Error: " + err.Error())
}
