package model

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"maps"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/hekmon/transmissionrpc/v3"
	"kr.dev/errorfmt"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/log/logcontext"
	"ily.dev/act3/tlog"
	"ily.dev/act3/xstrings"
)

type DownloadHead struct {
	d       schema.Download
	planLen int

	filesLenOnce sync.Once
	filesLen     int
}

func (d *DownloadHead) ID() string        { return d.d.ID }
func (d *DownloadHead) State() string     { return d.d.State }
func (d *DownloadHead) Title() string     { return d.d.Title }
func (d *DownloadHead) Error() string     { return d.d.Error }
func (d *DownloadHead) Progress() float64 { return d.d.Progress }
func (d *DownloadHead) AutoImport() bool  { return d.d.Autoimport != 0 }

func (d *DownloadHead) EditorPath() string {
	return "/app/downloads/" + d.d.ID
}

func (d *DownloadHead) PlanLen() int { return d.planLen }

func (d *DownloadHead) FilesLen() int {
	d.filesLenOnce.Do(func() {
		_, info, err := ParseTorrent(d.d.Torrent)
		if err != nil {
			panic(err)
		}
		if info.IsDir() {
			d.filesLen = len(info.Files)
		} else {
			d.filesLen = 1
		}
	})
	return d.filesLen
}

type DownloadFile struct {
	d    *Download
	fi   metainfo.FileInfo
	plan *schema.DownloadPlan
	path string // set for single-file torrents
}

func (df *DownloadFile) Path() string {
	if df.path != "" {
		return df.path
	}
	return fiPath(&df.fi)
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
	if df.plan == nil {
		return ""
	}
	return df.plan.State
}

func (df *DownloadFile) Episode() *Episode {
	// TODO(april): return actual episode (not planned episode) after import
	sed := df.SeriesEdition()
	if sed == nil || df.plan == nil || df.plan.EpisodeID == nil {
		return nil
	}
	return sed.episodeByID(*df.plan.EpisodeID)
}

func (df *DownloadFile) Season() *Season {
	// TODO(april): return actual season (not planned season) after import
	sed := df.SeriesEdition()
	if sed == nil || df.plan == nil || df.plan.EpisodeID == nil {
		return nil
	}
	return sed.seasonByEpisodeID(*df.plan.EpisodeID)
}

func (df *DownloadFile) SeriesEdition() *SeriesEdition {
	// TODO(april): return actual series (not planned series) after import
	return df.d.PlanSeriesEdition()
}

type Download struct {
	DownloadHead
	metaInfo     *metainfo.MetaInfo
	info         metainfo.Info
	plans        []schema.DownloadPlan
	planByPath   map[string]*schema.DownloadPlan
	planEd       *SeriesEdition
	planMovieEd  *MovieEdition
	fileProgress map[string]float64 // path -> fraction [0,1], nil if unknown
}

type addr struct{ Snn, Enn int }

func (tx *TxR) newDownload(ctx Context, dl schema.Download) (*Download, error) {
	d := &Download{DownloadHead: DownloadHead{d: dl}}

	var err error
	d.metaInfo, err = metainfo.Load(bytes.NewReader(dl.Torrent))
	if err != nil {
		return nil, err
	}
	d.info, err = d.metaInfo.UnmarshalInfo()
	if err != nil {
		return nil, err
	}

	d.plans, err = tx.q.DownloadPlanListByDownloadID(ctx, dl.ID)
	if err != nil {
		return nil, err
	}
	d.planByPath = make(map[string]*schema.DownloadPlan, len(d.plans))
	for i := range d.plans {
		d.planByPath[d.plans[i].Path] = &d.plans[i]
	}
	d.planLen = len(d.plans)

	if dl.PlanSeriesEditionID != nil {
		d.planEd, err = tx.SeriesEdition(ctx, *dl.PlanSeriesEditionID)
		if err != nil {
			return nil, err
		}
	}
	if dl.PlanMovieEditionID != nil {
		d.planMovieEd, err = tx.MovieEdition(ctx, *dl.PlanMovieEditionID)
		if err != nil {
			return nil, err
		}
	}
	items := tx.m.prog.List("dl-" + dl.ID)
	if len(items) > 0 {
		d.fileProgress = map[string]float64{}
		for _, item := range items {
			d.fileProgress[item.Description()] = item.Progress()
		}
	}
	return d, nil
}

func (d *Download) PlanSeriesEdition() *SeriesEdition { return d.planEd }
func (d *Download) PlanMovieEdition() *MovieEdition   { return d.planMovieEd }

