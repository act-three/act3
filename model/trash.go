package model

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"kr.dev/errorfmt"

	"ily.dev/act3/database/schema"
)

type TrashKind int

const (
	TrashKindInvalid TrashKind = iota
	TrashKindMovie
	TrashKindMovieEdition
	TrashKindSeries
	TrashKindSeriesEdition
	TrashKindSeason
	TrashKindEpisode
	TrashKindVideo
	TrashKindCollection
	TrashKindDownload
)

// trashRetention is how long a trashed entity stays in the trash
// before the purge loop hard-deletes it.
const trashRetention = 30 * 24 * time.Hour

// ErrAlreadyTrashed is returned by Trash when the target row is
// already soft-deleted.
var ErrAlreadyTrashed = errors.New("already trashed")

// ErrNotTrashed is returned by Restore and Purge when the target
// row is currently live.
var ErrNotTrashed = errors.New("not trashed")

// ErrCascadeTrashed is returned by Restore when the target row was
// trashed as part of an ancestor's cascade. Such rows aren't shown on
// the trash page and aren't individually restorable; restore the
// cascade root instead.
var ErrCascadeTrashed = errors.New("cascade-trashed; restore the root instead")

type TrashItem struct {
	Kind      TrashKind
	ID        string
	Title     string
	Subtitle  string
	DeletedAt time.Time
}

type trashState struct {
	trashed   bool
	cascadeOf string
}

func (s trashState) live() bool { return !s.trashed }

func (tx *TxR) trashState(ctx Context, id string) (trashState, error) {
	row, err := tx.q.TrashGet(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return trashState{}, nil
	}
	if err != nil {
		return trashState{}, err
	}
	cascadeOf := ""
	if row.CascadeOf != nil {
		cascadeOf = *row.CascadeOf
	}
	return trashState{trashed: true, cascadeOf: cascadeOf}, nil
}

func (tx *TxRW) insertTrashRow(ctx Context, id, root string, now time.Time) error {
	kind := kindOf(id)
	title, subtitle, err := tx.computeTrashTitle(ctx, id, kind)
	if err != nil {
		return err
	}
	params := schema.TrashInsertParams{
		ID: id, Title: title, Subtitle: subtitle,
		DeletedAt: now.UnixMilli(),
	}
	if root != id {
		params.CascadeOf = &root
	}
	return tx.q.TrashInsert(ctx, params)
}

func (tx *TxRW) computeTrashTitle(ctx Context, id string, kind TrashKind) (title, subtitle string, err error) {
	switch kind {
	case TrashKindMovie:
		r, err := tx.q.MovieEditionGetDefault(ctx, id)
		if err != nil {
			return "", "", err
		}
		return r.Title, "", nil
	case TrashKindMovieEdition:
		r, err := tx.q.MovieEditionGet(ctx, id)
		if err != nil {
			return "", "", err
		}
		return r.Title + " \u00b7 " + r.Label, "", nil
	case TrashKindSeries:
		r, err := tx.q.SeriesGet(ctx, id)
		if err != nil {
			return "", "", err
		}
		return r.Title, "", nil
	case TrashKindSeriesEdition:
		r, err := tx.q.SeriesEditionGet(ctx, id)
		if err != nil {
			return "", "", err
		}
		sr, err := tx.q.SeriesGet(ctx, r.SeriesID)
		if err != nil {
			return "", "", err
		}
		return sr.Title + " \u00b7 " + r.Label, "", nil
	case TrashKindSeason:
		r, err := tx.q.SeasonGet(ctx, id)
		if err != nil {
			return "", "", err
		}
		sed, err := tx.q.SeriesEditionGet(ctx, r.EditionID)
		if err != nil {
			return "", "", err
		}
		sr, err := tx.q.SeriesGet(ctx, sed.SeriesID)
		if err != nil {
			return "", "", err
		}
		return r.Title, sr.Title + " \u00b7 " + sed.Label, nil
	case TrashKindEpisode:
		r, err := tx.q.EpisodeGetAny(ctx, id)
		if err != nil {
			return "", "", err
		}
		sneps, err := tx.q.SeasonEpisodeListByEpisodeID(ctx, id)
		if err != nil {
			return "", "", err
		}
		if len(sneps) > 0 {
			sed, err := tx.q.SeriesEditionGet(ctx, sneps[0].EditionID)
			if err != nil {
				return "", "", err
			}
			sr, err := tx.q.SeriesGet(ctx, sed.SeriesID)
			if err != nil {
				return "", "", err
			}
			return r.Title, sr.Title, nil
		}
		return r.Title, "", nil
	case TrashKindVideo:
		r, err := tx.q.VideoGet(ctx, id)
		if err != nil {
			return "", "", err
		}
		return r.Name, "", nil
	case TrashKindCollection:
		r, err := tx.q.CollectionGet(ctx, id)
		if err != nil {
			return "", "", err
		}
		return r.Title, "", nil
	case TrashKindDownload:
		r, err := tx.q.DownloadGet(ctx, id)
		if err != nil {
			return "", "", err
		}
		return r.Title, "", nil
	}
	return "", "", fmt.Errorf("no trashable kind for ID %q", id)
}

