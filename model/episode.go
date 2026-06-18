package model

import (
	"database/sql"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/model/progress"
	"ily.dev/act3/xstrings"
	"kr.dev/errorfmt"
)

type EpisodeType uint

const (
	Regular EpisodeType = 1 << iota
	SignificantSpecial
	InsignificantSpecial

	AnySpecial  = SignificantSpecial | InsignificantSpecial
	AnyEpisode  = Regular | AnySpecial
	Significant = Regular | SignificantSpecial
)

type EpisodeState = int

const (
	EpIsEmpty EpisodeState = iota
	EpIsDownloading
	EpIsPlayable
)

var episodeTypeByName = map[string]EpisodeType{
	"regular":               Regular,
	"significant_special":   SignificantSpecial,
	"insignificant_special": InsignificantSpecial,
}

type seasonEpisode struct {
	ep   schema.Episode
	snep schema.SeasonEpisode
}

type EpisodeHead struct {
	ep schema.Episode
}

func (ep *EpisodeHead) ID() string { return ep.ep.ID }
func (ep *EpisodeHead) Thumbnail() Image {
	return Image{ID: ep.ep.ThumbnailID, Kind: ImageThumbnail}
}

func (ep *EpisodeHead) addr(field string) []string {
	return []string{"episode", ep.ep.ID, field}
}

func (ep *EpisodeHead) ThumbnailAddr() []string { return ep.addr("thumbnail") }
func (ep *EpisodeHead) ThumbnailField() (Image, []string) {
	return ep.Thumbnail(), ep.ThumbnailAddr()
}

type Episode struct {
	EpisodeHead
	snep   schema.SeasonEpisode
	type_  EpisodeType
	sn     *SeasonHead
	so     *SeriesEditionHead
	sr     *SeriesHead
	prog   []*progress.Item
	videos []*Video
}

func newEpisode(
	sr *SeriesHead,
	so *SeriesEditionHead,
	sn *SeasonHead,
	snepData schema.SeasonEpisode,
	epData schema.Episode,
	prog []*progress.Item,
	videos []*Video,
) *Episode {
	return &Episode{
		EpisodeHead: EpisodeHead{ep: epData},
		snep:        snepData,
		type_:       episodeTypeByName[epData.Type],
		sn:          sn,
		so:          so,
		sr:          sr,
		prog:        prog,
		videos:      videos,
	}
}

func (ep *Episode) Slug() string               { return ep.snep.Slug }
func (ep *Episode) Title() string              { return ep.ep.Title }
func (ep *Episode) Airdate() string            { return ep.ep.Airdate }
func (ep *Episode) Summary() string            { return ep.ep.Summary }
func (ep *Episode) Type() string               { return ep.ep.Type }
func (ep *Episode) Progress() []*progress.Item { return ep.prog }
func (ep *Episode) Videos() []*Video           { return ep.videos }

// ActiveVideo returns the video marked Active for this episode, or
// nil if none is set. Callers in theater contexts should use this
// rather than picking from Videos().
func (ep *Episode) ActiveVideo() *Video {
	for _, v := range ep.videos {
		if v.active {
			return v
		}
	}
	return nil
}
func (ep *Episode) SnnEnn() string {
	if eNN := ep.snep.Number; eNN != 0 {
		return fmt.Sprintf("S%dE%d", ep.sn.sn.Number, eNN)
	}
	return fmt.Sprintf("S%d Special", ep.sn.sn.Number)
}

func (ep *Episode) CoarseType() string {
	if ep.ep.Type == "regular" {
		return "regular"
	}
	return "special"
}

func (ep *Episode) State() EpisodeState {
	if ep.ActiveVideo() != nil {
		return EpIsPlayable
	}
	if len(ep.Videos()) > 0 {
		return EpIsDownloading
	}
	return EpIsEmpty
}

