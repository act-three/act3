package model

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/hekmon/transmissionrpc/v3"
	"kr.dev/errorfmt"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/log/logcontext"
	"ily.dev/act3/model/plan"
	"ily.dev/act3/tlog"
	"ily.dev/act3/xstrings"
)

type DownloadHead struct {
	d        schema.Download
	planLen  int
	filesLen int
}

func (d *DownloadHead) InfoHash() string  { return d.d.InfoHash }
func (d *DownloadHead) State() string     { return d.d.State }
func (d *DownloadHead) Title() string     { return d.d.Title }
func (d *DownloadHead) Error() string     { return d.d.Error }
func (d *DownloadHead) Progress() float64 { return d.d.Progress }
func (d *DownloadHead) AutoImport() bool  { return d.d.Autoimport != 0 }

func (d *DownloadHead) EditorPath() string {
	return "/app/downloads/" + d.d.InfoHash
}

func (d *DownloadHead) PlanLen() int  { return d.planLen }
func (d *DownloadHead) FilesLen() int { return d.filesLen }

type DownloadInfo struct {
	*DownloadHead
	sw *SeriesWork
	mw *MovieWork
}

func (di *DownloadInfo) SeriesWork() *SeriesWork { return di.sw }
func (di *DownloadInfo) MovieWork() *MovieWork   { return di.mw }

type DownloadFile struct {
	d     *Download
	fi    metainfo.FileInfo
	video *schema.Video
	path  string // set for single-file torrents
}

func (df *DownloadFile) Path() string {
	if df.path != "" {
		return df.path
	}
	return fiPath(&df.fi)
}

func (df *DownloadFile) HasVideoExtension() bool {
	return hasVideoExtension(df.Path())
}

// Progress returns the download progress of this file as a fraction (0-1),
// or -1 if progress data is not available.
func (df *DownloadFile) Progress() float64 {
	if df.d.fileProgress == nil {
		return -1
	}
	p, ok := df.d.fileProgress[df.Path()]
	if !ok {
		return -1
	}
	return p
}

func (df *DownloadFile) State() string {
	if df.video == nil {
		return ""
	}
	return df.video.State
}

func (df *DownloadFile) Episodes() []*Episode {
	sed := df.SeriesEdition()
	if sed == nil || df.video == nil {
		return nil
	}
	epIDs := df.d.epIDByVideoID[df.video.ID]
	if len(epIDs) == 0 {
		return nil
	}
	eps := make([]*Episode, 0, len(epIDs))
	for _, id := range epIDs {
		if ep := sed.episodeByID(id); ep != nil {
			eps = append(eps, ep)
		}
	}
	slices.SortFunc(eps, func(a, b *Episode) int {
		return cmp.Or(
			cmp.Compare(a.sn.sn.SortKey, b.sn.sn.SortKey),
			cmp.Compare(a.snep.SortKey, b.snep.SortKey),
		)
	})
	return eps
}

func (df *DownloadFile) InfoHash() string {
	return df.d.InfoHash()
}

func (df *DownloadFile) VideoID() string {
	if df.video == nil {
		return ""
	}
	return df.video.ID
}

func (df *DownloadFile) SeriesEdition() *SeriesEdition {
	return df.d.SeriesEdition()
}

type Download struct {
	DownloadHead
	metaInfo       *metainfo.MetaInfo
	info           metainfo.Info
	videos         []schema.Video
	videoByName    map[string]*schema.Video
	epIDByVideoID  map[string][]string // videoID -> episodeIDs
	medIDByVideoID map[string]string   // videoID -> movieEditionID
	planEd         *SeriesEdition
	planMovieEd    *MovieEdition
	fileProgress   map[string]float64 // path -> fraction [0,1], nil if unknown
}

