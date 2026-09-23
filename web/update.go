package web

import (
	"context"
	"log/slog"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"ily.dev/domi"

	"ily.dev/act3/hi"
	"ily.dev/act3/model"
	"ily.dev/act3/msg"
	"ily.dev/act3/ui"
)

type cmd = domi.Cmd[msg.Msg]

func (a *app) Update(ctx context.Context, m msg.Msg) cmd {
	// Notes delivered in the previous frame have been cloned into the
	// client-owned port by now; drop them so each note's outbox entry
	// lives for a single frame. See view.Notes.
	a.notes = nil

	switch m := m.(type) {
	case *msg.URLChange:
		a.setPath(ctx, m.URL)
		// The path's slugs may be stale — resolved through a tombstone
		// after a rename — so canonicalize in place.
		return a.replaceURL(ctx)
	case *msg.URLRequest:
		if !m.Internal {
			return domi.Load[msg.Msg](m.URL.String())
		}
		m.URL.Path = redirect(m.URL.Path)
		return domi.PushURL[msg.Msg](m.URL.String())
	case *msg.ModelEvent:
		return a.replaceURL(ctx)
	case *msg.Error:
		return notifyError(m.Err)

	case *msg.DialogClose:
		a.dialog = nil
		return nil
	case *msg.SeriesAddOpen:
		a.dialog = &seriesAddDialog{}
		return nil
	case *msg.SeriesSearchEdit:
		if d, ok := a.dialog.(*seriesAddDialog); ok {
			d.query = m.Query
		}
		return nil
	case *msg.SeriesSearch:
		d, ok := a.dialog.(*seriesAddDialog)
		if !ok {
			return nil
		}
		d.query = m.Query
		d.searching = true
		return domi.Func(func() msg.Msg {
			results, err := a.model.SearchSeries(ctx, m.Query)
			if err != nil {
				return &msg.SeriesSearchError{Query: m.Query, Err: err}
			}
			return &msg.SeriesSearched{Query: m.Query, Results: results}
		})
	case *msg.SeriesSearched:
		// Drop results that arrive after the dialog closed or the
		// query moved on.
		if d, ok := a.dialog.(*seriesAddDialog); ok && d.query == m.Query {
			d.searching = false
			d.results = m.Results
		}
		return nil
	case *msg.SeriesSearchError:
		// As with SeriesSearched, an error from an abandoned search
		// is dropped.
		if d, ok := a.dialog.(*seriesAddDialog); ok && d.query == m.Query {
			d.searching = false
			return notifyError(m.Err)
		}
		return nil
	case *msg.SeriesAdd:
		return domi.Func(func() msg.Msg {
			sw, err := a.model.AddSeriesByTVmazeID(ctx, m.TVmazeID)
			if err != nil {
				return &msg.Error{Err: err}
			}
			return &msg.SeriesAdded{TVmazeID: m.TVmazeID, Local: &sw.SeriesHead}
		})
	case *msg.SeriesAdded:
		// Mark the matching search result as in the library; the
		// series list itself updates with the re-render.
		if d, ok := a.dialog.(*seriesAddDialog); ok {
			for i := range d.results {
				if d.results[i].Show.ID == m.TVmazeID {
					d.results[i].Local = m.Local
				}
			}
		}
		return nil

	case *msg.MovieAddOpen:
		a.dialog = &movieAddDialog{}
		return nil
	case *msg.MovieSearchEdit:
		if d, ok := a.dialog.(*movieAddDialog); ok {
			d.query = m.Query
		}
		return nil
	case *msg.MovieSearch:
		d, ok := a.dialog.(*movieAddDialog)
		if !ok {
			return nil
		}
		d.query = m.Query
		d.searching = true
		return domi.Func(func() msg.Msg {
			results, err := a.model.SearchMovies(ctx, m.Query)
			if err != nil {
				return &msg.MovieSearchError{Query: m.Query, Err: err}
			}
			return &msg.MovieSearched{Query: m.Query, Results: results}
		})
	case *msg.MovieSearched:
		// Drop results that arrive after the dialog closed or the
		// query moved on.
		if d, ok := a.dialog.(*movieAddDialog); ok && d.query == m.Query {
			d.searching = false
			d.results = m.Results
		}
		return nil
	case *msg.MovieSearchError:
		// As with MovieSearched, an error from an abandoned search
		// is dropped.
		if d, ok := a.dialog.(*movieAddDialog); ok && d.query == m.Query {
			d.searching = false
			return notifyError(m.Err)
		}
		return nil
	case *msg.MovieAdd:
		return domi.Func(func() msg.Msg {
			mw, err := a.model.AddMovieByTMDBID(ctx, m.TMDBID)
			if err != nil {
				return &msg.Error{Err: err}
			}
			return &msg.MovieAdded{TMDBID: m.TMDBID, Local: &mw.MovieHead}
		})
	case *msg.MovieAdded:
		// Mark the matching search result as in the library; the
		// movie list itself updates with the re-render.
		if d, ok := a.dialog.(*movieAddDialog); ok {
			for i := range d.results {
				if d.results[i].Movie.ID == m.TMDBID {
					d.results[i].Local = m.Local
				}
			}
		}
		return nil
	case *msg.MovieCreate:
		return a.doNav(ctx, func(tx *model.TxRW) (string, error) {
			mw, err := tx.MovieCreate("New Movie", "")
			if err != nil {
				return "", err
			}
			return mw.EditorPath(), nil
		})

	case *msg.TaskRun:
		if err := a.model.RunTaskNow(ctx, m.ID); err != nil {
			return notifyError(err)
		}
		return nil
	case *msg.TaskKill:
		a.model.KillTask(m.ID)
		return nil
	case *msg.TaskDelete:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.TaskDelete(m.ID) })

	case *msg.Trash:
		return a.doNav(ctx, func(tx *model.TxRW) (string, error) {
			dest := trashRedirectTarget(tx, m.Kind, m.ID)
			return dest, tx.Trash(m.Kind, m.ID)
		})
	case *msg.Restore:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.Restore(m.ID) })
	case *msg.Purge:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.Purge(m.ID) })

	case *msg.CollectionAdd:
		return a.doNav(ctx, func(tx *model.TxRW) (string, error) {
			col, err := tx.CollectionCreate("New Collection")
			if err != nil {
				return "", err
			}
			return col.EditorPath(), nil
		})
	case *msg.CollectionMovieAddOpen:
		a.dialog = &collectionMovieAddDialog{colID: m.CollectionID}
		return nil
	case *msg.CollectionSeriesAddOpen:
		a.dialog = &collectionSeriesAddDialog{colID: m.CollectionID}
		return nil
	case *msg.ImageDialogOpen:
		a.dialog = &imageDialog{kind: m.Kind, id: m.ID}
		return nil
	case *msg.DownloadFileAttachOpen:
		a.dialog = &downloadFileAttachPopover{infoHash: m.InfoHash, path: m.Path}
		return domi.Func(func() msg.Msg {
			var attached []string
			err := a.model.WithTxR(ctx, func(tx *model.TxR) error {
				attached = tx.DownloadAttachedEpisodes(m.InfoHash, m.Path)
				return nil
			})
			if err != nil {
				return &msg.Error{Err: err}
			}
			return &msg.DownloadFileAttachOpened{InfoHash: m.InfoHash, Path: m.Path, Attached: attached}
		})
	case *msg.DownloadFileAttachOpened:
		// Drop snapshots that arrive after the picker closed or
		// moved to another file.
		if d, ok := a.dialog.(*downloadFileAttachPopover); ok && d.infoHash == m.InfoHash && d.path == m.Path {
			d.attached = m.Attached
			d.ready = true
		}
		return nil
	case *msg.DownloadFileAttachPick:
		a.dialog = nil
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.EpisodeVideoSet(m.InfoHash, m.Path, m.EpisodeID, true) })

	case *msg.Play:
		return a.doR(ctx, func(tx *model.TxR) cmd {
			a.player = getPlayer(tx, m)
			return nil
		})
	case *msg.PlayerClose:
		a.player = nil
		return nil
	case *msg.CollectionPickerSearch:
		switch d := a.dialog.(type) {
		case *collectionMovieAddDialog:
			d.query = m.Query
		case *collectionSeriesAddDialog:
			d.query = m.Query
		}
		return nil
	case *msg.CollectionMovieAdd:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.CollectionMovieAdd(m.CollectionID, m.MovieID) })
	case *msg.CollectionSeriesAdd:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.CollectionSeriesAdd(m.CollectionID, m.SeriesID) })
	case *msg.CollectionMovieRemove:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.CollectionMovieRemove(m.CollectionID, m.MovieID) })
	case *msg.CollectionSeriesRemove:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.CollectionSeriesRemove(m.CollectionID, m.SeriesID) })

	case *msg.SeasonAdd:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.SeasonAdd(m.EditionID) })
	case *msg.SeriesEditionAdd:
		return a.doNav(ctx, func(tx *model.TxRW) (string, error) {
			sw, err := tx.SeriesEditionClone(m.EditionID)
			if err != nil {
				return "", err
			}
			return sw.EditorPath(), nil
		})
	case *msg.MovieEditionAdd:
		return a.doNav(ctx, func(tx *model.TxRW) (string, error) {
			mw, err := tx.MovieEditionClone(m.EditionID)
			if err != nil {
				return "", err
			}
			return mw.EditorPath(), nil
		})
	case *msg.MovieEditionSetDefault:
		// No navigation: promoting changes the editions' slugs, and
		// every affected session — this one included — follows the
		// slug-change events to the right place.
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.MovieEditionSetDefault(m.ID) })

	case *msg.EpisodeCreate:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.SeasonEpisodeCreate(m.SeasonID) })
	case *msg.SeasonAddEpisode:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.SeasonEpisodeAdd(m.SeasonID, m.EpisodeID, m.SortKey) })
	case *msg.SeasonRemoveEpisode:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.SeasonEpisodeRemove(m.SeasonID, m.EpisodeID) })
	case *msg.EpisodeMove:
		return a.doRW(ctx, func(tx *model.TxRW) error {
			return tx.EpisodeMove(m.EpisodeID, m.FromSeasonID, m.SeasonID, m.Index)
		})

	case *msg.VideoReimport:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.ReimportVideo(m.ID) })
	case *msg.VideoReencode:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.ReencodeVideo(m.ID) })

	case *msg.EpisodeVideoSetActive:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.EpisodeVideoSetActive(m.EpisodeID, m.VideoID) })
	case *msg.MovieVideoSetActive:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.MovieVideoSetActive(m.MovieEditionID, m.VideoID) })

	case *msg.CollectionSetTitle:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.CollectionTitleSet(m.ID, m.Title) })
	case *msg.SeriesSetTitle:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.SeriesTitleSet(m.ID, m.Title) })
	case *msg.SeasonSetTitle:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.SeasonTitleSet(m.ID, m.Title) })

	case *msg.EpisodeSetTitle:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.EpisodeTitleSet(m.ID, m.Title) })
	case *msg.EpisodeSetAirdate:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.EpisodeAirdateSet(m.ID, m.Airdate) })
	case *msg.EpisodeSetSummary:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.EpisodeSummarySet(m.ID, m.Summary) })
	case *msg.EpisodeSetType:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.EpisodeTypeSet(m.ID, m.Type) })

	case *msg.SeriesEditionSetLabel:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.SeriesEditionLabelSet(m.ID, m.Label) })
	case *msg.SeriesEditionSetSummary:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.SeriesEditionSummarySet(m.ID, m.Summary) })

	case *msg.MovieEditionSetTitle:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.MovieEditionTitleSet(m.ID, m.Title) })
	case *msg.MovieEditionSetLabel:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.MovieEditionLabelSet(m.ID, m.Label) })
	case *msg.MovieEditionSetReleaseDate:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.MovieEditionReleaseDateSet(m.ID, m.ReleaseDate) })
	case *msg.MovieEditionSetRuntime:
		return a.doRW(ctx, func(tx *model.TxRW) error {
			var runtime int64
			if s := strings.TrimSpace(m.Runtime); s != "" {
				var err error
				runtime, err = strconv.ParseInt(s, 10, 64)
				if err != nil {
					return &model.ValidationError{Op: "set movie edition runtime", Err: err}
				}
			}
			return tx.MovieEditionRuntimeSet(m.ID, runtime)
		})
	case *msg.MovieEditionSetSummary:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.MovieEditionSummarySet(m.ID, m.Summary) })

	case *msg.DownloadImport:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.DownloadImport(m.ID) })
	case *msg.DownloadSetAutoImport:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.DownloadAutoImportSet(m.ID, m.On) })
	case *msg.EpisodeVideoSet:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.EpisodeVideoSet(m.InfoHash, m.Path, m.EpisodeID, m.Attach) })

	case *msg.TMDBSetToken:
		return a.doRW(ctx, func(tx *model.TxRW) error { return tx.SettingSetString(model.SettingKeyTMDBAccessToken, m.Token) })
	case *msg.TransmissionSetURL:
		return a.doRW(ctx, func(tx *model.TxRW) error {
			return tx.SettingSetString(model.SettingKeyTransmissionBaseURL, m.URL)
		})
	}
	panic("unreached")
}