func (ep *Episode) AirdateAddr() []string { return ep.addr("airdate") }
func (ep *Episode) SummaryAddr() []string { return ep.addr("summary") }
func (ep *Episode) TitleAddr() []string   { return ep.addr("title") }
func (ep *Episode) TypeAddr() []string    { return ep.addr("type") }

func (ep *Episode) TitleField() (string, []string) { return ep.Title(), ep.TitleAddr() }
func (ep *Episode) TypeField() (string, []string)  { return ep.Type(), ep.TypeAddr() }

func (ep *Episode) Info() []string {
	return []string{ep.SnnEnn(), ep.sr.Title()}
}

func (ep *Episode) ImageField() (Image, []string) { return ep.ThumbnailField() }
func (ep *Episode) ImageAspect() (n, d int)       { return ImageThumbnail.Aspect() }
func (ep *Episode) ReleaseDate() string           { return ep.Airdate() }

func (ep *Episode) Runtime() string {
	if ep.ep.Runtime > 0 {
		return fmt.Sprintf("%d", ep.ep.Runtime)
	}
	return ""
}

func (ep *Episode) basename() string {
	var p []string
	p = append(p, ep.sr.Title())
	if y, _, _ := strings.Cut(ep.sr.PremieredOn(), "-"); y != "" {
		p = append(p, "("+y+")")
	}
	if ep.so.Slug() != "" {
		p = append(p, ep.so.Label())
	}
	if n := ep.snep.Number; n != 0 {
		p = append(p, fmt.Sprintf("S%02dE%02d", ep.sn.sn.Number, n))
	} else {
		p = append(p, fmt.Sprintf("S%02d Special", ep.sn.sn.Number))
	}
	p = append(p, ep.Title())
	return xstrings.SanitizeFilename(strings.Join(p, " "))
}

func (ep *Episode) PlayIDs() PlayIDs {
	v := ep.ActiveVideo()
	if v == nil {
		return PlayIDs{}
	}
	return PlayIDs{
		VideoID:         v.ID(),
		EpisodeID:       ep.ID(),
		SeriesEditionID: ep.so.ID(),
	}
}

func (ep *Episode) HasType(types EpisodeType) bool {
	return types&ep.type_ != 0
}

func (ep *Episode) Label() string {
	l := ep.snep.Label
	if l == "Special" {
		return "Special: " + ep.ep.Title
	}
	return ep.snep.Label + ". " + ep.ep.Title
}

func (ep *Episode) EditorPath() string {
	return path.Join(
		"/app/series",
		ep.SeriesHead().Slug(),
		ep.SeriesEditionHead().Slug(),
		ep.Slug(),
	)
}

func (ep *Episode) SeriesHead() *SeriesHead {
	return ep.sr
}

func (ep *Episode) SeriesEditionHead() *SeriesEditionHead {
	return ep.so
}

func (ep *Episode) SeasonHead() *SeasonHead {
	return ep.sn
}

func (ep *Episode) EditionTheaterPath() string {
	if ep.so.Slug() == "" {
		return ep.sr.TheaterPath()
	}
	return path.Join(ep.sr.TheaterPath(), ep.so.Slug())
}

func (ep *Episode) TheaterPath() string {
	if ep.so.Slug() == "" {
		return path.Join("/", ep.sr.Slug(), ep.Slug())
	}
	return path.Join("/", ep.sr.Slug(), ep.so.Slug(), ep.Slug())
}

func (tx *TxR) EpisodeHead(ctx Context, id string) (*EpisodeHead, error) {
	ep, err := tx.q.EpisodeGet(id)
	if err != nil {
		return nil, err
	}
	return &EpisodeHead{ep: ep}, nil
}

