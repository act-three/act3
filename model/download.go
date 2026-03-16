package model

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json/v2"
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
	d schema.Download

	planLenOnce sync.Once
	planLen     int

	filesLenOnce sync.Once
	filesLen     int
}

func newDownloadHeadList(dls []schema.Download, err error) ([]*DownloadHead, error) {
	if err != nil {
		return nil, err
	}
	res := make([]*DownloadHead, len(dls))
	for i := range dls {
		res[i] = &DownloadHead{d: dls[i]}
	}
	return res, nil
}

func (d *DownloadHead) URL() string   { return "/app/downloads/" + d.d.ID }
func (d *DownloadHead) ID() string    { return d.d.ID }
func (d *DownloadHead) State() string { return d.d.State }
func (d *DownloadHead) Title() string { return d.d.Title }
func (d *DownloadHead) Error() string { return d.d.Error }

func (d *DownloadHead) PlanLen() int {
	d.planLenOnce.Do(func() {
		var plan map[string]span
		err := json.Unmarshal([]byte(d.d.Plan), &plan)
		if err != nil {
			panic(err)
		}
		d.planLen = len(plan)
	})
	return d.planLen
}

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
	plan span
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

func (df *DownloadFile) Episode() *Episode {
	// TODO(april): return actual episode (not planned episode) after import
	sed := df.SeriesEdition()
	if sed == nil {
		return nil
	}
	return sed.episodeByID(df.plan.ID)
}

func (df *DownloadFile) Season() *Season {
	// TODO(april): return actual season (not planned season) after import
	sed := df.SeriesEdition()
	if sed == nil {
		return nil
	}
	return sed.seasonByEpisodeID(df.plan.ID)
}

func (df *DownloadFile) SeriesEdition() *SeriesEdition {
	// TODO(april): return actual series (not planned series) after import
	return df.d.PlanSeriesEdition()
}

func (df *DownloadFile) MovieEdition() *MovieEdition {
	return df.d.PlanMovieEdition()
}

type Download struct {
	DownloadHead
	metaInfo     *metainfo.MetaInfo
	info         metainfo.Info
	plan         map[string]span
	planEd       *SeriesEdition
	planMovieEd  *MovieEdition
	fileProgress map[string]float64 // path -> fraction [0,1], nil if unknown
}

type addr struct{ Snn, Enn int }

type span struct {
	ID string // first episode ID
	N  int    // num episodes
}

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

	err = json.Unmarshal([]byte(d.d.Plan), &d.plan)
	if err != nil {
		return nil, err
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

func (d *Download) PlanFor(path string) (epID string, span int) {
	p := d.plan[path]
	return p.ID, p.N
}

func (d *Download) PlanEpisode(path string) *Episode {
	if d.planEd == nil {
		return nil
	}
	id, _ := d.PlanFor(path)
	return d.planEd.episodeByID(id)
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
			plan: d.plan[d.info.Name],
			path: d.info.Name,
		}}
	}
	dfs := make([]*DownloadFile, len(d.info.Files))
	for i, fi := range d.info.Files {
		dfs[i] = &DownloadFile{
			d:    d,
			fi:   fi,
			plan: d.plan[fiPath(&fi)],
		}
	}
	return dfs
}

func (tx *TxR) DownloadHeadList(ctx Context) ([]*DownloadHead, error) {
	return newDownloadHeadList(tx.q.DownloadList(ctx))
}

func (tx *TxR) DownloadHeadListBySeriesEditionID(ctx Context, id string) ([]*DownloadHead, error) {
	return newDownloadHeadList(tx.q.DownloadListByPlanSeriesEditionID(ctx, &id))
}

func (tx *TxR) DownloadHeadListByMovieEditionID(ctx Context, id string) ([]*DownloadHead, error) {
	return newDownloadHeadList(tx.q.DownloadListByPlanMovieEditionID(ctx, &id))
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
		State:    "added",
		Title:    info.Name,
		Torrent:  b,
		InfoHash: infoHash,
		Plan:     "{}",
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
	case "done", "error":
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
	plan, err := tx.planSeries(ctx, info, sedID)
	if err != nil {
		return nil, err
	}
	dl, err = tx.q.DownloadUpdatePlanSeries(ctx, schema.DownloadUpdatePlanSeriesParams{
		ID:                  dl.ID,
		PlanSeriesEditionID: &sedID,
		Plan:                string(plan),
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
	plan := planMovie(info, medID)
	b, err := json.Marshal(plan)
	if err != nil {
		return nil, err
	}
	dl, err = tx.q.DownloadUpdatePlanMovie(ctx, schema.DownloadUpdatePlanMovieParams{
		ID:                 dl.ID,
		PlanMovieEditionID: &medID,
		Plan:               string(b),
	})
	if err != nil {
		return nil, err
	}
	return tx.newDownload(ctx, dl)
}

// planMovie assigns all video files in the torrent to the
// given movie edition.
// The span ID is the movie edition ID itself;
// N=1 for each file.
func planMovie(info *metainfo.Info, medID string) map[string]span {
	plan := map[string]span{}
	if !info.IsDir() {
		// Single-file torrent: the path is info.Name.
		if hasVideoExtension(info.Name) {
			plan[info.Name] = span{medID, 1}
		}
		return plan
	}
	for _, fi := range info.Files {
		p := fiPath(&fi)
		if !hasVideoExtension(p) {
			continue
		}
		plan[p] = span{medID, 1}
	}
	return plan
}

func (tx *TxR) planSeries(ctx Context, info *metainfo.Info, sedID string) (b []byte, err error) {
	defer errorfmt.Handlef("planSeries(%s): %w", sedID, &err)
	sed, err := tx.SeriesEdition(ctx, sedID)
	if err != nil {
		return nil, err
	}
	plan := map[string]span{}
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
		plan[p] = span{ep.ep.ID, n}
	}
	return json.Marshal(plan)
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

	for path := range d.Paths() {
		planID, _ := d.PlanFor(path)
		if planID == "" || !done[path] {
			continue
		}
		err = tx.importDownloadPath(ctx, d, t, path)
		if err != nil {
			return schema.Download{}, err
		}
		delete(d.plan, path)
	}

	b, err := json.Marshal(d.plan)
	if err != nil {
		return schema.Download{}, err
	}
	state := "active"
	if len(d.plan) == 0 {
		state = "done"
	}
	tx.onCommit(func() {
		tx.m.setInfoHashActive(*t.HashString, state == "active")
	})
	doneness := 0.0
	if t.PercentDone != nil {
		doneness = *t.PercentDone
	}
	return tx.q.DownloadUpdateProgress(ctx, schema.DownloadUpdateProgressParams{
		ID:       d.ID(),
		State:    state,
		Progress: doneness,
		Plan:     string(b),
	})
}

func (tx *TxRW) importDownloadPath(ctx Context, d *Download, t *transmissionrpc.Torrent, path string) error {
	if med := d.PlanMovieEdition(); med != nil {
		return tx.importMovieVideo(ctx, t, med.ID(), path)
	}
	startEpID, n := d.PlanFor(path)
	sed := d.PlanSeriesEdition()
	if sed == nil {
		return nil
	}
	for ep := range sed.episodesBySpan(startEpID, n) {
		err := tx.importEpisode(ctx, t, ep.ep.ID, path)
		if err != nil {
			return err
		}
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
		epID, _ := dl.PlanFor(p)
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