func (tx *TxR) newDownload(ctx Context, dl schema.Download) (*Download, error) {
	d := &Download{DownloadHead: DownloadHead{d: dl}}

	var err error
	mi, info, err := parseTorrent(dl.Torrent)
	if err != nil {
		return nil, err
	}
	d.metaInfo = mi
	d.info = *info
	d.filesLen = filesLen(info)

	d.videos, err = tx.q.VideoListByInfoHash(ctx, &dl.InfoHash)
	if err != nil {
		return nil, err
	}
	d.videoByName = make(map[string]*schema.Video, len(d.videos))
	for i := range d.videos {
		d.videoByName[d.videos[i].Name] = &d.videos[i]
	}
	d.planLen = len(d.videos)

	evs, err := tx.q.EpisodeVideoListByInfoHash(ctx, &dl.InfoHash)
	if err != nil {
		return nil, err
	}
	d.epIDByVideoID = make(map[string][]string, len(evs))
	for _, ev := range evs {
		d.epIDByVideoID[ev.VideoID] = append(d.epIDByVideoID[ev.VideoID], ev.EpisodeID)
	}

	mvs, err := tx.q.MovieVideoListByInfoHash(ctx, &dl.InfoHash)
	if err != nil {
		return nil, err
	}
	d.medIDByVideoID = make(map[string]string, len(mvs))
	for _, mv := range mvs {
		d.medIDByVideoID[mv.VideoID] = mv.MovieEditionID
	}

	if dl.SeriesEditionID != nil {
		d.planEd, err = tx.SeriesEdition(ctx, *dl.SeriesEditionID)
		if err != nil {
			return nil, err
		}
	}
	if dl.MovieEditionID != nil {
		d.planMovieEd, err = tx.MovieEdition(ctx, *dl.MovieEditionID)
		if err != nil {
			return nil, err
		}
	}
	items := tx.m.prog.List("dl-" + dl.InfoHash)
	if len(items) > 0 {
		d.fileProgress = map[string]float64{}
		for _, item := range items {
			d.fileProgress[item.Description()] = item.Progress()
		}
	}
	return d, nil
}

func (d *Download) SeriesEdition() *SeriesEdition { return d.planEd }

func (d *Download) Work() Work {
	if d.planMovieEd != nil {
		return &MovieWork{
			MovieHead:        *d.planMovieEd.MovieHead(),
			MovieEditionHead: d.planMovieEd.MovieEditionHead,
		}
	}
	if d.planEd != nil {
		return &SeriesWork{
			SeriesHead:        *d.planEd.SeriesHead(),
			SeriesEditionHead: d.planEd.SeriesEditionHead,
		}
	}
	return nil
}

func (d *Download) EpisodeIDsByVideoID(videoID string) []string {
	return d.epIDByVideoID[videoID]
}

func (tx *TxR) VideoGetByName(ctx Context, infoHash, name string) (*Video, error) {
	v, err := tx.q.VideoGetByName(ctx, schema.VideoGetByNameParams{
		InfoHash: &infoHash,
		Name:     name,
	})
	if err != nil {
		return nil, err
	}
	return &Video{v: v}, nil
}

func (d *Download) PlanFor(path string) []string {
	v := d.videoByName[path]
	if v == nil {
		return nil
	}
	if epIDs := d.epIDByVideoID[v.ID]; len(epIDs) > 0 {
		return epIDs
	}
	if medID, ok := d.medIDByVideoID[v.ID]; ok {
		return []string{medID}
	}
	return nil
}

func (d *Download) Files() []*DownloadFile {
	if !d.info.IsDir() {
		return []*DownloadFile{{
			d:     d,
			fi:    metainfo.FileInfo{Length: d.info.Length},
			video: d.videoByName[d.info.Name],
			path:  d.info.Name,
		}}
	}
	dfs := make([]*DownloadFile, len(d.info.Files))
	for i, fi := range d.info.Files {
		p := fiPath(&fi)
		dfs[i] = &DownloadFile{
			d:     d,
			fi:    fi,
			video: d.videoByName[p],
		}
	}
	return dfs
}