func (d *Download) PlanFor(path string) string {
	p := d.planByPath[path]
	if p == nil {
		return ""
	}
	if p.EpisodeID != nil {
		return *p.EpisodeID
	}
	if p.MovieEditionID != nil {
		return *p.MovieEditionID
	}
	return ""
}

func (d *Download) Paths() iter.Seq[string] {
	return func(yield func(string) bool) {
		if !d.info.IsDir() {
			yield(d.info.Name)
			return
		}
		for i := range d.info.Files {
			if !yield(fiPath(&d.info.Files[i])) {
				return
			}
		}
	}
}

func (d *Download) Files() []*DownloadFile {
	if !d.info.IsDir() {
		return []*DownloadFile{{
			d:    d,
			fi:   metainfo.FileInfo{Length: d.info.Length},
			plan: d.planByPath[d.info.Name],
			path: d.info.Name,
		}}
	}
	dfs := make([]*DownloadFile, len(d.info.Files))
	for i, fi := range d.info.Files {
		p := fiPath(&fi)
		dfs[i] = &DownloadFile{
			d:    d,
			fi:   fi,
			plan: d.planByPath[p],
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
		n, err := tx.q.DownloadPlanCountByDownloadID(context.Background(), dls[i].ID)
		if err != nil {
			return nil, err
		}
		res[i] = &DownloadHead{d: dls[i], planLen: int(n)}
	}
	return res, nil
}

func (tx *TxR) DownloadHeadList(ctx Context) ([]*DownloadHead, error) {
	return tx.newDownloadHeadList(tx.q.DownloadList(ctx))
}

func (tx *TxR) DownloadHeadListBySeriesEditionID(ctx Context, id string) ([]*DownloadHead, error) {
	return tx.newDownloadHeadList(tx.q.DownloadListByPlanSeriesEditionID(ctx, &id))
}

func (tx *TxR) DownloadHeadListByMovieEditionID(ctx Context, id string) ([]*DownloadHead, error) {
	return tx.newDownloadHeadList(tx.q.DownloadListByPlanMovieEditionID(ctx, &id))
}

func (tx *TxR) Download(ctx Context, id string) (*Download, error) {
	dl, err := tx.q.DownloadGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return tx.newDownload(ctx, dl)
}

func (tx *TxR) DownloadByInfoHash(ctx Context, hash string) (*Download, error) {
	dl, err := tx.q.DownloadGetByInfoHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	return tx.newDownload(ctx, dl)
}

func (tx *TxRW) DownloadAutoImportSet(ctx Context, id string, auto bool) error {
	v := int64(0)
	if auto {
		v = 1
	}
	_, err := tx.q.DownloadUpdateAutoImport(ctx, schema.DownloadUpdateAutoImportParams{
		Autoimport: v,
		ID:         id,
	})
	if err != nil {
		return err
	}
	if auto {
		return tx.DownloadImport(ctx, id)
	}
	return nil
}

// DownloadImport imports all planned files for a download.
// It fetches the torrent state from Transmission and imports
// each completed file that has a plan entry.
func (tx *TxRW) DownloadImport(ctx Context, id string) (err error) {
	defer errorfmt.Handlef("DownloadImport(%s): %w", id, &err)
	d, err := tx.Download(ctx, id)
	if err != nil {
		return err
	}
	tm := tx.m.transmission.Load()
	if tm == nil {
		return fmt.Errorf("no transmission client available")
	}
	ts, err := tm.TorrentGetAllForHashes(ctx, []string{d.d.InfoHash})
	if err != nil {
		return err
	}
	if len(ts) != 1 {
		return fmt.Errorf("torrent %s: got %d results, wanted 1", d.d.InfoHash, len(ts))
	}
	t := &ts[0]
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
	for path := range d.Paths() {
		plan := d.planByPath[path]
		if plan == nil || plan.State == "imported" || !done[path] {
			continue
		}
		err = tx.importDownloadPath(ctx, d, t, path)
		if err != nil {
			return err
		}
		err = tx.q.DownloadPlanUpdateState(ctx, schema.DownloadPlanUpdateStateParams{
			State:      "imported",
			DownloadID: d.ID(),
			Path:       path,
		})
		if err != nil {
			return err
		}
	}
	return tx.downloadUpdateState(ctx, d)
}

func (tx *TxRW) DownloadCreate(ctx Context, torrent io.Reader) (d *Download, err error) {
	defer errorfmt.Handlef("CreateDownload: %w", &err)
	b, err := io.ReadAll(torrent)
	if err != nil {
		return nil, err
	}
	mi, info, err := ParseTorrent(b)
	if err != nil {
		return nil, &ValidationError{
			Op:  "parse torrent",
			Err: err,
		}
	}
	infoHash := mi.HashInfoBytes().HexString()
	if dl, err := tx.q.DownloadGetByInfoHash(ctx, infoHash); err != nil && err != sql.ErrNoRows {
		return nil, err
	} else if err == nil {
		return tx.newDownload(ctx, dl)
	}
	dl, err := tx.q.DownloadCreate(ctx, schema.DownloadCreateParams{
		State:    "queued",
		Title:    info.Name,
		Torrent:  b,
		InfoHash: infoHash,
	})
	if err != nil {
		return nil, err
	}
	err = tx.addTask(ctx, taskAddDownloadToTransmission, dl.ID)
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
		_, err = tx.updateDownload(ctx, &ts[0])
		return err
	})
}