// Trash soft-deletes the item and cascades the trash down its owned
// sub-tree, orphan-reaping shared Episode/Video rows that lose their
// last live reference.
func (tx *TxRW) Trash(ctx Context, id string) (err error) {
	kind := kindOf(id)
	defer errorfmt.Handlef("trash %s: %w", id, &err)

	state, err := tx.trashState(ctx, id)
	if err != nil {
		return err
	}
	if !state.live() {
		return ErrAlreadyTrashed
	}

	var cascaded []TrashItem
	tx.onCommit(func() {
		tx.m.emitEvent(&Event{Type: EventTrash, ID: id, TrashKind: kind})
		if len(cascaded) > 0 {
			tx.m.emitEvent(&Event{
				Type:       EventTrashCascade,
				ID:         id,
				TrashKind:  kind,
				TrashItems: cascaded,
			})
		}
	})

	now := time.Now()
	switch kind {
	case TrashKindMovie:
		err = tx.trashMovie(ctx, id, id, now)
	case TrashKindMovieEdition:
		err = tx.trashMovieEdition(ctx, id, id, now)
	case TrashKindSeries:
		err = tx.trashSeries(ctx, id, id, now)
	case TrashKindSeriesEdition:
		err = tx.trashSeriesEdition(ctx, id, id, now)
	case TrashKindSeason:
		err = tx.trashSeason(ctx, id, id, now)
	case TrashKindEpisode:
		err = tx.trashEpisode(ctx, id, id, now)
	case TrashKindVideo:
		err = tx.trashVideo(ctx, id, id, now)
	case TrashKindCollection:
		err = tx.trashCollection(ctx, id, id, now)
	case TrashKindDownload:
		err = tx.trashDownload(ctx, id, id, now)
	default:
		return fmt.Errorf("no trashable kind for ID %q", id)
	}
	if err != nil {
		return err
	}

	cascaded, err = tx.trashListByRoot(ctx, id)
	return err
}

func (tx *TxRW) trashMovie(ctx Context, id, root string, now time.Time) error {
	if err := tx.insertTrashRow(ctx, id, root, now); err != nil {
		return err
	}
	meds, err := tx.q.MovieEditionListByMovieID(ctx, id)
	if err != nil {
		return err
	}
	for _, med := range meds {
		if err := tx.trashMovieEdition(ctx, med.ID, root, now); err != nil {
			return err
		}
	}
	if err := tx.q.CollectionMovieSoftDeleteByMovieID(ctx, schema.CollectionMovieSoftDeleteByMovieIDParams{
		DeletedAt: new(now.UnixMilli()), MovieID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.MovieSoftDelete(ctx, schema.MovieSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), ID: id,
	}); err != nil {
		return err
	}
	return tx.q.SlugDelete(ctx, id)
}