func (tx *TxR) newDownloadHeadList(dls []schema.Download, err error) ([]*DownloadHead, error) {
	if err != nil {
		return nil, err
	}
	res := make([]*DownloadHead, len(dls))
	for i := range dls {
		n, err := tx.q.VideoCountByInfoHash(context.Background(), &dls[i].InfoHash)
		if err != nil {
			return nil, err
		}
		_, info, err := parseTorrent(dls[i].Torrent)
		if err != nil {
			return nil, err
		}
		res[i] = &DownloadHead{d: dls[i], planLen: int(n), filesLen: filesLen(info)}
	}
	return res, nil
}

func (tx *TxR) DownloadHeadList(ctx Context) ([]*DownloadHead, error) {
	return tx.newDownloadHeadList(tx.q.DownloadList(ctx))
}

func (tx *TxR) DownloadInfoList(ctx Context) ([]*DownloadInfo, error) {
	heads, err := tx.DownloadHeadList(ctx)
	if err != nil {
		return nil, err
	}
	sedList, err := tx.q.SeriesEditionListByDownload(ctx)
	if err != nil {
		return nil, err
	}
	srList, err := tx.q.SeriesListByDownload(ctx)
	if err != nil {
		return nil, err
	}
	srs := make(map[string]*SeriesHead, len(srList))
	for i := range srList {
		srs[srList[i].ID] = &SeriesHead{srList[i]}
	}
	sws := make(map[string]*SeriesWork, len(sedList))
	for i := range sedList {
		sed := &SeriesEditionHead{sed: sedList[i]}
		sws[sedList[i].ID] = &SeriesWork{
			SeriesHead:        *srs[sedList[i].SeriesID],
			SeriesEditionHead: *sed,
		}
	}
	medList, err := tx.q.MovieEditionListByDownload(ctx)
	if err != nil {
		return nil, err
	}
	moList, err := tx.q.MovieListByDownload(ctx)
	if err != nil {
		return nil, err
	}
	mos := make(map[string]*MovieHead, len(moList))
	for i := range moList {
		mos[moList[i].ID] = &MovieHead{mo: moList[i]}
	}
	mws := make(map[string]*MovieWork, len(medList))
	for i := range medList {
		med := &MovieEditionHead{med: medList[i]}
		mws[medList[i].ID] = &MovieWork{
			MovieHead:        *mos[medList[i].MovieID],
			MovieEditionHead: *med,
		}
	}
	res := make([]*DownloadInfo, len(heads))
	for i, h := range heads {
		di := &DownloadInfo{DownloadHead: h}
		if id := h.d.SeriesEditionID; id != nil {
			di.sw = sws[*id]
		}
		if id := h.d.MovieEditionID; id != nil {
			di.mw = mws[*id]
		}
		res[i] = di
	}
	return res, nil
}

func (tx *TxR) DownloadHeadListBySeriesEditionID(ctx Context, id string) ([]*DownloadHead, error) {
	return tx.newDownloadHeadList(tx.q.DownloadListBySeriesEditionID(ctx, &id))
}

func (tx *TxR) DownloadHeadListByMovieEditionID(ctx Context, id string) ([]*DownloadHead, error) {
	return tx.newDownloadHeadList(tx.q.DownloadListByMovieEditionID(ctx, &id))
}

func (tx *TxR) Download(ctx Context, infoHash string) (*Download, error) {
	dl, err := tx.q.DownloadGet(ctx, infoHash)
	if err != nil {
		return nil, err
	}
	return tx.newDownload(ctx, dl)
}

func (tx *TxR) DownloadInfo(ctx Context, infoHash string) (*DownloadInfo, error) {
	dl, err := tx.Download(ctx, infoHash)
	if err != nil {
		return nil, err
	}
	di := &DownloadInfo{DownloadHead: &dl.DownloadHead}
	if dl.planMovieEd != nil {
		di.mw = &MovieWork{
			MovieHead:        *dl.planMovieEd.MovieHead(),
			MovieEditionHead: dl.planMovieEd.MovieEditionHead,
		}
	}
	if dl.planEd != nil {
		di.sw = &SeriesWork{
			SeriesHead:        *dl.planEd.SeriesHead(),
			SeriesEditionHead: dl.planEd.SeriesEditionHead,
		}
	}
	return di, nil
}