// EpisodeBySlug looks up an episode by its slug components.
// edSlug selects the edition; empty string selects the default.
func (tx *TxR) EpisodeBySlug(ctx Context, seriesSlug, edSlug, epSlug string) (*Episode, error) {
	sed, err := tx.q.SeriesEditionGetBySlug(schema.SeriesEditionGetBySlugParams{
		SeriesSlug:  seriesSlug,
		EditionSlug: edSlug,
	})

	if err != nil {
		return nil, err
	}
	snep, err := tx.q.SeasonEpisodeGetBySlug(schema.SeasonEpisodeGetBySlugParams{
		EditionID: sed.ID,
		Slug:      epSlug,
	})

	if err != nil {
		return nil, err
	}
	return tx.EpisodeInEdition(ctx, snep.EpisodeID, sed.ID)
}

func (tx *TxR) EpisodeInEdition(ctx Context, id, edID string) (*Episode, error) {
	epRec, err := tx.q.EpisodeGet(id)
	if err != nil {
		return nil, err
	}
	sneps, err := tx.q.SeasonEpisodeListByEpisodeID(id)
	if err != nil {
		return nil, err
	}
	vids, err := tx.q.VideoListByEpisodeID(id)
	if err != nil {
		return nil, err
	}
	evs, err := tx.q.EpisodeVideoListByEpisodeID(id)
	if err != nil {
		return nil, err
	}
	activeByVID := map[string]bool{}
	for _, ev := range evs {
		if ev.Active != 0 {
			activeByVID[ev.VideoID] = true
		}
	}
	var videos []*Video
	for i := range vids {
		videos = append(videos, &Video{v: vids[i], active: activeByVID[vids[i].ID]})
	}
	ep := &Episode{
		EpisodeHead: EpisodeHead{ep: epRec},
		videos:      videos,
	}
	for i, snep := range sneps {
		sn, err := tx.q.SeasonGet(snep.SeasonID)
		if err != nil {
			return nil, err
		}
		seq, err := tx.q.SeriesEditionGet(sn.EditionID)
		if err != nil {
			return nil, err
		}
		if seq.ID != edID && i < len(sneps)-1 {
			continue
		}
		sr, err := tx.q.SeriesGet(seq.SeriesID)
		if err != nil {
			return nil, err
		}
		ep.snep = snep
		ep.sn = &SeasonHead{sn}
		ep.so = &SeriesEditionHead{sed: seq}
		ep.sr = &SeriesHead{sr}
		return ep, nil
	}
	return nil, fmt.Errorf("cannot load ep")
}

// EpisodeEditions returns the episode as it appears in each edition.
func (tx *TxR) EpisodeEditions(ctx Context, episodeID string) ([]*Episode, error) {
	epRec, err := tx.q.EpisodeGet(episodeID)
	if err != nil {
		return nil, err
	}
	sneps, err := tx.q.SeasonEpisodeListByEpisodeID(episodeID)
	if err != nil {
		return nil, err
	}
	var eps []*Episode
	for _, snep := range sneps {
		sn, err := tx.q.SeasonGet(snep.SeasonID)
		if err != nil {
			return nil, err
		}
		sed, err := tx.q.SeriesEditionGet(sn.EditionID)
		if err != nil {
			return nil, err
		}
		sr, err := tx.q.SeriesGet(sed.SeriesID)
		if err != nil {
			return nil, err
		}
		eps = append(eps, &Episode{
			EpisodeHead: EpisodeHead{ep: epRec},
			snep:        snep,
			type_:       episodeTypeByName[epRec.Type],
			sn:          &SeasonHead{sn},
			so:          &SeriesEditionHead{sed: sed},
			sr:          &SeriesHead{sr},
		})
	}
	return eps, nil
}

func (tx *TxRW) EpisodeThumbnailIDSet(ctx Context, id, thumbnailID string) error {
	ep, err := tx.q.EpisodeGet(id)
	if err != nil {
		return err
	}
	err = tx.q.EpisodeThumbnailIDSet(schema.EpisodeThumbnailIDSetParams{
		ThumbnailID: thumbnailID,
		ID:          id,
	})

	if err != nil {
		return err
	}
	if isPlaceholderImageID(ep.ThumbnailID) {
		return nil
	}
	return tx.imageDelete(ctx, ep.ThumbnailID)
}

