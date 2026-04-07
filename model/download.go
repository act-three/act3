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

func (d *DownloadHead) InfoHash() string  { return d.d.InfoHash }
func (d *DownloadHead) State() string     { return d.d.State }
func (d *DownloadHead) Title() string     { return d.d.Title }
func (d *DownloadHead) Error() string     { return d.d.Error }
func (d *DownloadHead) Progress() float64 { return d.d.Progress }
func (d *DownloadHead) AutoImport() bool  { return d.d.Autoimport != 0 }

func (d *DownloadHead) EditorPath() string {
	return "/app/downloads/" + d.d.InfoHash
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

func (df *DownloadFile) Episode() *Episode {
	sed := df.SeriesEdition()
	if sed == nil || df.video == nil {
		return nil
	}
	epID, ok := df.d.epIDByVideoID[df.video.ID]
	if !ok {
		return nil
	}
	return sed.episodeByID(epID)
}

func (df *DownloadFile) Season() *Season {
	sed := df.SeriesEdition()
	if sed == nil || df.video == nil {
		return nil
	}
	epID, ok := df.d.epIDByVideoID[df.video.ID]
	if !ok {
		return nil
	}
	return sed.seasonByEpisodeID(epID)
}

func (df *DownloadFile) SeriesEdition() *SeriesEdition {
	return df.d.PlanSeriesEdition()
}

type Download struct {
	DownloadHead
	metaInfo       *metainfo.MetaInfo
	info           metainfo.Info
	videos         []schema.Video
	videoByName    map[string]*schema.Video
	epIDByVideoID  map[string]string // videoID -> episodeID
	medIDByVideoID map[string]string // videoID -> movieEditionID
	planEd         *SeriesEdition
	planMovieEd    *MovieEdition
	fileProgress   map[string]float64 // path -> fraction [0,1], nil if unknown
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
	d.epIDByVideoID = make(map[string]string, len(evs))
	for _, ev := range evs {
		d.epIDByVideoID[ev.VideoID] = ev.EpisodeID
	}

	mvs, err := tx.q.MovieVideoListByInfoHash(ctx, &dl.InfoHash)
	if err != nil {
		return nil, err
	}
	d.medIDByVideoID = make(map[string]string, len(mvs))
	for _, mv := range mvs {
		d.medIDByVideoID[mv.VideoID] = mv.MovieEditionID
	}

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
	items := tx.m.prog.List("dl-" + dl.InfoHash)
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
	v := d.videoByName[path]
	if v == nil {
		return ""
	}
	if epID, ok := d.epIDByVideoID[v.ID]; ok {
		return epID
	}
	if medID, ok := d.medIDByVideoID[v.ID]; ok {
		return medID
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

func (tx *TxR) Download(ctx Context, infoHash string) (*Download, error) {
	dl, err := tx.q.DownloadGet(ctx, infoHash)
	if err != nil {
		return nil, err
	}
	return tx.newDownload(ctx, dl)
}

func (tx *TxRW) DownloadAutoImportSet(ctx Context, infoHash string, auto bool) error {
	v := int64(0)
	if auto {
		v = 1
	}
	_, err := tx.q.DownloadUpdateAutoImport(ctx, schema.DownloadUpdateAutoImportParams{
		Autoimport: v,
		InfoHash:   infoHash,
	})
	if err != nil {
		return err
	}
	if auto {
		return tx.DownloadImport(ctx, infoHash)
	}
	return nil
}

func (tx *TxRW) DownloadImport(ctx Context, infoHash string) (err error) {
	defer errorfmt.Handlef("DownloadImport(%s): %w", infoHash, &err)
	d, err := tx.Download(ctx, infoHash)
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
		if epID, ok := d.epIDByVideoID[v.ID]; ok {
			tx.m.prog.AddEdge(epID, v.ID)
		}
		diskPath, err := tx.transmissionDiskPath(ctx, t, v.Name)
		if err != nil {
			return err
		}
		err = tx.addTask(ctx, taskIngest, v.ID, diskPath)
		if err != nil {
			return err
		}
		v.State = "importing"
	}
	return tx.downloadUpdateState(ctx, d, done)
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
	if dl, err := tx.q.DownloadGet(ctx, infoHash); err != nil && err != sql.ErrNoRows {
		return nil, err
	} else if err == nil {
		return tx.newDownload(ctx, dl)
	}
	dl, err := tx.q.DownloadCreate(ctx, schema.DownloadCreateParams{
		InfoHash: infoHash,
		State:    "queued",
		Title:    info.Name,
		Torrent:  b,
	})
	if err != nil {
		return nil, err
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
		_, err = tx.updateDownload(ctx, &ts[0])
		return err
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
	_, info, err := ParseTorrent(dl.Torrent)
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
		vid, err := tx.q.VideoCreate(ctx, schema.VideoCreateParams{
			InfoHash: &dl.InfoHash,
			Name:     p,
		})
		if err != nil {
			return nil, err
		}
		_, err = tx.q.EpisodeVideoCreate(ctx, schema.EpisodeVideoCreateParams{
			EpisodeID: ep.ep.ID,
			VideoID:   vid.ID,
		})
		if err != nil {
			return nil, err
		}
	}
	dl, err = tx.q.DownloadUpdatePlanSeries(ctx, schema.DownloadUpdatePlanSeriesParams{
		InfoHash:            dl.InfoHash,
		PlanSeriesEditionID: &sedID,
	})
	if err != nil {
		return nil, err
	}
	return tx.newDownload(ctx, dl)
}

func (tx *TxRW) DownloadCreatePlanMovie(ctx Context, infoHash, medID string) (d *Download, err error) {
	defer errorfmt.Handlef("DownloadCreatePlanMovie(%s, %s): %w", infoHash, medID, &err)
	dl, info, err := tx.downloadForPlanning(ctx, infoHash)
	if err != nil {
		return nil, err
	}
	createVideo := func(path string) error {
		if !hasVideoExtension(path) {
			return nil
		}
		vid, err := tx.q.VideoCreate(ctx, schema.VideoCreateParams{
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
		if err := createVideo(info.Name); err != nil {
			return nil, err
		}
	} else {
		for _, fi := range info.Files {
			if err := createVideo(fiPath(&fi)); err != nil {
				return nil, err
			}
		}
	}
	dl, err = tx.q.DownloadUpdatePlanMovie(ctx, schema.DownloadUpdatePlanMovieParams{
		InfoHash:           dl.InfoHash,
		PlanMovieEditionID: &medID,
	})
	if err != nil {
		return nil, err
	}
	return tx.newDownload(ctx, dl)
}

func (tx *TxRW) updateDownload(ctx Context, t *transmissionrpc.Torrent) (schema.Download, error) {
	slog.InfoContext(ctx, "update-download", "hash", *t.HashString)

	d, err := tx.Download(ctx, *t.HashString)
	if err != nil {
		return schema.Download{}, err
	}
	ctx = logcontext.With(ctx, "dl", d.InfoHash())

	if t.ErrorString != nil && *t.ErrorString != "" {
		return tx.q.DownloadUpdateError(ctx, schema.DownloadUpdateErrorParams{
			InfoHash: d.InfoHash(),
			Error:    *t.ErrorString,
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
				return schema.Download{}, err
			}
			if epID, ok := d.epIDByVideoID[v.ID]; ok {
				tx.m.prog.AddEdge(epID, v.ID)
			}
			diskPath, err := tx.transmissionDiskPath(ctx, t, v.Name)
			if err != nil {
				return schema.Download{}, err
			}
			err = tx.addTask(ctx, taskIngest, v.ID, diskPath)
			if err != nil {
				return schema.Download{}, err
			}
			v.State = "importing"
		}
	}

	doneness := 0.0
	if t.PercentDone != nil {
		doneness = *t.PercentDone
	}
	state := d.deriveState(done)
	_, err = tx.q.DownloadUpdateProgress(ctx, schema.DownloadUpdateProgressParams{
		InfoHash: d.InfoHash(),
		State:    state,
		Progress: doneness,
	})
	if err != nil {
		return schema.Download{}, err
	}
	tx.onCommit(func() {
		tx.m.setInfoHashActive(*t.HashString, state == "downloading")
	})
	return d.d, nil
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

func (tx *TxRW) downloadUpdateState(ctx Context, d *Download, done map[string]bool) error {
	state := d.deriveState(done)
	tx.onCommit(func() {
		tx.m.setInfoHashActive(d.d.InfoHash, state == "downloading")
	})
	_, err := tx.q.DownloadUpdateProgress(ctx, schema.DownloadUpdateProgressParams{
		InfoHash: d.InfoHash(),
		State:    state,
		Progress: d.Progress(),
	})
	return err
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
			dl, err := tx.Download(ctx, *ts[i].HashString)
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
		key := "dlf-" + dl.InfoHash() + "-" + hex.EncodeToString(h[:8])
		if tf.Length > 0 {
			frac := float64(tf.BytesCompleted) / float64(tf.Length)
			m.prog.AddEdge("dl-"+dl.InfoHash(), key)
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