func (tx *TxRW) DownloadAutoImportSet(ctx Context, infoHash string, auto bool) (err error) {
	defer errorfmt.Handlef("DownloadAutoImportSet(%s, %v): %w", infoHash, auto, &err)
	v := int64(0)
	if auto {
		v = 1
	}
	_, err = tx.q.DownloadUpdateAutoImport(ctx, schema.DownloadUpdateAutoImportParams{
		Autoimport:     v,
		LastActivityAt: time.Now().UnixMilli(),
		InfoHash:       infoHash,
	})
	if err != nil {
		return err
	}
	return tx.processDownload(ctx, infoHash)
}

// EpisodeAttachToggleAddr returns the live-update addr for the
// SettingsToggle that attaches an episode to a download file.
func EpisodeAttachToggleAddr(infoHash, filePath, episodeID string) []string {
	return []string{"episode-attach", infoHash, filePath, episodeID}
}

// EpisodeVideoSet adds or removes the link between a download file
// and an episode. Returns a clear error if the Video row no longer
// exists — typically because duplicate-content ingest merged it into
// another Video — so a stale UI click surfaces as a user-facing
// message rather than a raw sqlite miss.
func (tx *TxRW) EpisodeVideoSet(ctx Context, infoHash, filePath, episodeID string, attach bool) (err error) {
	defer errorfmt.Handlef("EpisodeVideoSet: %w", &err)
	vid, err := tx.q.VideoGetByName(ctx, schema.VideoGetByNameParams{
		InfoHash: &infoHash,
		Name:     filePath,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("no video for %s/%s (merged into another duplicate?)", infoHash, filePath)
	}
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		tx.m.emitEvent(&Event{
			Type:    EventDownloadFileAttach,
			ID:      infoHash,
			NewText: filePath,
		})
		tx.m.emitEvent(&Event{
			Type:    EventLiveUpdate,
			Addr:    EpisodeAttachToggleAddr(infoHash, filePath, episodeID),
			NewText: strconv.FormatBool(attach),
		})
	})
	if attach {
		if err := tx.q.EpisodeVideoEnsure(ctx, schema.EpisodeVideoEnsureParams{
			EpisodeID: episodeID,
			VideoID:   vid.ID,
		}); err != nil {
			return err
		}
	} else {
		if err := tx.q.EpisodeVideoDelete(ctx, schema.EpisodeVideoDeleteParams{
			EpisodeID: episodeID,
			VideoID:   vid.ID,
		}); err != nil {
			return err
		}
	}
	return tx.bumpDownloadActivity(ctx, infoHash)
}

// bumpDownloadActivity updates LastActivityAt on the live Download
// with the given InfoHash to time.Now(). No-op when infoHash is empty
// or the Download has been trashed. Call after any mutation of an
// EpisodeVideo or MovieVideo junction owned by a Download, so that
// user curation counts as activity against the 7-day auto-trash timer.
func (tx *TxRW) bumpDownloadActivity(ctx Context, infoHash string) error {
	if infoHash == "" {
		return nil
	}
	return tx.q.DownloadBumpActivity(ctx, schema.DownloadBumpActivityParams{
		LastActivityAt: time.Now().UnixMilli(),
		InfoHash:       infoHash,
	})
}

// DownloadImport is the manual import action: it enables auto-import,
// which triggers ingest for any already-complete files. Once auto-import
// is on, future polling cycles also ingest files as they complete.
func (tx *TxRW) DownloadImport(ctx Context, infoHash string) error {
	return tx.DownloadAutoImportSet(ctx, infoHash, true)
}