func (tx *TxRW) EpisodeSummarySet(ctx Context, id, summary string) error {
	if _, err := tx.q.EpisodeGet(id); err != nil {
		return err
	}
	return tx.q.EpisodeSummarySet(schema.EpisodeSummarySetParams{
		Summary: summary,
		ID:      id,
	})

}

func (tx *TxRW) EpisodeAirdateSet(ctx Context, id, airdate string) error {
	if _, err := tx.q.EpisodeGet(id); err != nil {
		return err
	}
	return tx.q.EpisodeAirdateSet(schema.EpisodeAirdateSetParams{
		Airdate: airdate,
		ID:      id,
	})

}

func (tx *TxRW) EpisodeTypeSet(ctx Context, id, typ string) error {
	if _, err := tx.q.EpisodeGet(id); err != nil {
		return err
	}
	err := tx.q.EpisodeTypeSet(schema.EpisodeTypeSetParams{
		Type: typ,
		ID:   id,
	})

	if err != nil {
		return err
	}
	sneps, err := tx.q.SeasonEpisodeListByEpisodeID(id)
	if err != nil {
		return err
	}
	for _, snep := range sneps {
		if err := tx.renumberSeason(ctx, snep.SeasonID); err != nil {
			return err
		}
	}
	return nil
}

func (tx *TxRW) EpisodeTitleSet(ctx Context, id, title string) error {
	if _, err := tx.q.EpisodeGet(id); err != nil {
		return err
	}
	err := tx.q.EpisodeTitleSet(schema.EpisodeTitleSetParams{
		Title: title,
		ID:    id,
	})

	if err != nil {
		return err
	}
	sneps, err := tx.q.SeasonEpisodeListByEpisodeID(id)
	if err != nil {
		return err
	}
	for _, snep := range sneps {
		sn, err := tx.q.SeasonGet(snep.SeasonID)
		if err != nil {
			return err
		}
		slug, err := tx.episodeFindSlug(ctx, snep.EditionID, sn.Number, snep.Number, title, snep.Slug)
		if err != nil {
			return err
		}
		if slug != snep.Slug {
			tx.emitDetail(Detail{SlugChangeID: id})
			err = tx.q.SeasonEpisodeSlugSet(schema.SeasonEpisodeSlugSetParams{
				Slug:      slug,
				SeasonID:  snep.SeasonID,
				EpisodeID: id,
			})

			if err != nil {
				return err
			}
		}
	}
	return nil
}