func (tx *TxRW) trashMovieEdition(ctx Context, id, root string, now time.Time) error {
	if id == root {
		// Directly trashed: if this was the default edition, promote a
		// sibling to default before trashing so the movie retains one.
		med, err := tx.q.MovieEditionGet(ctx, id)
		if err != nil {
			return err
		}
		if med.Slug == "" {
			succ, err := tx.q.MovieEditionDefaultSuccessor(ctx, med.MovieID)
			if err != nil {
				return err
			}
			if err := tx.MovieEditionSetDefault(ctx, succ.ID); err != nil {
				return err
			}
		}
	}
	if err := tx.insertTrashRow(ctx, id, root, now); err != nil {
		return err
	}
	dls, err := tx.q.DownloadListByMovieEditionID(ctx, &id)
	if err != nil {
		return err
	}
	for _, dl := range dls {
		if err := tx.trashDownload(ctx, dl.InfoHash, root, now); err != nil {
			return err
		}
	}
	if err := tx.q.MovieVideoSoftDeleteByMovieEditionID(ctx, schema.MovieVideoSoftDeleteByMovieEditionIDParams{
		DeletedAt: new(now.UnixMilli()), MovieEditionID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.MovieEditionSoftDelete(ctx, schema.MovieEditionSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), ID: id,
	}); err != nil {
		return err
	}
	return tx.reapOrphanVideos(ctx, root, now)
}

func (tx *TxRW) trashSeries(ctx Context, id, root string, now time.Time) error {
	if err := tx.insertTrashRow(ctx, id, root, now); err != nil {
		return err
	}
	seds, err := tx.q.SeriesEditionListBySeriesID(ctx, id)
	if err != nil {
		return err
	}
	for _, sed := range seds {
		if err := tx.trashSeriesEdition(ctx, sed.ID, root, now); err != nil {
			return err
		}
	}
	if err := tx.q.CollectionSeriesSoftDeleteBySeriesID(ctx, schema.CollectionSeriesSoftDeleteBySeriesIDParams{
		DeletedAt: new(now.UnixMilli()), SeriesID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.SeriesSoftDelete(ctx, schema.SeriesSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), ID: id,
	}); err != nil {
		return err
	}
	return tx.q.SlugDelete(ctx, id)
}

func (tx *TxRW) trashSeriesEdition(ctx Context, id, root string, now time.Time) error {
	if id == root {
		sed, err := tx.q.SeriesEditionGet(ctx, id)
		if err != nil {
			return err
		}
		if sed.Slug == "" {
			succ, err := tx.q.SeriesEditionDefaultSuccessor(ctx, sed.SeriesID)
			if err != nil {
				return err
			}
			if err := tx.SeriesEditionSetDefault(ctx, succ.ID); err != nil {
				return err
			}
		}
	}
	if err := tx.insertTrashRow(ctx, id, root, now); err != nil {
		return err
	}
	dls, err := tx.q.DownloadListBySeriesEditionID(ctx, &id)
	if err != nil {
		return err
	}
	for _, dl := range dls {
		if err := tx.trashDownload(ctx, dl.InfoHash, root, now); err != nil {
			return err
		}
	}
	sns, err := tx.q.SeasonListByEditionID(ctx, id)
	if err != nil {
		return err
	}
	for _, sn := range sns {
		if err := tx.trashSeason(ctx, sn.ID, root, now); err != nil {
			return err
		}
	}
	return tx.q.SeriesEditionSoftDelete(ctx, schema.SeriesEditionSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), ID: id,
	})
}