// downloadForPlanning fetches a download and validates it
// can still be planned (not already done/error),
// then parses the torrent info.
func (tx *TxRW) downloadForPlanning(ctx Context, id string) (schema.Download, *metainfo.Info, error) {
	dl, err := tx.q.DownloadGet(ctx, id)
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
	_, info, err := ParseTorrent(dl.Torrent)
	if err != nil {
		return dl, nil, &ValidationError{
			Op:  "parse torrent",
			Err: err,
		}
	}
	return dl, info, nil
}

func (tx *TxRW) DownloadCreatePlanSeries(ctx Context, id, sedID string) (d *Download, err error) {
	defer errorfmt.Handlef("CreateDownloadPlanSeries(%s, %s): %w", id, sedID, &err)
	dl, info, err := tx.downloadForPlanning(ctx, id)
	if err != nil {
		return nil, err
	}
	sed, err := tx.SeriesEdition(ctx, sedID)
	if err != nil {
		return nil, err
	}
	err = tx.q.DownloadPlanDeleteByDownloadID(ctx, dl.ID)
	if err != nil {
		return nil, err
	}
	for _, fi := range info.Files {
		p := fiPath(&fi)
		a, n := scanSpan(p)
		if n == 0 {
			continue
		}
		ep := sed.episodeByNumber(a.Snn, a.Enn)
		if ep == nil {
			continue
		}
		err = tx.q.DownloadPlanCreate(ctx, schema.DownloadPlanCreateParams{
			DownloadID: dl.ID,
			Path:       p,
			EpisodeID:  &ep.ep.ID,
		})
		if err != nil {
			return nil, err
		}
	}
	dl, err = tx.q.DownloadUpdatePlanSeries(ctx, schema.DownloadUpdatePlanSeriesParams{
		ID:                  dl.ID,
		PlanSeriesEditionID: &sedID,
	})
	if err != nil {
		return nil, err
	}
	return tx.newDownload(ctx, dl)
}

func (tx *TxRW) DownloadCreatePlanMovie(ctx Context, id, medID string) (d *Download, err error) {
	defer errorfmt.Handlef("CreateDownloadPlanMovie(%s, %s): %w", id, medID, &err)
	dl, info, err := tx.downloadForPlanning(ctx, id)
	if err != nil {
		return nil, err
	}
	err = tx.q.DownloadPlanDeleteByDownloadID(ctx, dl.ID)
	if err != nil {
		return nil, err
	}
	createPlan := func(path string) error {
		if !hasVideoExtension(path) {
			return nil
		}
		return tx.q.DownloadPlanCreate(ctx, schema.DownloadPlanCreateParams{
			DownloadID:     dl.ID,
			Path:           path,
			MovieEditionID: &medID,
		})
	}
	if !info.IsDir() {
		if err := createPlan(info.Name); err != nil {
			return nil, err
		}
	} else {
		for _, fi := range info.Files {
			if err := createPlan(fiPath(&fi)); err != nil {
				return nil, err
			}
		}
	}
	dl, err = tx.q.DownloadUpdatePlanMovie(ctx, schema.DownloadUpdatePlanMovieParams{
		ID:                 dl.ID,
		PlanMovieEditionID: &medID,
	})
	if err != nil {
		return nil, err
	}
	return tx.newDownload(ctx, dl)
}