// processDownload reconciles a download against its current torrent
// status: queues ingest tasks for any pending videos whose files are
// complete (when auto-import is enabled), then writes the derived state
// and progress back to the database.
//
// The torrent status is read from an in-memory cache populated by
// polling. If the cache is empty for this download, processDownload is
// a no-op: a future polling cycle will repopulate the cache and
// trigger any imports that have become possible.
func (tx *TxRW) processDownload(ctx Context, infoHash string) (err error) {
	defer errorfmt.Handlef("processDownload(%s): %w", infoHash, &err)
	t := tx.m.getTorrent(infoHash)
	if t == nil {
		return nil
	}

	d, err := tx.Download(ctx, infoHash)
	if err != nil {
		return err
	}
	done := torrentDone(t)

	if d.AutoImport() {
		for i := range d.videos {
			v := &d.videos[i]
			if v.State != "pending" || !done[v.Name] {
				continue
			}
			err = tx.q.VideoUpdateState(ctx, schema.VideoUpdateStateParams{
				State: "importing",
				ID:    v.ID,
			})
			if err != nil {
				return err
			}
			for _, epID := range d.epIDByVideoID[v.ID] {
				tx.m.prog.AddEdge(epID, v.ID)
			}
			err = tx.addTask(ctx, taskIngest, v.ID, *t.DownloadDir, torrentRelPath(d.d.Title, v.Name))
			if err != nil {
				return err
			}
			v.State = "importing"
		}
	}

	state := d.deriveState(done)
	progress := 0.0
	if t.PercentDone != nil {
		progress = *t.PercentDone
	}
	_, err = tx.q.DownloadUpdateProgress(ctx, schema.DownloadUpdateProgressParams{
		State:          state,
		Progress:       progress,
		LastActivityAt: time.Now().UnixMilli(),
		InfoHash:       infoHash,
	})
	return err
}

// torrentDone returns a map of relative file paths to completion status
// for the files in a Transmission torrent.
func torrentDone(t *transmissionrpc.Torrent) map[string]bool {
	done := map[string]bool{}
	for _, tf := range t.Files {
		p := tf.Name
		if _, after, ok := strings.Cut(tf.Name, "/"); ok {
			p = after
		}
		if tf.BytesCompleted == tf.Length {
			done[p] = true
		}
	}
	return done
}

func (tx *TxRW) DownloadCreate(ctx Context, torrent io.Reader, sedID, medID *string) (d *Download, err error) {
	defer errorfmt.Handlef("CreateDownload: %w", &err)
	b, err := io.ReadAll(torrent)
	if err != nil {
		return nil, err
	}
	mi, info, err := parseTorrent(b)
	if err != nil {
		return nil, &ValidationError{
			Op:  "parse torrent",
			Err: err,
		}
	}
	infoHash := mi.HashInfoBytes().HexString()
	dl, err := tx.q.DownloadGet(ctx, infoHash)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil && dl.DeletedAt == nil {
		// Already live: nothing to do.
		return tx.newDownload(ctx, dl)
	} else if err == nil && dl.DeletedAt != nil {
		// Re-add of a trashed Download: restore rather than fail on the
		// unique InfoHash PK. Restore rehydrates any orphan-reaped Videos
		// under this Download's cascade, so we don't need to create new
		// Video rows. Overwrite the edition targeting unconditionally to
		// match the new caller's intent.
		if err := tx.Restore(ctx, infoHash); err != nil {
			return nil, err
		}
		if err := tx.q.DownloadUpdateTargeting(ctx, schema.DownloadUpdateTargetingParams{
			SeriesEditionID: sedID,
			MovieEditionID:  medID,
			InfoHash:        infoHash,
		}); err != nil {
			return nil, err
		}
		dl, err = tx.q.DownloadGet(ctx, infoHash)
		if err != nil {
			return nil, err
		}
		return tx.newDownload(ctx, dl)
	}
	dl, err = tx.q.DownloadCreate(ctx, schema.DownloadCreateParams{
		InfoHash:        infoHash,
		State:           "queued",
		Title:           info.Name,
		Torrent:         b,
		SeriesEditionID: sedID,
		MovieEditionID:  medID,
	})
	if err != nil {
		return nil, err
	}
	createVideos := func(path string) error {
		if !hasVideoExtension(path) {
			return nil
		}
		_, err := tx.q.VideoCreate(ctx, schema.VideoCreateParams{
			InfoHash: &dl.InfoHash,
			Name:     path,
		})
		return err
	}
	if !info.IsDir() {
		if err := createVideos(info.Name); err != nil {
			return nil, err
		}
	} else {
		for _, fi := range info.Files {
			if err := createVideos(fiPath(&fi)); err != nil {
				return nil, err
			}
		}
	}
	err = tx.addTask(ctx, taskAddDownloadToTransmission, dl.InfoHash)
	if err != nil {
		return nil, err
	}
	return tx.newDownload(ctx, dl)
}