func (tx *TxRW) trashSeason(ctx Context, id, root string, now time.Time) error {
	if err := tx.insertTrashRow(ctx, id, root, now); err != nil {
		return err
	}
	if err := tx.q.SeasonEpisodeSoftDeleteBySeasonID(ctx, schema.SeasonEpisodeSoftDeleteBySeasonIDParams{
		DeletedAt: new(now.UnixMilli()), SeasonID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.SeasonSoftDelete(ctx, schema.SeasonSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), ID: id,
	}); err != nil {
		return err
	}
	return tx.reapOrphanEpisodes(ctx, root, now)
}

func (tx *TxRW) trashEpisode(ctx Context, id, root string, now time.Time) error {
	var sneps []schema.SeasonEpisode
	if id == root {
		// Directly trashed: collect live junctions so we can renumber
		// the affected seasons after the episode is out.
		var err error
		sneps, err = tx.q.SeasonEpisodeListByEpisodeID(ctx, id)
		if err != nil {
			return err
		}
	}
	if err := tx.insertTrashRow(ctx, id, root, now); err != nil {
		return err
	}
	if err := tx.q.SeasonEpisodeSoftDeleteByEpisodeID(ctx, schema.SeasonEpisodeSoftDeleteByEpisodeIDParams{
		DeletedAt: new(now.UnixMilli()), EpisodeID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.EpisodeVideoSoftDeleteByEpisodeID(ctx, schema.EpisodeVideoSoftDeleteByEpisodeIDParams{
		DeletedAt: new(now.UnixMilli()), EpisodeID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.EpisodeSoftDelete(ctx, schema.EpisodeSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), ID: id,
	}); err != nil {
		return err
	}
	if err := tx.reapOrphanVideos(ctx, root, now); err != nil {
		return err
	}
	for _, snep := range sneps {
		if err := tx.renumberSeason(ctx, snep.SeasonID); err != nil {
			return err
		}
	}
	return nil
}

func (tx *TxRW) trashVideo(ctx Context, id, root string, now time.Time) error {
	if err := tx.insertTrashRow(ctx, id, root, now); err != nil {
		return err
	}
	if err := tx.q.EpisodeVideoSoftDeleteByVideoID(ctx, schema.EpisodeVideoSoftDeleteByVideoIDParams{
		DeletedAt: new(now.UnixMilli()), VideoID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.MovieVideoSoftDeleteByVideoID(ctx, schema.MovieVideoSoftDeleteByVideoIDParams{
		DeletedAt: new(now.UnixMilli()), VideoID: id,
	}); err != nil {
		return err
	}
	return tx.q.VideoSoftDelete(ctx, schema.VideoSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), ID: id,
	})
}

func (tx *TxRW) trashCollection(ctx Context, id, root string, now time.Time) error {
	if err := tx.insertTrashRow(ctx, id, root, now); err != nil {
		return err
	}
	if err := tx.q.CollectionMovieSoftDeleteByCollectionID(ctx, schema.CollectionMovieSoftDeleteByCollectionIDParams{
		DeletedAt: new(now.UnixMilli()), CollectionID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.CollectionSeriesSoftDeleteByCollectionID(ctx, schema.CollectionSeriesSoftDeleteByCollectionIDParams{
		DeletedAt: new(now.UnixMilli()), CollectionID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.CollectionSoftDelete(ctx, schema.CollectionSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), ID: id,
	}); err != nil {
		return err
	}
	return tx.q.SlugDelete(ctx, id)
}

func (tx *TxRW) trashDownload(ctx Context, id, root string, now time.Time) error {
	if err := tx.insertTrashRow(ctx, id, root, now); err != nil {
		return err
	}
	if err := tx.q.DownloadSoftDelete(ctx, schema.DownloadSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), InfoHash: id,
	}); err != nil {
		return err
	}
	return tx.reapOrphanVideos(ctx, root, now)
}

// reapOrphanEpisodes trashes every live Episode with no live
// SeasonEpisode junction, under the given cascade root. Called after
// a cascade soft-deletes its SeasonEpisode junctions.
func (tx *TxRW) reapOrphanEpisodes(ctx Context, root string, now time.Time) error {
	epIDs, err := tx.q.EpisodeListOrphans(ctx)
	if err != nil {
		return err
	}
	for _, epID := range epIDs {
		if err := tx.trashEpisode(ctx, epID, root, now); err != nil {
			return err
		}
	}
	return nil
}

// reapOrphanVideos trashes every live Video with no live EpisodeVideo
// or MovieVideo junction, under the given cascade root. Called after
// a cascade soft-deletes its video junctions.
func (tx *TxRW) reapOrphanVideos(ctx Context, root string, now time.Time) error {
	vidIDs, err := tx.q.VideoListOrphans(ctx)
	if err != nil {
		return err
	}
	for _, vidID := range vidIDs {
		if err := tx.trashVideo(ctx, vidID, root, now); err != nil {
			return err
		}
	}
	return nil
}

// Restore un-trashes a directly-trashed item, restoring any trashed
// structural ancestors first so it's reachable. Cascade-trashed items
// (CascadeOf set) aren't individually restorable and return
// ErrCascadeTrashed.
func (tx *TxRW) Restore(ctx Context, id string) (err error) {
	defer errorfmt.Handlef("restore %s: %w", id, &err)

	state, err := tx.trashState(ctx, id)
	if err != nil {
		return err
	}
	if state.live() {
		return ErrNotTrashed
	}
	if state.cascadeOf != "" {
		return ErrCascadeTrashed
	}
	return tx.ensureLive(ctx, id)
}

// ensureParentsLive makes every immediate structural parent of the
// item live, delegating back to ensureLive for each so the chain
// climbs as far as needed. For an Episode whose containing Series
// was separately trashed after the episode itself, this walks the
// junctions up to the Seasons and ensureLive pulls the rest back.
func (tx *TxRW) ensureParentsLive(ctx Context, id string) error {
	switch kindOf(id) {
	case TrashKindMovieEdition:
		med, err := tx.q.MovieEditionGet(ctx, id)
		if err != nil {
			return err
		}
		return tx.ensureLive(ctx, med.MovieID)
	case TrashKindSeriesEdition:
		sed, err := tx.q.SeriesEditionGet(ctx, id)
		if err != nil {
			return err
		}
		return tx.ensureLive(ctx, sed.SeriesID)
	case TrashKindSeason:
		sn, err := tx.q.SeasonGet(ctx, id)
		if err != nil {
			return err
		}
		return tx.ensureLive(ctx, sn.EditionID)
	case TrashKindEpisode:
		seasonIDs, err := tx.q.SeasonEpisodeDistinctSeasonsByEpisode(ctx, id)
		if err != nil {
			return err
		}
		for _, snID := range seasonIDs {
			if err := tx.ensureLive(ctx, snID); err != nil {
				return err
			}
		}
	case TrashKindVideo:
		epIDs, err := tx.q.EpisodeVideoDistinctEpisodesByVideo(ctx, id)
		if err != nil {
			return err
		}
		for _, epID := range epIDs {
			if err := tx.ensureLive(ctx, epID); err != nil {
				return err
			}
		}
		medIDs, err := tx.q.MovieVideoDistinctEditionsByVideo(ctx, id)
		if err != nil {
			return err
		}
		for _, medID := range medIDs {
			if err := tx.ensureLive(ctx, medID); err != nil {
				return err
			}
		}
	}
	return nil
}

// ensureLive makes the target live. Cascade-trashed targets are
// restored by restoring their cascade root (whose restoreRoot then
// brings the target along). Directly-trashed targets walk their own
// ancestor chain before being restored via restoreRoot. No-op if
// already live.
func (tx *TxRW) ensureLive(ctx Context, id string) error {
	state, err := tx.trashState(ctx, id)
	if err != nil {
		return err
	}
	if state.live() {
		return nil
	}
	if state.cascadeOf != "" {
		return tx.ensureLive(ctx, state.cascadeOf)
	}
	if err := tx.ensureParentsLive(ctx, id); err != nil {
		return err
	}
	return tx.restoreRoot(ctx, id)
}

func (tx *TxRW) restoreRoot(ctx Context, id string) error {
	kind := kindOf(id)
	cascaded, err := tx.trashListByRoot(ctx, id)
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		tx.m.emitEvent(&Event{
			Type:       EventRestore,
			ID:         id,
			TrashKind:  kind,
			TrashItems: cascaded,
		})
	})

	var sneps []schema.SeasonEpisode
	switch kind {
	case TrashKindMovie:
		if err := tx.restoreSlug(ctx, id); err != nil {
			return err
		}
		if err := tx.q.MovieRestore(ctx, id); err != nil {
			return err
		}
	case TrashKindMovieEdition:
		if err := tx.restoreSlug(ctx, id); err != nil {
			return err
		}
		if err := tx.q.MovieEditionRestore(ctx, id); err != nil {
			return err
		}
	case TrashKindSeries:
		if err := tx.restoreSlug(ctx, id); err != nil {
			return err
		}
		if err := tx.q.SeriesRestore(ctx, id); err != nil {
			return err
		}
	case TrashKindSeriesEdition:
		if err := tx.restoreSlug(ctx, id); err != nil {
			return err
		}
		if err := tx.q.SeriesEditionRestore(ctx, id); err != nil {
			return err
		}
	case TrashKindSeason:
		if err := tx.q.SeasonRestore(ctx, id); err != nil {
			return err
		}
	case TrashKindEpisode:
		if err := tx.q.EpisodeRestore(ctx, id); err != nil {
			return err
		}
		sneps, err = tx.q.SeasonEpisodeListRestorableByEpisode(ctx, id)
		if err != nil {
			return err
		}
		for _, snep := range sneps {
			if err := tx.seasonEpisodeSortKeyBump(ctx, snep.SeasonID, snep.SortKey); err != nil {
				return err
			}
		}
	case TrashKindVideo:
		if err := tx.q.VideoRestore(ctx, id); err != nil {
			return err
		}
	case TrashKindCollection:
		if err := tx.restoreSlug(ctx, id); err != nil {
			return err
		}
		if err := tx.q.CollectionRestore(ctx, id); err != nil {
			return err
		}
	case TrashKindDownload:
		if err := tx.q.DownloadRestore(ctx, schema.DownloadRestoreParams{
			LastActivityAt: time.Now().UnixMilli(),
			InfoHash:       id,
		}); err != nil {
			return err
		}
	default:
		return fmt.Errorf("no trashable kind for ID %q", id)
	}

	cascadeOf := &id
	for _, f := range []func(context.Context, *string) error{
		tx.q.CollectionRestoreByCascade,
		tx.q.DownloadRestoreByCascade,
		tx.q.EpisodeRestoreByCascade,
		tx.q.MovieRestoreByCascade,
		tx.q.MovieEditionRestoreByCascade,
		tx.q.SeasonRestoreByCascade,
		tx.q.SeriesRestoreByCascade,
		tx.q.SeriesEditionRestoreByCascade,
		tx.q.VideoRestoreByCascade,
	} {
		if err := f(ctx, cascadeOf); err != nil {
			return err
		}
	}

	for _, f := range []func(context.Context, *string) error{
		tx.q.SeasonEpisodeRestoreByCascade,
		tx.q.EpisodeVideoRestoreByCascade,
		tx.q.MovieVideoRestoreByCascade,
		tx.q.CollectionMovieRestoreByCascade,
		tx.q.CollectionSeriesRestoreByCascade,
	} {
		if err := f(ctx, cascadeOf); err != nil {
			return err
		}
	}

	for _, snep := range sneps {
		if err := tx.renumberSeason(ctx, snep.SeasonID); err != nil {
			return err
		}
	}

	return tx.q.TrashDelete(ctx, id)
}

