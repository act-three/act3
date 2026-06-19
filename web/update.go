package web

import (
	"context"
	"maps"
	"net/url"
	"path"
	"strconv"
	"strings"

	"ily.dev/domi"

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
		a.dialog = nil // navigating away closes any open dialog
		a.player = nil // and the player
		return nil
	case *msg.URLRequest:
		if !m.Internal {
			return domi.Load[msg.Msg](m.URL.String())
		}
		if dest := redirects[strings.TrimRight(path.Clean(m.URL.Path), "/")]; dest != "" {
			m.URL.Path = dest
		}
		return domi.PushURL[msg.Msg](m.URL.String())
	case *msg.ModelEvent:
		for _, d := range m.Details {
			if d.SlugChangeID == "" {
				continue
			}
			// Regenerate the current page's URL if a relevant slug changed.
			if c := a.follow(ctx, d.SlugChangeID); c != nil {
				return c
			}
		}
		return nil
	case *msg.Error:
		a.notify(ui.NoteError, m.Err.Error())
		return nil

	case *msg.DialogClose:
		a.dialog = nil
		return nil
	case *msg.SeriesAddOpen:
		a.dialog = &seriesAddDialog{}
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
			a.notify(ui.NoteError, m.Err.Error())
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
			a.notify(ui.NoteError, m.Err.Error())
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

	case *msg.TaskRun:
		if err := a.model.RunTaskNow(ctx, m.ID); err != nil {
			a.notify(ui.NoteError, err.Error())
		}
		return nil
	case *msg.TaskKill:
		a.model.KillTask(m.ID)
		return nil
	case *msg.TaskDelete:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.TaskDelete(m.ID) })
		return nil

	case *msg.Trash:
		return a.doNav(ctx, func(tx *model.TxRW) (string, error) {
			dest, err := trashRedirectTarget(tx, m.ID)
			if err != nil {
				return "", err
			}
			return dest, tx.Trash(m.ID)
		})
	case *msg.Restore:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.Restore(m.ID) })
		return nil
	case *msg.Purge:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.Purge(m.ID) })
		return nil

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
		a.dialog = &imageDialog{id: m.ID}
		return nil
	case *msg.DownloadFileAttachOpen:
		a.dialog = &downloadFileAttachPopover{infoHash: m.InfoHash, path: m.Path}
		return domi.Func(func() msg.Msg {
			var attached []string
			err := a.model.WithTxR(ctx, func(tx *model.TxR) error {
				var err error
				attached, err = downloadAttachedEpisodes(tx, m.InfoHash, m.Path)
				return err
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
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.EpisodeVideoSet(m.InfoHash, m.Path, m.EpisodeID, true) })
		return nil

	case *msg.Play:
		a.doR(ctx, func(tx *model.TxR) (err error) {
			a.player, err = getPlayer(tx, m)
			return err
		})
		return nil
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
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.CollectionMovieAdd(m.CollectionID, m.MovieID) })
		return nil
	case *msg.CollectionSeriesAdd:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.CollectionSeriesAdd(m.CollectionID, m.SeriesID) })
		return nil
	case *msg.CollectionMovieRemove:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.CollectionMovieRemove(m.CollectionID, m.MovieID) })
		return nil
	case *msg.CollectionSeriesRemove:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.CollectionSeriesRemove(m.CollectionID, m.SeriesID) })
		return nil

	case *msg.SeasonAdd:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.SeasonAdd(m.EditionID) })
		return nil
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
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.MovieEditionSetDefault(m.ID) })
		return nil

	case *msg.EpisodeCreate:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.SeasonEpisodeCreate(m.SeasonID) })
		return nil
	case *msg.SeasonAddEpisode:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.SeasonEpisodeAdd(m.SeasonID, m.EpisodeID, m.SortKey) })
		return nil
	case *msg.SeasonRemoveEpisode:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.SeasonEpisodeRemove(m.SeasonID, m.EpisodeID) })
		return nil
	case *msg.EpisodeMove:
		a.doRW(ctx, func(tx *model.TxRW) error {
			return tx.EpisodeMove(m.EpisodeID, m.FromSeasonID, m.SeasonID, m.Index)
		})
		return nil

	case *msg.VideoReimport:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.ReimportVideo(m.ID) })
		return nil
	case *msg.VideoReencode:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.ReencodeVideo(m.ID) })
		return nil

	case *msg.EpisodeVideoSetActive:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.EpisodeVideoSetActive(m.EpisodeID, m.VideoID) })
		return nil
	case *msg.MovieVideoSetActive:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.MovieVideoSetActive(m.MovieEditionID, m.VideoID) })
		return nil

	case *msg.CollectionSetTitle:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.CollectionTitleSet(m.ID, m.Title) })
		return nil
	case *msg.SeriesSetTitle:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.SeriesTitleSet(m.ID, m.Title) })
		return nil
	case *msg.SeasonSetTitle:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.SeasonTitleSet(m.ID, m.Title) })
		return nil

	case *msg.EpisodeSetTitle:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.EpisodeTitleSet(m.ID, m.Title) })
		return nil
	case *msg.EpisodeSetAirdate:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.EpisodeAirdateSet(m.ID, m.Airdate) })
		return nil
	case *msg.EpisodeSetSummary:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.EpisodeSummarySet(m.ID, m.Summary) })
		return nil
	case *msg.EpisodeSetType:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.EpisodeTypeSet(m.ID, m.Type) })
		return nil

	case *msg.SeriesEditionSetLabel:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.SeriesEditionLabelSet(m.ID, m.Label) })
		return nil
	case *msg.SeriesEditionSetSummary:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.SeriesEditionSummarySet(m.ID, m.Summary) })
		return nil

	case *msg.MovieEditionSetTitle:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.MovieEditionTitleSet(m.ID, m.Title) })
		return nil
	case *msg.MovieEditionSetLabel:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.MovieEditionLabelSet(m.ID, m.Label) })
		return nil
	case *msg.MovieEditionSetReleaseDate:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.MovieEditionReleaseDateSet(m.ID, m.ReleaseDate) })
		return nil
	case *msg.MovieEditionSetRuntime:
		a.doRW(ctx, func(tx *model.TxRW) error {
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
		return nil
	case *msg.MovieEditionSetSummary:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.MovieEditionSummarySet(m.ID, m.Summary) })
		return nil

	case *msg.DownloadImport:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.DownloadImport(m.ID) })
		return nil
	case *msg.DownloadSetAutoImport:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.DownloadAutoImportSet(m.ID, m.On) })
		return nil
	case *msg.EpisodeVideoSet:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.EpisodeVideoSet(m.InfoHash, m.Path, m.EpisodeID, m.Attach) })
		return nil

	case *msg.TMDBSetToken:
		a.doRW(ctx, func(tx *model.TxRW) error { return tx.SettingSetString(model.SettingKeyTMDBAccessToken, m.Token) })
		return nil
	case *msg.TransmissionSetURL:
		a.doRW(ctx, func(tx *model.TxRW) error {
			return tx.SettingSetString(model.SettingKeyTransmissionBaseURL, m.URL)
		})
		return nil
	}
	panic("unreached")
}