func (tx *TxR) taskAddDownloadToTransmission(ctx Context, args []string) error {
	tm := tx.m.transmission.Load()
	if tm == nil {
		return fmt.Errorf("no transmission client available")
	}

	dl, err := tx.q.DownloadGet(ctx, args[0])
	if err != nil {
		return err
	}
	s := base64.StdEncoding.EncodeToString(dl.Torrent)
	t, err := tm.TorrentAdd(ctx, transmissionrpc.TorrentAddPayload{
		Labels:   []string{"act3"},
		MetaInfo: &s,
	})
	if err != nil {
		return err
	}
	ts, err := tm.TorrentGetAllForHashes(ctx, []string{*t.HashString})
	if err != nil {
		return err
	} else if len(ts) != 1 {
		return fmt.Errorf("%s: got %d torrents, wanted 1", *t.HashString, len(ts))
	}
	return tx.m.WithTxRW(func(tx *TxRW) error {
		return tx.updateDownload(ctx, &ts[0])
	})
}

// downloadForPlanning fetches a download and validates it
// can still be planned (not already done/error),
// then parses the torrent info.
func (tx *TxRW) downloadForPlanning(ctx Context, infoHash string) (schema.Download, *metainfo.Info, error) {
	dl, err := tx.q.DownloadGet(ctx, infoHash)
	if err != nil {
		return dl, nil, err
	}
	switch dl.State {
	case "downloaded", "imported", "error":
		return dl, nil, &ValidationError{
			Op:  "create download",
			Err: errors.New("already imported"),
		}
	}
	_, info, err := parseTorrent(dl.Torrent)
	if err != nil {
		return dl, nil, &ValidationError{
			Op:  "parse torrent",
			Err: err,
		}
	}
	return dl, info, nil
}

func (tx *TxRW) DownloadCreatePlanSeries(ctx Context, infoHash, sedID string) (d *Download, err error) {
	defer errorfmt.Handlef("DownloadCreatePlanSeries(%s, %s): %w", infoHash, sedID, &err)
	dl, info, err := tx.downloadForPlanning(ctx, infoHash)
	if err != nil {
		return nil, err
	}
	sed, err := tx.SeriesEdition(ctx, sedID)
	if err != nil {
		return nil, err
	}
	var planEps []plan.Episode
	for ep := range sed.Episodes(AnyEpisode) {
		planEps = append(planEps, ep)
	}
	planner := plan.NewPlanner(planEps)
	for _, fi := range info.Files {
		p := fiPath(&fi)
		if !hasVideoExtension(p) {
			continue
		}
		epIDs := planner.Plan(p)
		if len(epIDs) == 0 {
			continue
		}
		vid, err := tx.q.VideoGetByName(ctx, schema.VideoGetByNameParams{
			InfoHash: &dl.InfoHash,
			Name:     p,
		})
		if err != nil {
			return nil, err
		}
		for _, epID := range epIDs {
			_, err = tx.q.EpisodeVideoCreate(ctx, schema.EpisodeVideoCreateParams{
				EpisodeID: epID,
				VideoID:   vid.ID,
			})
			if err != nil {
				return nil, err
			}
		}
	}
	if err := tx.bumpDownloadActivity(ctx, dl.InfoHash); err != nil {
		return nil, err
	}
	return tx.Download(ctx, dl.InfoHash)
}

