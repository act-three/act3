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
	"ily.dev/act3/model/kind"
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
	Kind      kind.Trash
	ID        string
	Title     string
	Subtitle  string
	DeletedAt time.Time
}

type trashState struct {
	trashed   bool
	kind      kind.Trash
	cascadeOf string
}

func (s trashState) live() bool { return !s.trashed }

func (tx *TxR) trashState(id string) (trashState, error) {
	row, err := tx.q.TrashGet(id)
	if errors.Is(err, sql.ErrNoRows) {
		return trashState{}, nil
	}
	if err != nil {
		return trashState{}, err
	}
	k, err := kind.ParseTrash(row.Kind)
	if err != nil {
		return trashState{}, err
	}
	cascadeOf := ""
	if row.CascadeOf != nil {
		cascadeOf = *row.CascadeOf
	}
	return trashState{trashed: true, kind: k, cascadeOf: cascadeOf}, nil
}

func (tx *TxRW) insertTrashRow[K kind.Trash](id, root string, now time.Time) error {
	var k K
	title, subtitle, err := tx.computeTrashTitle(id, k)
	if err != nil {
		return err
	}
	params := schema.TrashInsertParams{
		ID: id, Kind: k.String(), Title: title, Subtitle: subtitle,
		DeletedAt: now.UnixMilli(),
	}
	if root != id {
		params.CascadeOf = &root
	}
	return tx.q.TrashInsert(params)
}