func (tx *TxRW) updateDownload(ctx Context, t *transmissionrpc.Torrent) (schema.Download, error) {
	slog.InfoContext(ctx, "update-download", "hash", *t.HashString)

	d, err := tx.DownloadByInfoHash(ctx, *t.HashString)
	if err != nil {
		return schema.Download{}, err
	}
	ctx = logcontext.With(ctx, "dl", d.ID())

	if t.ErrorString != nil && *t.ErrorString != "" {
		return tx.q.DownloadUpdateError(ctx, schema.DownloadUpdateErrorParams{
			ID:    d.ID(),
			Error: *t.ErrorString,
		})
	}

	done := map[string]bool{}
	for _, tf := range t.Files {
		// Multi-file torrents have paths like "dirname/path";
		// single-file torrents have just the filename.
		p := tf.Name
		if _, after, ok := strings.Cut(tf.Name, "/"); ok {
			p = after
		}
		if tf.BytesCompleted == tf.Length {
			done[p] = true
		}
	}

	// Mark completed files as "downloaded".
	for path := range d.Paths() {
		plan := d.planByPath[path]
		if plan == nil || plan.State != "downloading" || !done[path] {
			continue
		}
		err = tx.q.DownloadPlanUpdateState(ctx, schema.DownloadPlanUpdateStateParams{
			State:      "downloaded",
			DownloadID: d.ID(),
			Path:       path,
		})
		if err != nil {
			return schema.Download{}, err
		}
		plan.State = "downloaded"
	}

	// Auto-import: import all downloaded files immediately.
	if d.AutoImport() {
		for path := range d.Paths() {
			plan := d.planByPath[path]
			if plan == nil || plan.State != "downloaded" {
				continue
			}
			err = tx.importDownloadPath(ctx, d, t, path)
			if err != nil {
				return schema.Download{}, err
			}
			err = tx.q.DownloadPlanUpdateState(ctx, schema.DownloadPlanUpdateStateParams{
				State:      "imported",
				DownloadID: d.ID(),
				Path:       path,
			})
			if err != nil {
				return schema.Download{}, err
			}
			plan.State = "imported"
		}
	}

	doneness := 0.0
	if t.PercentDone != nil {
		doneness = *t.PercentDone
	}
	_, err = tx.q.DownloadUpdateProgress(ctx, schema.DownloadUpdateProgressParams{
		ID:       d.ID(),
		State:    d.deriveState(),
		Progress: doneness,
	})
	if err != nil {
		return schema.Download{}, err
	}
	tx.onCommit(func() {
		tx.m.setInfoHashActive(*t.HashString, d.deriveState() == "downloading")
	})
	return d.d, nil
}

// deriveState computes the download-level state from plan states.
func (d *Download) deriveState() string {
	hasActive := false
	hasDownloading := false
	for _, p := range d.plans {
		switch p.State {
		case "downloading":
			hasDownloading = true
			hasActive = true
		case "downloaded":
			hasActive = true
		}
	}
	if !hasActive {
		return "imported"
	}
	if hasDownloading {
		return "downloading"
	}
	return "downloaded"
}

// downloadUpdateState derives the download state from plan states
// and updates it in the database.
func (tx *TxRW) downloadUpdateState(ctx Context, d *Download) error {
	state := d.deriveState()
	tx.onCommit(func() {
		tx.m.setInfoHashActive(d.d.InfoHash, state == "downloading")
	})
	_, err := tx.q.DownloadUpdateProgress(ctx, schema.DownloadUpdateProgressParams{
		ID:       d.ID(),
		State:    state,
		Progress: d.Progress(),
	})
	return err
}

func (tx *TxRW) importDownloadPath(ctx Context, d *Download, t *transmissionrpc.Torrent, path string) error {
	p := d.planByPath[path]
	if p == nil {
		return nil
	}
	if p.MovieEditionID != nil {
		return tx.importMovieVideo(ctx, t, *p.MovieEditionID, path)
	}
	if p.EpisodeID != nil {
		return tx.importEpisode(ctx, t, *p.EpisodeID, path)
	}
	return nil
}

func (tx *TxRW) importMovieVideo(ctx Context,
	t *transmissionrpc.Torrent,
	medID, path string,
) (err error) {
	defer errorfmt.Handlef("importMovieVideo(%s, %s): %w", path, medID, &err)
	vid, err := tx.importVideo(ctx, t, path)
	if err != nil {
		return err
	}
	_, err = tx.q.MovieVideoCreate(ctx, schema.MovieVideoCreateParams{
		MovieEditionID: medID,
		VideoID:        vid.ID,
	})
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "import movie", "med", medID, "vid", vid.ID)
	return nil
}