func (tx *TxRW) DownloadCreatePlanMovie(ctx Context, infoHash, medID string) (d *Download, err error) {
	defer errorfmt.Handlef("DownloadCreatePlanMovie(%s, %s): %w", infoHash, medID, &err)
	dl, info, err := tx.downloadForPlanning(ctx, infoHash)
	if err != nil {
		return nil, err
	}
	linkVideo := func(path string) error {
		if !hasVideoExtension(path) {
			return nil
		}
		vid, err := tx.q.VideoGetByName(ctx, schema.VideoGetByNameParams{
			InfoHash: &dl.InfoHash,
			Name:     path,
		})
		if err != nil {
			return err
		}
		_, err = tx.q.MovieVideoCreate(ctx, schema.MovieVideoCreateParams{
			MovieEditionID: medID,
			VideoID:        vid.ID,
		})
		return err
	}
	if !info.IsDir() {
		if err := linkVideo(info.Name); err != nil {
			return nil, err
		}
	} else {
		for _, fi := range info.Files {
			if err := linkVideo(fiPath(&fi)); err != nil {
				return nil, err
			}
		}
	}
	if err := tx.bumpDownloadActivity(ctx, dl.InfoHash); err != nil {
		return nil, err
	}
	return tx.Download(ctx, dl.InfoHash)
}

func (tx *TxRW) updateDownload(ctx Context, t *transmissionrpc.Torrent) error {
	infoHash := *t.HashString
	ctx = logcontext.With(ctx, "dl", infoHash)
	slog.InfoContext(ctx, "update-download", "hash", infoHash)

	if t.ErrorString != nil && *t.ErrorString != "" {
		_, err := tx.q.DownloadUpdateError(ctx, schema.DownloadUpdateErrorParams{
			Error:          *t.ErrorString,
			LastActivityAt: time.Now().UnixMilli(),
			InfoHash:       infoHash,
		})
		return err
	}

	tx.m.setTorrent(infoHash, t)
	return tx.processDownload(ctx, infoHash)
}

func (d *Download) deriveState(done map[string]bool) string {
	hasIncomplete := false
	hasReady := false
	for i := range d.videos {
		v := &d.videos[i]
		if v.State != "pending" {
			continue
		}
		if done[v.Name] {
			hasReady = true
		} else {
			hasIncomplete = true
		}
	}
	if hasIncomplete {
		return "downloading"
	}
	if hasReady {
		return "downloaded"
	}
	return "imported"
}

func (m *Model) getTorrent(infoHash string) *transmissionrpc.Torrent {
	m.torrentMu.Lock()
	defer m.torrentMu.Unlock()
	return m.torrent[infoHash]
}

func (m *Model) setTorrent(infoHash string, t *transmissionrpc.Torrent) {
	m.torrentMu.Lock()
	defer m.torrentMu.Unlock()
	m.torrent[infoHash] = t
}

func (m *Model) pollTransission() {
	for {
		time.Sleep(time.Minute)
		err := m.pollTransmissionOnce()
		if err != nil {
			slog.Error("error", "error", err)
		}
	}
}

// downloadIdleTimeout is how long a terminal-state Download stays live
// before being auto-trashed.
const downloadIdleTimeout = 7 * 24 * time.Hour

func (m *Model) autoTrashDownloadsLoop() {
	for {
		time.Sleep(time.Hour)
		if err := m.autoTrashDownloadsOnce(context.Background()); err != nil {
			slog.Error("auto-trash downloads", "error", err)
		}
	}
}

func (m *Model) autoTrashDownloadsOnce(ctx Context) error {
	threshold := time.Now().Add(-downloadIdleTimeout).UnixMilli()
	return m.WithTxRW(func(tx *TxRW) error {
		infoHashes, err := tx.q.DownloadListAutoTrashCandidates(ctx, threshold)
		if err != nil {
			return err
		}
		for _, ih := range infoHashes {
			if err := tx.Trash(ctx, ih); err != nil && !errors.Is(err, ErrAlreadyTrashed) {
				return err
			}
		}
		return nil
	})
}