func (tx *TxRW) computeTrashTitle(id string, k kind.Trash) (title, subtitle string, err error) {
	switch k.(type) {
	case kind.Movie:
		r, err := tx.q.MovieEditionGetDefault(id)
		if err != nil {
			return "", "", err
		}
		return r.Title, "", nil
	case kind.MovieEdition:
		r, err := tx.q.MovieEditionGet(id)
		if err != nil {
			return "", "", err
		}
		return r.Title + " \u00b7 " + r.Label, "", nil
	case kind.Series:
		r, err := tx.q.SeriesGet(id)
		if err != nil {
			return "", "", err
		}
		return r.Title, "", nil
	case kind.SeriesEdition:
		r, err := tx.q.SeriesEditionGet(id)
		if err != nil {
			return "", "", err
		}
		sr, err := tx.q.SeriesGet(r.SeriesID)
		if err != nil {
			return "", "", err
		}
		return sr.Title + " \u00b7 " + r.Label, "", nil
	case kind.Season:
		r, err := tx.q.SeasonGet(id)
		if err != nil {
			return "", "", err
		}
		sed, err := tx.q.SeriesEditionGet(r.EditionID)
		if err != nil {
			return "", "", err
		}
		sr, err := tx.q.SeriesGet(sed.SeriesID)
		if err != nil {
			return "", "", err
		}
		return r.Title, sr.Title + " \u00b7 " + sed.Label, nil
	case kind.Episode:
		r, err := tx.q.EpisodeGetAny(id)
		if err != nil {
			return "", "", err
		}
		sneps, err := tx.q.SeasonEpisodeListByEpisodeID(id)
		if err != nil {
			return "", "", err
		}
		if len(sneps) > 0 {
			sed, err := tx.q.SeriesEditionGet(sneps[0].EditionID)
			if err != nil {
				return "", "", err
			}
			sr, err := tx.q.SeriesGet(sed.SeriesID)
			if err != nil {
				return "", "", err
			}
			return r.Title, sr.Title, nil
		}
		return r.Title, "", nil
	case kind.Video:
		r, err := tx.q.VideoGet(id)
		if err != nil {
			return "", "", err
		}
		return r.Name, "", nil
	case kind.Collection:
		r, err := tx.q.CollectionGet(id)
		if err != nil {
			return "", "", err
		}
		return r.Title, "", nil
	case kind.Download:
		r, err := tx.q.DownloadGet(id)
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
func (tx *TxRW) Trash(id string) (err error) {
	defer errorfmt.Handlef("trash %s: %w", id, &err)

	state, err := tx.trashState(id)
	if err != nil {
		return err
	}
	if !state.live() {
		return ErrAlreadyTrashed
	}

	now := time.Now()
	switch KindOf(id).(type) {
	case nil:
		return fmt.Errorf("no trashable kind for ID %q", id)
	case kind.Movie:
		err = tx.trashMovie(id, id, now)
	case kind.MovieEdition:
		err = tx.trashMovieEdition(id, id, now)
	case kind.Series:
		err = tx.trashSeries(id, id, now)
	case kind.SeriesEdition:
		err = tx.trashSeriesEdition(id, id, now)
	case kind.Season:
		err = tx.trashSeason(id, id, now)
	case kind.Episode:
		err = tx.trashEpisode(id, id, now)
	case kind.Video:
		if err := tx.guardActiveVideo(id); err != nil {
			return err
		}
		err = tx.trashVideo(id, id, now)
	case kind.Collection:
		err = tx.trashCollection(id, id, now)
	case kind.Download:
		err = tx.trashDownload(id, id, now)
	}
	if err != nil {
		return err
	}
	return nil
}

func (tx *TxRW) trashMovie(id, root string, now time.Time) error {
	if err := tx.insertTrashRow[kind.Movie](id, root, now); err != nil {
		return err
	}
	meds, err := tx.q.MovieEditionListByMovieID(id)
	if err != nil {
		return err
	}
	for _, med := range meds {
		if err := tx.trashMovieEdition(med.ID, root, now); err != nil {
			return err
		}
	}
	if err := tx.q.CollectionMovieSoftDeleteByMovieID(schema.CollectionMovieSoftDeleteByMovieIDParams{
		DeletedAt: new(now.UnixMilli()), MovieID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.MovieSoftDelete(schema.MovieSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), ID: id,
	}); err != nil {
		return err
	}
	return tx.q.SlugDelete(id)
}

func (tx *TxRW) trashMovieEdition(id, root string, now time.Time) error {
	if id == root {
		// Directly trashed: if this was the default edition, promote a
		// sibling to default before trashing so the movie retains one.
		med, err := tx.q.MovieEditionGet(id)
		if err != nil {
			return err
		}
		if med.Slug == "" {
			succ, err := tx.q.MovieEditionDefaultSuccessor(med.MovieID)
			if err != nil {
				return err
			}
			if err := tx.MovieEditionSetDefault(succ.ID); err != nil {
				return err
			}
		}
	}
	if err := tx.insertTrashRow[kind.MovieEdition](id, root, now); err != nil {
		return err
	}
	dls, err := tx.q.DownloadListByMovieEditionID(&id)
	if err != nil {
		return err
	}
	for _, dl := range dls {
		if err := tx.trashDownload(dl.InfoHash, root, now); err != nil {
			return err
		}
	}
	if err := tx.q.MovieVideoSoftDeleteByMovieEditionID(schema.MovieVideoSoftDeleteByMovieEditionIDParams{
		DeletedAt: new(now.UnixMilli()), MovieEditionID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.MovieEditionSoftDelete(schema.MovieEditionSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), ID: id,
	}); err != nil {
		return err
	}
	return tx.reapOrphanVideos(root, now)
}

func (tx *TxRW) trashSeries(id, root string, now time.Time) error {
	if err := tx.insertTrashRow[kind.Series](id, root, now); err != nil {
		return err
	}
	seds, err := tx.q.SeriesEditionListBySeriesID(id)
	if err != nil {
		return err
	}
	for _, sed := range seds {
		if err := tx.trashSeriesEdition(sed.ID, root, now); err != nil {
			return err
		}
	}
	if err := tx.q.CollectionSeriesSoftDeleteBySeriesID(schema.CollectionSeriesSoftDeleteBySeriesIDParams{
		DeletedAt: new(now.UnixMilli()), SeriesID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.SeriesSoftDelete(schema.SeriesSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), ID: id,
	}); err != nil {
		return err
	}
	return tx.q.SlugDelete(id)
}

func (tx *TxRW) trashSeriesEdition(id, root string, now time.Time) error {
	if id == root {
		sed, err := tx.q.SeriesEditionGet(id)
		if err != nil {
			return err
		}
		if sed.Slug == "" {
			succ, err := tx.q.SeriesEditionDefaultSuccessor(sed.SeriesID)
			if err != nil {
				return err
			}
			if err := tx.seriesEditionSetDefault(succ.ID); err != nil {
				return err
			}
		}
	}
	if err := tx.insertTrashRow[kind.SeriesEdition](id, root, now); err != nil {
		return err
	}
	dls, err := tx.q.DownloadListBySeriesEditionID(&id)
	if err != nil {
		return err
	}
	for _, dl := range dls {
		if err := tx.trashDownload(dl.InfoHash, root, now); err != nil {
			return err
		}
	}
	sns, err := tx.q.SeasonListByEditionID(id)
	if err != nil {
		return err
	}
	for _, sn := range sns {
		if err := tx.trashSeason(sn.ID, root, now); err != nil {
			return err
		}
	}
	return tx.q.SeriesEditionSoftDelete(schema.SeriesEditionSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), ID: id,
	})

}