// getPlayer resolves the video, its options, and the played content
// for m. A lookup failure surfaces as a note.
func getPlayer(tx *model.TxR, m *msg.Play) (pl *player) {
	v := tx.Video(m.IDs.VideoID)
	pl = &player{
		audio:        m.Audio,
		subtitle:     m.Subtitle,
		pinAudio:     m.PinAudio,
		video:        v,
		qualityOpts:  tx.QualityOptions(v),
		captionsOpts: tx.SubtitleOptions(v),
		audioOpts:    tx.AudioOptions(v),
	}
	if m.IDs.EpisodeID != "" {
		pl.episode = tx.EpisodeInEdition(m.IDs.EpisodeID, m.IDs.SeriesEditionID)
	} else {
		pl.movie = tx.MovieEditionHead(m.IDs.MovieEditionID)
	}
	return pl
}

// doR runs f inside a readonly tx, and displays a note on error.
func (a *app) doR(ctx context.Context, f func(tx *model.TxR) cmd) cmd {
	var c cmd
	err := a.model.WithTxR(ctx, func(tx *model.TxR) error {
		c = f(tx)
		return nil
	})
	return domi.Batch[msg.Msg](c, notifyError(err))
}

// doRW opens a read-write transaction as part of the update,
// surfacing a failure as an error note. The database is part of the
// app's state, so a state-transition write happens inline, where the
// render that follows reflects it — unlike a slow or external effect,
// which returns a cmd instead.
func (a *app) doRW(ctx context.Context, f func(tx *model.TxRW) error) cmd {
	err := a.model.WithTxRW(ctx, f)
	return notifyError(err)
}