// EpisodeMove moves an episode to a given position within a
// (possibly different) season and renumbers affected seasons.
func (tx *TxRW) EpisodeMove(ctx Context, episodeID, fromSeasonID, targetSeasonID string, index int) (err error) {
	defer errorfmt.Handlef("episode move: %w", &err)

	src, err := tx.q.SeasonEpisodeGet(schema.SeasonEpisodeGetParams{
		SeasonID:  fromSeasonID,
		EpisodeID: episodeID,
	})

	if err != nil {
		return err
	}

	targetSn, err := tx.q.SeasonGet(targetSeasonID)
	if err != nil {
		return err
	}

	if src.EditionID != targetSn.EditionID {
		return fmt.Errorf("episode %s and season %s are in different editions", episodeID, targetSeasonID)
	}

	isSameSeason := fromSeasonID == targetSeasonID

	// Build the new ordering for the target season and
	// move the episode to its final postion in the target list.
	targetEps, err := tx.q.SeasonEpisodeListBySeasonID(targetSeasonID)
	if err != nil {
		return err
	}
	if isSameSeason {
		targetEps = slices.DeleteFunc(targetEps, func(s schema.SeasonEpisode) bool {
			return s.EpisodeID == episodeID
		})
	}
	index = max(0, min(index, len(targetEps)))
	targetEps = slices.Insert(targetEps, index, src)

	if !isSameSeason {
		// Delete the moved episode from the source season to free
		// the UNIQUE(EditionID, Slug) before inserting into the target.
		if err := tx.q.SeasonEpisodeDelete(schema.SeasonEpisodeDeleteParams{
			SeasonID:  src.SeasonID,
			EpisodeID: episodeID,
		}); err != nil {
			return err
		}
		if err := tx.renumberSeason(ctx, src.SeasonID); err != nil {
			return err
		}
	}

	// Delete and re-insert the target season in the new order.
	if err := tx.q.SeasonEpisodeDeleteBySeasonID(targetSeasonID); err != nil {
		return err
	}
	for i, snep := range targetEps {
		err = tx.q.SeasonEpisodeCreate(schema.SeasonEpisodeCreateParams{
			EditionID: targetSn.EditionID,
			SeasonID:  targetSeasonID,
			EpisodeID: snep.EpisodeID,
			SortKey:   int64(i),
			Label:     snep.Label,
			Number:    snep.Number,
			Slug:      snep.Slug,
		})

		if err != nil {
			return err
		}
	}
	return tx.renumberSeason(ctx, targetSeasonID)
}

// SeasonEpisodeCreate creates a new regular episode at the end of the
// given season with reasonable defaults.
func (tx *TxRW) SeasonEpisodeCreate(ctx Context, seasonID string) error {
	sn, err := tx.q.SeasonGet(seasonID)
	if err != nil {
		return err
	}

	existing, err := tx.q.SeasonEpisodeListBySeasonID(sn.ID)
	if err != nil {
		return err
	}

	// Count existing regular episodes to determine the next number.
	var maxNum, maxSortKey int64
	for _, snep := range existing {
		maxNum = max(maxNum, snep.Number)
		maxSortKey = max(maxSortKey, snep.SortKey)
	}
	num := maxNum + 1
	sortKey := maxSortKey + 1
	title := "New Episode"
	label := strconv.FormatInt(num, 10)

	ep, err := tx.q.EpisodeCreate(schema.EpisodeCreateParams{
		Title:   title,
		Summary: "The main character encounters an unexpected challenge!",
		Type:    "regular",
	})

	if err != nil {
		return err
	}

	slug, err := tx.episodeFindSlug(ctx, sn.EditionID, sn.Number, num, title)
	if err != nil {
		return err
	}

	return tx.q.SeasonEpisodeCreate(schema.SeasonEpisodeCreateParams{
		EditionID: sn.EditionID,
		SeasonID:  sn.ID,
		EpisodeID: ep.ID,
		SortKey:   sortKey,
		Label:     label,
		Number:    num,
		Slug:      slug,
	})

}

// SeasonEpisodeRemove removes an episode from a season,
// leaving the Episode row intact so other editions are unaffected.
// The emitted event carries the episode's prior SortKey so a later
// SeasonEpisodeAdd can restore it to the same position.
func (tx *TxRW) SeasonEpisodeRemove(ctx Context, seasonID, episodeID string) (err error) {
	defer errorfmt.Handlef("season episode remove: %w", &err)

	if _, err := tx.q.SeasonEpisodeGet(schema.SeasonEpisodeGetParams{
		SeasonID:  seasonID,
		EpisodeID: episodeID,
	}); err != nil {
		return err
	}

	if err := tx.q.SeasonEpisodeDelete(schema.SeasonEpisodeDeleteParams{
		SeasonID:  seasonID,
		EpisodeID: episodeID,
	}); err != nil {
		return err
	}
	return tx.renumberSeason(ctx, seasonID)
}