// restoreSlug regenerates the slug for a restored entity, emitting a
// set-slug event if it changed and reinserting the Slug table row for
// top-level entities. Dispatches to the kind-specific ensureSlug
// helper, which handles both the live (title/label-change) and
// trashed (restore) cases via the entity's DeletedAt.
func (tx *TxRW) restoreSlug(ctx Context, id string) error {
	switch kindOf(id) {
	case TrashKindMovie:
		return tx.movieEnsureSlug(ctx, id)
	case TrashKindSeries:
		return tx.seriesEnsureSlug(ctx, id)
	case TrashKindCollection:
		return tx.collectionEnsureSlug(ctx, id)
	case TrashKindMovieEdition:
		return tx.movieEditionEnsureSlug(ctx, id)
	case TrashKindSeriesEdition:
		return tx.seriesEditionEnsureSlug(ctx, id)
	}
	return nil
}

// Purge hard-deletes a trashed item and all rows in its cascade
// subtree. Returns ErrNotTrashed if the target is currently live.
func (tx *TxRW) Purge(ctx Context, id string) (err error) {
	kind := kindOf(id)
	defer errorfmt.Handlef("purge %s: %w", id, &err)

	state, err := tx.trashState(ctx, id)
	if err != nil {
		return err
	}
	if state.live() {
		return ErrNotTrashed
	}
	if state.cascadeOf != "" {
		return ErrCascadeTrashed
	}

	tx.onCommit(func() {
		tx.m.emitEvent(&Event{
			Type:      EventPurge,
			ID:        id,
			TrashKind: kind,
		})
	})

	cascadeOf := &id
	vids, err := tx.q.VideoListPurgeByCascade(ctx, cascadeOf)
	if err != nil {
		return err
	}
	var vidIDs, origKeys []string
	for _, v := range vids {
		vidIDs = append(vidIDs, v.ID)
		if v.OriginalKey != "" {
			origKeys = append(origKeys, v.OriginalKey)
		}
	}
	if err := tx.purgeVideoBlobs(ctx, vidIDs, origKeys); err != nil {
		return err
	}

	for _, f := range []func(context.Context, *string) error{
		tx.q.EpisodeVideoPurgeByCascade,
		tx.q.MovieVideoPurgeByCascade,
		tx.q.SeasonEpisodePurgeByCascade,
		tx.q.CollectionMoviePurgeByCascade,
		tx.q.CollectionSeriesPurgeByCascade,
	} {
		if err := f(ctx, cascadeOf); err != nil {
			return err
		}
	}

	for _, f := range []func(context.Context, *string) error{
		tx.q.SeasonPurgeByCascade,
		tx.q.EpisodePurgeByCascade,
		tx.q.SeriesEditionPurgeByCascade,
		tx.q.MovieEditionPurgeByCascade,
		tx.q.VideoPurgeByCascade,
		tx.q.SeriesPurgeByCascade,
		tx.q.MoviePurgeByCascade,
		tx.q.CollectionPurgeByCascade,
		tx.q.DownloadPurgeByCascade,
	} {
		if err := f(ctx, cascadeOf); err != nil {
			return err
		}
	}

	return tx.q.TrashDelete(ctx, id)
}