func (m *Model) pollTransmissionOnce() error {
	ctx := context.Background()
	tm := m.transmission.Load()
	if tm == nil {
		return nil
	}
	var active []string
	if err := m.WithTxR(func(tx *TxR) error {
		var err error
		active, err = tx.q.DownloadListInfoHashesActive(ctx)
		return err
	}); err != nil {
		return err
	}
	if len(active) == 0 {
		return nil
	}
	defer tlog.Elapsed(ctx, "poll-transmission", "active", len(active))()
	ts, err := tm.TorrentGetAllForHashes(ctx, active)
	if err != nil {
		return err
	}
	return m.WithTxRW(func(tx *TxRW) error {
		for i := range ts {
			if ts[i].HashString == nil {
				continue
			}
			dl, err := tx.Download(ctx, *ts[i].HashString)
			if err != nil {
				return err
			}
			m.updateFileProgress(&ts[i], dl)
			err = tx.updateDownload(ctx, &ts[i])
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// updateFileProgress updates the progress tracker with per-file download
// progress from a Transmission torrent response.
func (m *Model) updateFileProgress(t *transmissionrpc.Torrent, dl *Download) {
	for _, tf := range t.Files {
		_, p, _ := strings.Cut(tf.Name, "/")
		ids := dl.PlanFor(p)
		if len(ids) == 0 {
			continue
		}
		h := sha256.Sum256([]byte(p))
		key := "dlf-" + dl.InfoHash() + "-" + hex.EncodeToString(h[:8])
		if tf.Length > 0 {
			frac := float64(tf.BytesCompleted) / float64(tf.Length)
			m.prog.AddEdge("dl-"+dl.InfoHash(), key)
			for _, id := range ids {
				m.prog.AddEdge(id, key)
			}
			m.prog.Open(key, p, "Downloading")
			m.prog.Update(key, frac)
			if tf.BytesCompleted == tf.Length {
				m.prog.Close(key, nil)
			}
		}
	}
}

func parseTorrent(p []byte) (*metainfo.MetaInfo, *metainfo.Info, error) {
	mi, err := metainfo.Load(bytes.NewReader(p))
	if err != nil {
		return nil, nil, err
	}
	info, err := mi.UnmarshalInfo()
	if err != nil {
		return nil, nil, err
	}
	if err := checkTorrentPaths(&info); err != nil {
		return nil, nil, err
	}
	return mi, &info, nil
}

// checkTorrentPaths rejects torrents whose declared paths are not
// local-relative. We consult fi.Path rather than fi.BestPath because
// Transmission uses Path for on-disk layout; a Path/PathUtf8 divergence
// would otherwise be a covert channel for presenting us a different
// filename than Transmission wrote to disk.
func checkTorrentPaths(info *metainfo.Info) error {
	if !isLocalPathParts([]string{info.Name}) {
		return fmt.Errorf("torrent info.Name %q is not a safe local path", info.Name)
	}
	for i, fi := range info.Files {
		if !isLocalPathParts(fi.Path) {
			return fmt.Errorf("torrent file %d path %q is not a safe local path", i, fi.Path)
		}
	}
	return nil
}

func isLocalPathParts(parts []string) bool {
	if len(parts) == 0 {
		return false
	}
	return filepath.IsLocal(filepath.Join(parts...))
}

func filesLen(info *metainfo.Info) int {
	if info.IsDir() {
		return len(info.Files)
	}
	return 1
}

// fiPath is safe to feed to filesystem ops because parseTorrent rejects
// non-local fi.Path.
func fiPath(fi *metainfo.FileInfo) string {
	return filepath.Join(fi.Path...)
}

var videoExtensions = map[string]bool{
	"mkv": true,
	"mp4": true,
}

func hasVideoExtension(s string) bool {
	_, ext, found := xstrings.LastCut(s, ".")
	return found && videoExtensions[ext]
}