func (tx *TxRW) trashSeason(id, root string, now time.Time) error {
	if err := tx.insertTrashRow[kind.Season](id, root, now); err != nil {
		return err
	}
	if err := tx.q.SeasonEpisodeSoftDeleteBySeasonID(schema.SeasonEpisodeSoftDeleteBySeasonIDParams{
		DeletedAt: new(now.UnixMilli()), SeasonID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.SeasonSoftDelete(schema.SeasonSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), ID: id,
	}); err != nil {
		return err
	}
	return tx.reapOrphanEpisodes(root, now)
}

func (tx *TxRW) trashEpisode(id, root string, now time.Time) error {
	var sneps []schema.SeasonEpisode
	if id == root {
		// Directly trashed: collect live junctions so we can renumber
		// the affected seasons after the episode is out.
		var err error
		sneps, err = tx.q.SeasonEpisodeListByEpisodeID(id)
		if err != nil {
			return err
		}
	}
	if err := tx.insertTrashRow[kind.Episode](id, root, now); err != nil {
		return err
	}
	if err := tx.q.SeasonEpisodeSoftDeleteByEpisodeID(schema.SeasonEpisodeSoftDeleteByEpisodeIDParams{
		DeletedAt: new(now.UnixMilli()), EpisodeID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.EpisodeVideoSoftDeleteByEpisodeID(schema.EpisodeVideoSoftDeleteByEpisodeIDParams{
		DeletedAt: new(now.UnixMilli()), EpisodeID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.EpisodeSoftDelete(schema.EpisodeSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), ID: id,
	}); err != nil {
		return err
	}
	if err := tx.reapOrphanVideos(root, now); err != nil {
		return err
	}
	for _, snep := range sneps {
		if err := tx.renumberSeason(snep.SeasonID); err != nil {
			return err
		}
	}
	return nil
}

func (tx *TxRW) trashVideo(id, root string, now time.Time) error {
	if err := tx.insertTrashRow[kind.Video](id, root, now); err != nil {
		return err
	}
	if err := tx.q.EpisodeVideoSoftDeleteByVideoID(schema.EpisodeVideoSoftDeleteByVideoIDParams{
		DeletedAt: new(now.UnixMilli()), VideoID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.MovieVideoSoftDeleteByVideoID(schema.MovieVideoSoftDeleteByVideoIDParams{
		DeletedAt: new(now.UnixMilli()), VideoID: id,
	}); err != nil {
		return err
	}
	return tx.q.VideoSoftDelete(schema.VideoSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), ID: id,
	})

}

func (tx *TxRW) trashCollection(id, root string, now time.Time) error {
	if err := tx.insertTrashRow[kind.Collection](id, root, now); err != nil {
		return err
	}
	if err := tx.q.CollectionMovieSoftDeleteByCollectionID(schema.CollectionMovieSoftDeleteByCollectionIDParams{
		DeletedAt: new(now.UnixMilli()), CollectionID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.CollectionSeriesSoftDeleteByCollectionID(schema.CollectionSeriesSoftDeleteByCollectionIDParams{
		DeletedAt: new(now.UnixMilli()), CollectionID: id,
	}); err != nil {
		return err
	}
	if err := tx.q.CollectionSoftDelete(schema.CollectionSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), ID: id,
	}); err != nil {
		return err
	}
	return tx.q.SlugDelete(id)
}

func (tx *TxRW) trashDownload(id, root string, now time.Time) error {
	if err := tx.insertTrashRow[kind.Download](id, root, now); err != nil {
		return err
	}
	if err := tx.q.DownloadSoftDelete(schema.DownloadSoftDeleteParams{
		DeletedAt: new(now.UnixMilli()), InfoHash: id,
	}); err != nil {
		return err
	}
	return tx.reapOrphanVideos(root, now)
}

// reapOrphanEpisodes trashes every live Episode with no live
// SeasonEpisode junction, under the given cascade root. Called after
// a cascade soft-deletes its SeasonEpisode junctions.
func (tx *TxRW) reapOrphanEpisodes(root string, now time.Time) error {
	epIDs, err := tx.q.EpisodeListOrphans()
	if err != nil {
		return err
	}
	for _, epID := range epIDs {
		if err := tx.trashEpisode(epID, root, now); err != nil {
			return err
		}
	}
	return nil
}

// reapOrphanVideos trashes every live Video with no live EpisodeVideo
// or MovieVideo junction, under the given cascade root. Called after
// a cascade soft-deletes its video junctions.
func (tx *TxRW) reapOrphanVideos(root string, now time.Time) error {
	vidIDs, err := tx.q.VideoListOrphans()
	if err != nil {
		return err
	}
	for _, vidID := range vidIDs {
		if err := tx.trashVideo(vidID, root, now); err != nil {
			return err
		}
	}
	return nil
}

// Restore un-trashes a directly-trashed item, restoring any trashed
// structural ancestors first so it's reachable. Cascade-trashed items
// (CascadeOf set) aren't individually restorable and return
// ErrCascadeTrashed.
func (tx *TxRW) Restore(id string) (err error) {
	defer errorfmt.Handlef("restore %s: %w", id, &err)

	state, err := tx.trashState(id)
	if err != nil {
		return err
	}
	if state.live() {
		return ErrNotTrashed
	}
	if state.cascadeOf != "" {
		return ErrCascadeTrashed
	}
	return tx.ensureLive(id)
}

// ensureParentsLive makes every immediate structural parent of the
// item live, delegating back to ensureLive for each so the chain
// climbs as far as needed. For an Episode whose containing Series
// was separately trashed after the episode itself, this walks the
// junctions up to the Seasons and ensureLive pulls the rest back.
func (tx *TxRW) ensureParentsLive(k kind.Trash, id string) error {
	switch k.(type) {
	case kind.MovieEdition:
		med, err := tx.q.MovieEditionGet(id)
		if err != nil {
			return err
		}
		return tx.ensureLive(med.MovieID)
	case kind.SeriesEdition:
		sed, err := tx.q.SeriesEditionGet(id)
		if err != nil {
			return err
		}
		return tx.ensureLive(sed.SeriesID)
	case kind.Season:
		sn, err := tx.q.SeasonGet(id)
		if err != nil {
			return err
		}
		return tx.ensureLive(sn.EditionID)
	case kind.Episode:
		seasonIDs, err := tx.q.SeasonEpisodeDistinctSeasonsByEpisode(id)
		if err != nil {
			return err
		}
		for _, snID := range seasonIDs {
			if err := tx.ensureLive(snID); err != nil {
				return err
			}
		}
	case kind.Video:
		epIDs, err := tx.q.EpisodeVideoDistinctEpisodesByVideo(id)
		if err != nil {
			return err
		}
		for _, epID := range epIDs {
			if err := tx.ensureLive(epID); err != nil {
				return err
			}
		}
		medIDs, err := tx.q.MovieVideoDistinctEditionsByVideo(id)
		if err != nil {
			return err
		}
		for _, medID := range medIDs {
			if err := tx.ensureLive(medID); err != nil {
				return err
			}
		}
	default:
		// Movies, series, collections, and downloads have no
		// structural parent.
	}
	return nil
}

// ensureLive makes the target live. Cascade-trashed targets are
// restored by restoring their cascade root (whose restoreRoot then
// brings the target along). Directly-trashed targets walk their own
// ancestor chain before being restored via restoreRoot. No-op if
// already live.
func (tx *TxRW) ensureLive(id string) error {
	state, err := tx.trashState(id)
	if err != nil {
		return err
	}
	if state.live() {
		return nil
	}
	if state.cascadeOf != "" {
		return tx.ensureLive(state.cascadeOf)
	}
	if err := tx.ensureParentsLive(state.kind, id); err != nil {
		return err
	}
	return tx.restoreRoot(state.kind, id)
}

func (tx *TxRW) restoreRoot(k kind.Trash, id string) error {
	var sneps []schema.SeasonEpisode
	switch k.(type) {
	case nil:
		return fmt.Errorf("no trashable kind for ID %q", id)
	case kind.Movie:
		if err := tx.movieEnsureSlug(id); err != nil {
			return err
		}
		if err := tx.q.MovieRestore(id); err != nil {
			return err
		}
	case kind.MovieEdition:
		if err := tx.movieEditionEnsureSlug(id); err != nil {
			return err
		}
		if err := tx.q.MovieEditionRestore(id); err != nil {
			return err
		}
	case kind.Series:
		if err := tx.seriesEnsureSlug(id); err != nil {
			return err
		}
		if err := tx.q.SeriesRestore(id); err != nil {
			return err
		}
	case kind.SeriesEdition:
		if err := tx.seriesEditionEnsureSlug(id); err != nil {
			return err
		}
		if err := tx.q.SeriesEditionRestore(id); err != nil {
			return err
		}
	case kind.Season:
		if err := tx.q.SeasonRestore(id); err != nil {
			return err
		}
	case kind.Episode:
		if err := tx.q.EpisodeRestore(id); err != nil {
			return err
		}
		restorable, err := tx.q.SeasonEpisodeListRestorableByEpisode(id)
		if err != nil {
			return err
		}
		sneps = restorable
		for _, snep := range sneps {
			if err := tx.seasonEpisodeSortKeyBump(snep.SeasonID, snep.SortKey); err != nil {
				return err
			}
		}
	case kind.Video:
		if err := tx.q.VideoRestore(id); err != nil {
			return err
		}
	case kind.Collection:
		if err := tx.collectionEnsureSlug(id); err != nil {
			return err
		}
		if err := tx.q.CollectionRestore(id); err != nil {
			return err
		}
	case kind.Download:
		if err := tx.q.DownloadRestore(schema.DownloadRestoreParams{
			LastActivityAt: time.Now().UnixMilli(),
			InfoHash:       id,
		}); err != nil {
			return err
		}
	}

	cascadeOf := &id
	for _, f := range []func(*string) error{
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
		if err := f(cascadeOf); err != nil {
			return err
		}
	}

	for _, f := range []func(*string) error{
		tx.q.SeasonEpisodeRestoreByCascade,
		tx.q.EpisodeVideoRestoreByCascade,
		tx.q.MovieVideoRestoreByCascade,
		tx.q.CollectionMovieRestoreByCascade,
		tx.q.CollectionSeriesRestoreByCascade,
	} {
		if err := f(cascadeOf); err != nil {
			return err
		}
	}

	for _, snep := range sneps {
		if err := tx.renumberSeason(snep.SeasonID); err != nil {
			return err
		}
	}

	return tx.q.TrashDelete(id)
}

// Purge hard-deletes a trashed item and all rows in its cascade
// subtree. Returns ErrNotTrashed if the target is currently live.
func (tx *TxRW) Purge(id string) (err error) {
	defer errorfmt.Handlef("purge %s: %w", id, &err)

	state, err := tx.trashState(id)
	if err != nil {
		return err
	}
	if state.live() {
		return ErrNotTrashed
	}
	if state.cascadeOf != "" {
		return ErrCascadeTrashed
	}

	cascadeOf := &id
	vids, err := tx.q.VideoListPurgeByCascade(cascadeOf)
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
	if err := tx.purgeVideoBlobs(vidIDs, origKeys); err != nil {
		return err
	}

	for _, f := range []func(*string) error{
		tx.q.EpisodeVideoPurgeByCascade,
		tx.q.MovieVideoPurgeByCascade,
		tx.q.SeasonEpisodePurgeByCascade,
		tx.q.CollectionMoviePurgeByCascade,
		tx.q.CollectionSeriesPurgeByCascade,
	} {
		if err := f(cascadeOf); err != nil {
			return err
		}
	}

	for _, f := range []func(*string) error{
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
		if err := f(cascadeOf); err != nil {
			return err
		}
	}

	return tx.q.TrashDelete(id)
}

func (tx *TxR) TrashItem(id string) *TrashItem {
	row := txmust1(tx.q.TrashGet(id))
	return newTrashItem(row)
}

func (tx *TxR) FindTrashItem(id string) (*TrashItem, bool) {
	row, ok := txfind1(tx.q.TrashGet(id))
	if !ok {
		return nil, false
	}
	return newTrashItem(row), true
}

func newTrashItem(row schema.Trash) *TrashItem {
	return &TrashItem{
		Kind:      txmust1(kind.ParseTrash(row.Kind)),
		ID:        row.ID,
		Title:     row.Title,
		Subtitle:  row.Subtitle,
		DeletedAt: time.UnixMilli(row.DeletedAt),
	}
}

// TrashList returns every directly-trashed entity (roots only),
// ordered newest-trashed first.
func (tx *TxR) TrashList() []TrashItem {
	rows := txmust1(tx.q.TrashList())
	items := make([]TrashItem, len(rows))
	for i, r := range rows {
		items[i] = *newTrashItem(r)
	}
	return items
}

func (tx *TxRW) trashPurge(threshold time.Time) (err error) {
	defer errorfmt.Handlef("trash purge: %w", &err)
	roots, err := tx.q.TrashRootsBefore(threshold.UnixMilli())
	if err != nil {
		return err
	}
	for _, rootID := range roots {
		if err := tx.Purge(rootID); err != nil {
			return err
		}
	}
	return nil
}

// purgeVideoBlobs deletes AudioTrack, SubtitleTrack and Rendition rows
// for the given videos and schedules their blob keys for removal on
// commit.
func (tx *TxRW) purgeVideoBlobs(vidIDs, origKeys []string) error {
	if len(vidIDs) == 0 {
		return nil
	}
	rendKeys, err := tx.q.RenditionListKeysByVideoIDs(vidIDs)
	if err != nil {
		return err
	}
	keys := append(origKeys, rendKeys...)
	audKeys, err := tx.q.AudioRenditionListKeysByVideoIDs(vidIDs)
	if err != nil {
		return err
	}
	keys = append(keys, audKeys...)
	for _, vid := range vidIDs {
		subs, err := tx.q.SubtitleTrackListByVideoID(vid)
		if err != nil {
			return err
		}
		for _, st := range subs {
			if st.OriginalKey != "" {
				keys = append(keys, st.OriginalKey)
			}
			if st.WebVTTKey != "" {
				keys = append(keys, st.WebVTTKey)
			}
		}
	}
	if len(keys) > 0 {
		tx.onCommit(func() {
			for _, k := range keys {
				tx.m.store.Remove(k)
			}
		})
	}
	if err := tx.q.AudioRenditionDeleteByVideoIDList(vidIDs); err != nil {
		return err
	}
	if err := tx.q.AudioTrackDeleteByVideoIDList(vidIDs); err != nil {
		return err
	}
	if err := tx.q.SubtitleTrackDeleteByVideoIDList(vidIDs); err != nil {
		return err
	}
	return tx.q.RenditionDeleteByVideoIDList(vidIDs)
}

func (m *Model) purgeTrashOnce(ctx context.Context) error {
	threshold := time.Now().Add(-trashRetention)
	return m.WithTxRW(ctx, func(tx *TxRW) error {
		return tx.trashPurge(threshold)
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

// KindOf returns the kind implied by a flurry ID prefix,
// or nil if the ID doesn't match a known prefix.
func KindOf(id string) kind.Trash {
	switch {
	case strings.HasPrefix(id, "med"):
		return kind.MovieEdition{}
	case strings.HasPrefix(id, "mo"):
		return kind.Movie{}
	case strings.HasPrefix(id, "sed"):
		return kind.SeriesEdition{}
	case strings.HasPrefix(id, "sn"):
		return kind.Season{}
	case strings.HasPrefix(id, "sr"):
		return kind.Series{}
	case strings.HasPrefix(id, "ep"):
		return kind.Episode{}
	case strings.HasPrefix(id, "vid"):
		return kind.Video{}
	case strings.HasPrefix(id, "col"):
		return kind.Collection{}
	}
	// Downloads have no flurry prefix; their ID is a 40-char hex SHA-1
	// info hash, which can't collide with any of the prefixes above.
	if len(id) == 40 {
		if _, err := hex.DecodeString(id); err == nil {
			return kind.Download{}
		}
	}
	return nil
}