func (tx *TxR) TrashItem(ctx Context, id string) (TrashItem, error) {
	row, err := tx.q.TrashGet(ctx, id)
	if err != nil {
		return TrashItem{}, err
	}
	return TrashItem{
		Kind:      kindOf(row.ID),
		ID:        row.ID,
		Title:     row.Title,
		Subtitle:  row.Subtitle,
		DeletedAt: time.UnixMilli(row.DeletedAt),
	}, nil
}

// TrashList returns every directly-trashed entity (roots only),
// ordered newest-trashed first.
func (tx *TxR) TrashList(ctx Context) ([]TrashItem, error) {
	rows, err := tx.q.TrashList(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]TrashItem, len(rows))
	for i, r := range rows {
		items[i] = TrashItem{
			Kind:      kindOf(r.ID),
			ID:        r.ID,
			Title:     r.Title,
			Subtitle:  r.Subtitle,
			DeletedAt: time.UnixMilli(r.DeletedAt),
		}
	}
	return items, nil
}

func (tx *TxR) trashListByRoot(ctx Context, rootID string) ([]TrashItem, error) {
	rows, err := tx.q.TrashListByRoot(ctx, &rootID)
	if err != nil {
		return nil, err
	}
	items := make([]TrashItem, len(rows))
	for i, r := range rows {
		items[i] = TrashItem{
			Kind:      kindOf(r.ID),
			ID:        r.ID,
			Title:     r.Title,
			Subtitle:  r.Subtitle,
			DeletedAt: time.UnixMilli(r.DeletedAt),
		}
	}
	return items, nil
}