func (tx *TxRW) importEpisode(ctx Context,
	t *transmissionrpc.Torrent,
	epID, path string,
) (err error) {
	defer errorfmt.Handlef("importEpisode(%s, %s): %w", path, epID, &err)
	vid, err := tx.importVideo(ctx, t, path)
	if err != nil {
		return err
	}

	tx.m.prog.AddEdge(epID, vid.ID)
	_, err = tx.q.EpisodeVideoCreate(ctx, schema.EpisodeVideoCreateParams{
		EpisodeID: epID,
		VideoID:   vid.ID,
	})
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "import", "ep", epID, "vid", vid.ID)
	return nil
}

func (tx *TxRW) importVideo(ctx Context, t *transmissionrpc.Torrent, path string) (*schema.Video, error) {
	rel, err := tx.q.ReleaseGetByInfoHash(ctx, t.HashString)
	if err == sql.ErrNoRows {
		rel, err = tx.q.ReleaseCreate(ctx, schema.ReleaseCreateParams{
			Name:     *t.Name,
			InfoHash: t.HashString,
		})
	}
	if err != nil {
		return nil, err
	}

	vid, err := tx.q.VideoGetByReleasePath(ctx, schema.VideoGetByReleasePathParams{
		ReleaseID:   rel.ID,
		ReleasePath: path,
	})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	} else if err == nil {
		return &vid, nil
	}

	vid, err = tx.q.VideoCreate(ctx, schema.VideoCreateParams{
		ReleaseID:   rel.ID,
		ReleasePath: path,
	})
	if err != nil {
		return nil, err
	}

	diskPath, err := tx.transmissionDiskPath(ctx, t, path)
	if err != nil {
		return nil, err
	}
	err = tx.addTask(ctx, taskIngest, vid.ID, diskPath)
	if err != nil {
		return nil, err
	}
	return &vid, nil
}

func (m *Model) activeInfoHashes() []string {
	m.activeInfoHashMu.Lock()
	defer m.activeInfoHashMu.Unlock()
	return slices.Collect(maps.Keys(m.activeInfoHash))
}

func (m *Model) setInfoHashActive(hash string, active bool) {
	m.activeInfoHashMu.Lock()
	defer m.activeInfoHashMu.Unlock()
	if active {
		m.activeInfoHash[hash] = true
	} else {
		delete(m.activeInfoHash, hash)
	}
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

func (m *Model) pollTransmissionOnce() error {
	ctx := context.Background()
	active := m.activeInfoHashes()
	tm := m.transmission.Load()
	if len(active) == 0 || tm == nil {
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
			dl, err := tx.DownloadByInfoHash(ctx, *ts[i].HashString)
			if err != nil {
				return err
			}
			m.updateFileProgress(&ts[i], dl)
			_, err = tx.updateDownload(ctx, &ts[i])
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
		epID := dl.PlanFor(p)
		if epID == "" {
			continue
		}
		h := sha256.Sum256([]byte(p))
		key := "dlf-" + dl.ID() + "-" + hex.EncodeToString(h[:8])
		if tf.Length > 0 {
			frac := float64(tf.BytesCompleted) / float64(tf.Length)
			m.prog.AddEdge("dl-"+dl.ID(), key)
			m.prog.AddEdge(epID, key)
			m.prog.Open(key, p, "Downloading")
			m.prog.Update(key, frac)
			if tf.BytesCompleted == tf.Length {
				m.prog.Close(key, nil)
			}
		}
	}
}

func ParseTorrent(p []byte) (*metainfo.MetaInfo, *metainfo.Info, error) {
	mi, err := metainfo.Load(bytes.NewReader(p))
	if err != nil {
		return nil, nil, err
	}
	info, err := mi.UnmarshalInfo()
	if err != nil {
		return nil, nil, err
	}
	return mi, &info, nil
}

func fiPath(fi *metainfo.FileInfo) string {
	return path.Join(fi.BestPath()...)
}

var videoExtensions = map[string]bool{
	"mkv": true,
}

func hasVideoExtension(s string) bool {
	_, ext, found := xstrings.LastCut(s, ".")
	return found && videoExtensions[ext]
}

var addrSnnEnn = regexp.MustCompile(`\b[Ss](\d\d)[Ee](\d\d)\b`)

func scanSpan(s string) (addr, int) {
	if !hasVideoExtension(s) {
		return addr{}, 0
	}

	// TODO(april): scan more forms, incl range & list.
	m := addrSnnEnn.FindStringSubmatch(s)
	if m == nil {
		return addr{}, 0
	}
	return addr{
		Snn: mustParseInt(m[1]),
		Enn: mustParseInt(m[2]),
	}, 1
}

func mustParseInt(s string) int {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return int(n)
}