// SeasonEpisodeAdd links an existing Episode into a season at the
// given SortKey, bumping existing episodes at or after that key by 1.
// Numbering is assigned by renumberSeason.
func (tx *TxRW) SeasonEpisodeAdd(ctx Context, seasonID, episodeID string, sortKey int64) (err error) {
	defer errorfmt.Handlef("season episode add: %w", &err)

	sn, err := tx.q.SeasonGet(seasonID)
	if err != nil {
		return err
	}
	ep, err := tx.q.EpisodeGet(episodeID)
	if err != nil {
		return err
	}

	// Placeholder slug; renumberSeason assigns final Number/Label/Slug.
	slug, err := tx.episodeFindSlug(ctx, sn.EditionID, sn.Number, sortKey+1, ep.Title)
	if err != nil {
		return err
	}

	if err := tx.seasonEpisodeSortKeyBump(ctx, sn.ID, sortKey); err != nil {
		return err
	}
	if err := tx.q.SeasonEpisodeCreate(schema.SeasonEpisodeCreateParams{
		EditionID: sn.EditionID,
		SeasonID:  sn.ID,
		EpisodeID: ep.ID,
		SortKey:   sortKey,
		Slug:      slug,
	}); err != nil {
		return err
	}
	return tx.renumberSeason(ctx, sn.ID)
}

// episodeFindSlug builds an episode slug from the season number,
// episode number (0 for specials), and title.
// Slugs in allow are accepted without a uniqueness check;
// otherwise deduplication appends "-2", "-3", etc.
func (tx *TxRW) episodeFindSlug(ctx Context, editionID string, seasonNum, episodeNum int64, title string, allow ...string) (string, error) {
	var base string
	if episodeNum == 0 {
		base = fmt.Sprintf("s%d-special", seasonNum)
	} else {
		base = fmt.Sprintf("s%de%d", seasonNum, episodeNum)
	}
	slug := base
	if titleSlug := xstrings.ToSlug(title); titleSlug != "" {
		slug += "-" + titleSlug
	}
	if slices.Contains(allow, slug) {
		return slug, nil
	}
	for i := 1; ; i++ {
		candidate := slug
		if i >= 2 {
			candidate += "-" + strconv.Itoa(i)
		}
		n, err := tx.q.SeasonEpisodeSlugExists(schema.SeasonEpisodeSlugExistsParams{
			EditionID: editionID,
			Slug:      candidate,
		})

		if err != nil {
			return "", err
		}
		if n == 0 {
			return candidate, nil
		}
	}
}

func (tx *TxR) EpisodeDownloadList(ctx Context, ep *Episode) ([]*RenditionForDownload, error) {
	active := ep.ActiveVideo()
	if active == nil {
		return nil, nil
	}
	sedID := ep.SeriesEditionHead().ID()

	var rends []*RenditionForDownload
	rfd, err := tx.q.RenditionGetDownloadByVideoID(active.ID())
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil && rfd.Key != "" {
		rends = append(rends, &RenditionForDownload{
			path:  path.Join("/-/dl", rfd.ID, ep.ID(), sedID),
			label: "Best Quality MP4 (Recommended)",
		})
	}

	ext := videoExtensionForContentType(active.v.OriginalType)
	rends = append(rends, &RenditionForDownload{
		path:  path.Join("/-/dl", active.ID(), ep.ID(), sedID),
		label: "Original " + strings.ToUpper(ext),
	})
	return rends, nil
}

func epMapByID(eps []schema.Episode) map[string]*schema.Episode {
	epByID := map[string]*schema.Episode{}
	for i := range eps {
		epByID[eps[i].ID] = &eps[i]
	}
	return epByID
}

func snepMapBySeasonID(sneps []schema.SeasonEpisode) map[string][]schema.SeasonEpisode {
	snepBySeasonID := map[string][]schema.SeasonEpisode{}
	for i := range sneps {
		snID := sneps[i].SeasonID
		snepBySeasonID[snID] = append(snepBySeasonID[snID], sneps[i])
	}
	return snepBySeasonID
}