func (tx *TxRW) trashPurge(ctx Context, threshold time.Time) (err error) {
	defer errorfmt.Handlef("trash purge: %w", &err)
	roots, err := tx.q.TrashRootsBefore(ctx, threshold.UnixMilli())
	if err != nil {
		return err
	}
	for _, rootID := range roots {
		if err := tx.Purge(ctx, rootID); err != nil {
			return err
		}
	}
	return nil
}

// purgeVideoBlobs deletes AudioTrack and Rendition rows for the given
// videos and schedules their CAS blob keys for removal on commit.
func (tx *TxRW) purgeVideoBlobs(ctx Context, vidIDs, origKeys []string) error {
	if len(vidIDs) == 0 {
		return nil
	}
	rendKeys, err := tx.q.RenditionListKeysByVideoIDs(ctx, vidIDs)
	if err != nil {
		return err
	}
	keys := append(origKeys, rendKeys...)
	if len(keys) > 0 {
		tx.onCommit(func() {
			for _, k := range keys {
				tx.m.store.Remove(k)
			}
		})
	}
	if err := tx.q.AudioTrackDeleteByVideoIDList(ctx, vidIDs); err != nil {
		return err
	}
	return tx.q.RenditionDeleteByVideoIDList(ctx, vidIDs)
}