// getPlayer resolves the video, its options, and the played content
// for m. A lookup failure surfaces as a note and returns nil.
func getPlayer(tx *model.TxR, m *msg.Play) (pl *player, err error) {
	pl = &player{audio: m.Audio, subtitle: m.Subtitle, pinAudio: m.PinAudio}
	if pl.video, err = tx.Video(m.IDs.VideoID); err != nil {
		return nil, err
	}
	if pl.qualityOpts, err = tx.QualityOptions(pl.video); err != nil {
		return nil, err
	}
	if pl.captionsOpts, err = tx.SubtitleOptions(pl.video); err != nil {
		return nil, err
	}
	if pl.audioOpts, err = tx.AudioOptions(pl.video); err != nil {
		return nil, err
	}
	if m.IDs.EpisodeID != "" {
		pl.episode, err = tx.EpisodeInEdition(m.IDs.EpisodeID, m.IDs.SeriesEditionID)
	} else {
		pl.movie, err = tx.MovieEditionHead(m.IDs.MovieEditionID)
	}
	if err != nil {
		return nil, err
	}
	return pl, nil
}

// doR runs f inside a readonly tx, and calls notify on error.
func (a *app) doR(ctx context.Context, f func(tx *model.TxR) error) {
	if err := a.model.WithTxR(ctx, f); err != nil {
		a.notify(ui.NoteError, err.Error())
	}
}

// doRW opens a read-write transaction as part of the update,
// surfacing a failure as an error note. The database is part of the
// app's state, so a state-transition write happens inline, where the
// render that follows reflects it — unlike a slow or external effect,
// which returns a cmd instead.
func (a *app) doRW(ctx context.Context, f func(tx *model.TxRW) error) {
	if err := a.model.WithTxRW(ctx, f); err != nil {
		a.notify(ui.NoteError, err.Error())
	}
}

// doNav is [app.doTx] for a write whose result names a path, to which
// the session then navigates.
func (a *app) doNav(ctx context.Context, f func(tx *model.TxRW) (string, error)) cmd {
	var dest string
	a.doRW(ctx, func(tx *model.TxRW) error {
		var err error
		dest, err = f(tx)
		return err
	})
	if dest == "" {
		return nil
	}
	return domi.PushURL[msg.Msg](dest)
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

func (a *app) setPath(ctx context.Context, u *url.URL) {
	a.path = u.Path
	a.odesc = nil
	if section, slugs := slugs(splitPath(a.path)); section != "" {
		a.model.WithTxR(ctx, func(tx *model.TxR) error {
			a.odesc = map[string]string{"section": section}
			maps.Copy(a.odesc, tx.SlugResolve(slugs))
			return nil
		})
	}
}

// slugs returns the section ("theater" or "editor") and slugs
// for the given path, if any.
func slugs(path []string) (section string, slugs []string) {
	if len(path) == 0 || path[0] == "collections" {
		return "", nil
	}
	if path[0] != "app" {
		return sectionTheater, path
	}
	if len(path) >= 3 {
		switch path[1] {
		case "movies", "series", "collections":
			return sectionEditor, path[2:]
		}
	}
	return "", nil
}