// doNav navigates to the returned path and displays a note on error.
func (a *app) doNav(ctx context.Context, f func(tx *model.TxRW) (string, error)) cmd {
	var dest string
	c := a.doRW(ctx, func(tx *model.TxRW) (err error) {
		dest, err = f(tx)
		return err
	})
	if dest == "" {
		return c
	}
	return domi.Batch[msg.Msg](c, domi.PushURL[msg.Msg](dest))
}

func notifyError(err error) cmd {
	if err == nil {
		return nil
	}
	return hi.Notify[msg.Msg](hi.Note{
		Icon:        "line/x-circle",
		Message:     "Error",
		Description: err.Error(),
	})
}

// notify queues a note for delivery to the client on the next render.
func (a *app) notify(variant ui.NoteVariant, title string) {
	a.noteSeq++
	a.notes = append(a.notes, ui.Note{
		ID:      strconv.Itoa(a.noteSeq),
		Variant: variant,
		Title:   title,
	})
}

// setPath is used by both Update *and* Preview (and newApp).
// It can set fields on a but must not mutate deeper structures
// or touch the db.
func (a *app) setPath(ctx context.Context, u *url.URL) {
	slog.InfoContext(ctx, "navigate", "path", u.Path)
	a.dialog = nil // navigating away closes any open dialog
	a.player = nil // and the player
	a.path = u.Path
}

func slugResolve(tx *model.TxR, slugs, allowed []string) map[string]string {
	odesc := tx.SlugResolve(slugs)
	if len(allowed) == 0 || slices.Contains(allowed, odesc["kind"]) {
		return odesc
	}
	return nil
}

// slugs returns the section ("theater" or "editor") and slugs
// for the given path, if any.
func slugs(path []string) (section string, slugs, allowed []string) {
	if len(path) == 0 || path[0] == "collections" {
		return "", nil, nil
	}
	if path[0] != "app" {
		return sectionTheater, path, nil // all allowed
	}
	if len(path) >= 3 {
		switch path[1] {
		case "movies":
			return sectionEditor, path[2:], []string{model.KindMovieEdition}
		case "series":
			return sectionEditor, path[2:], []string{model.KindSeriesEdition, model.KindEpisode}
		case "collections":
			return sectionEditor, path[2:], []string{model.KindCollectionOverview}
		}
	}
	return "", nil, nil
}