func (m *Model) purgeTrashOnce(ctx Context) error {
	threshold := time.Now().Add(-trashRetention)
	return m.WithTxRW(func(tx *TxRW) error {
		return tx.trashPurge(ctx, threshold)
	})
}

func (m *Model) purgeTrashLoop() {
	for {
		time.Sleep(time.Hour)
		if err := m.purgeTrashOnce(context.Background()); err != nil {
			slog.Error("trash purge", "error", err)
		}
	}
}

// kindOf returns the TrashKind implied by a flurry ID prefix, or
// TrashKindInvalid if the ID doesn't match a known prefix. Longer
// prefixes are checked first so "med" and "sed" don't collide with
// "mo" and "sr".
func kindOf(id string) TrashKind {
	switch {
	case strings.HasPrefix(id, "med"):
		return TrashKindMovieEdition
	case strings.HasPrefix(id, "mo"):
		return TrashKindMovie
	case strings.HasPrefix(id, "sed"):
		return TrashKindSeriesEdition
	case strings.HasPrefix(id, "sn"):
		return TrashKindSeason
	case strings.HasPrefix(id, "sr"):
		return TrashKindSeries
	case strings.HasPrefix(id, "ep"):
		return TrashKindEpisode
	case strings.HasPrefix(id, "vid"):
		return TrashKindVideo
	case strings.HasPrefix(id, "col"):
		return TrashKindCollection
	}
	// Downloads have no flurry prefix; their ID is a 40-char hex SHA-1
	// info hash, which can't collide with any of the prefixes above.
	if len(id) == 40 {
		if _, err := hex.DecodeString(id); err == nil {
			return TrashKindDownload
		}
	}
	return TrashKindInvalid
}
