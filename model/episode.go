package model

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/model/progress"
	"ily.dev/act3/xstrings"
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
	Empty EpisodeState = iota
	Downloading
	Playable
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

type Episode struct {
	ep     schema.Episode
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
	ep := &Episode{
		ep:     epData,
		snep:   snepData,
		type_:  episodeTypeByName[epData.Type],
		sn:     sn,
		so:     so,
		sr:     sr,
		prog:   prog,
		videos: videos,
	}
	return ep
}

func (ep *Episode) ID() string                 { return ep.ep.ID }
func (ep *Episode) Slug() string               { return ep.snep.Slug }
func (ep *Episode) Title() string              { return ep.ep.Title }
func (ep *Episode) Airdate() string            { return ep.ep.Airdate }
func (ep *Episode) Summary() string            { return ep.ep.Summary }
func (ep *Episode) ImageURL() string           { return ep.ep.TVmazeImageURL }
func (ep *Episode) Type() string               { return ep.ep.Type }
func (ep *Episode) Progress() []*progress.Item { return ep.prog }
func (ep *Episode) Videos() []*Video           { return ep.videos }
func (ep *Episode) SnnEnn() string {
	if eNN := ep.snep.Number; eNN != 0 {
		return fmt.Sprintf("S%02dE%02d", ep.sn.sn.Number, eNN)
	}
	return fmt.Sprintf("S%02d Special", ep.sn.sn.Number)
}

func (ep *Episode) State() EpisodeState {
	for _, v := range ep.Videos() {
		if v.MVPlaylist() == "" {
			return Downloading
		}
	}
	if len(ep.Videos()) > 0 {
		return Playable
	}
	return Empty
}

func (ep *Episode) AirdateAddr() []string { return ep.addr("airdate") }
func (ep *Episode) SummaryAddr() []string { return ep.addr("summary") }
func (ep *Episode) TitleAddr() []string   { return ep.addr("title") }
func (ep *Episode) TypeAddr() []string    { return ep.addr("type") }

func (ep *Episode) TitleField() (string, []string) { return ep.Title(), ep.TitleAddr() }
func (ep *Episode) TypeField() (string, []string)  { return ep.Type(), ep.TypeAddr() }

func (ep *Episode) addr(field string) []string {
	return []string{"episode", ep.ep.ID, field}
}

func (ep *Episode) PlayerPath(v *Video) string {
	return fmt.Sprintf("/-/player/%s/%s/%s", v.ID(), ep.ID(), ep.so.ID())
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

func (ep *Episode) EditDialogPath() string {
	return "/-/dialog/episode-edit/" + ep.ep.ID
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

func (ep *Episode) TheaterPath() string {
	if ep.so.Slug() == "" {
		return path.Join("/", ep.sr.Slug(), ep.Slug())
	}
	return path.Join("/", ep.sr.Slug(), ep.so.Slug(), ep.Slug())
}

// EpisodeBySlug looks up an episode by its slug components.
// edSlug selects the edition; empty string selects the default.
func (tx *TxR) EpisodeBySlug(ctx Context, seriesSlug, edSlug, epSlug string) (*Episode, error) {
	sed, err := tx.q.SeriesEditionGetBySlug(ctx, schema.SeriesEditionGetBySlugParams{
		SeriesSlug:  seriesSlug,
		EditionSlug: edSlug,
	})
	if err != nil {
		return nil, err
	}
	snep, err := tx.q.SeasonEpisodeGetBySlug(ctx, schema.SeasonEpisodeGetBySlugParams{
		EditionID: sed.ID,
		Slug:      epSlug,
	})
	if err != nil {
		return nil, err
	}
	return tx.episodeInContext(ctx, snep.EpisodeID, sed.ID, "", "")
}

// Episode is like EpisodeInEdition, but it assumes the Air Date edition.
func (tx *TxR) Episode(ctx Context, id string) (*Episode, error) {
	return tx.episodeInContext(ctx, id, "", AirDate, "")
}

func (tx *TxR) EpisodeInEdition(ctx Context, id, edID string) (*Episode, error) {
	return tx.episodeInContext(ctx, id, edID, "", "")
}

func (tx *TxR) episodeInContext(ctx Context, id, edID, edName, edSlug string) (*Episode, error) {
	epRec, err := tx.q.EpisodeGet(ctx, id)
	if err != nil {
		return nil, err
	}
	sneps, err := tx.q.SeasonEpisodeListByEpisodeID(ctx, id)
	if err != nil {
		return nil, err
	}
	vids, err := tx.q.VideoListByEpisodeID(ctx, id)
	if err != nil {
		return nil, err
	}
	var videos []*Video
	for i := range vids {
		v := &Video{v: vids[i]}
		ats, err := tx.q.AudioTrackListByVideoID(ctx, vids[i].ID)
		if err != nil {
			return nil, err
		}
		for j := range ats {
			v.audioTracks = append(v.audioTracks, &AudioTrack{at: ats[j]})
		}
		videos = append(videos, v)
	}
	ep := &Episode{
		ep:     epRec,
		videos: videos,
	}
	for i, snep := range sneps {
		sn, err := tx.q.SeasonGet(ctx, snep.SeasonID)
		if err != nil {
			return nil, err
		}
		seq, err := tx.q.SeriesEditionGet(ctx, sn.EditionID)
		if err != nil {
			return nil, err
		}
		if seq.ID != edID && seq.Label != edName && seq.Slug != edSlug && i < len(sneps)-1 {
			continue
		}
		sr, err := tx.q.SeriesGet(ctx, seq.SeriesID)
		if err != nil {
			return nil, err
		}
		ep.snep = snep
		ep.sn = &SeasonHead{sn}
		ep.so = &SeriesEditionHead{seq}
		ep.sr = &SeriesHead{sr}
		return ep, nil
	}
	return nil, fmt.Errorf("cannot load ep")
}

func (tx *TxRW) EpisodeSummarySet(ctx Context, id, summary string) error {
	ep, err := tx.q.EpisodeGet(ctx, id)
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventLiveUpdate,
			Addr:    (&Episode{ep: ep}).SummaryAddr(),
			NewText: summary,
			OldText: ep.Summary,
		})
	})
	return tx.q.EpisodeSummarySet(ctx, schema.EpisodeSummarySetParams{
		Summary: summary,
		ID:      id,
	})
}

func (tx *TxRW) EpisodeAirdateSet(ctx Context, id, airdate string) error {
	ep, err := tx.q.EpisodeGet(ctx, id)
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventLiveUpdate,
			Addr:    (&Episode{ep: ep}).AirdateAddr(),
			NewText: airdate,
			OldText: ep.Airdate,
		})
	})
	return tx.q.EpisodeAirdateSet(ctx, schema.EpisodeAirdateSetParams{
		Airdate: airdate,
		ID:      id,
	})
}

func (tx *TxRW) EpisodeTypeSet(ctx Context, id, typ string) error {
	ep, err := tx.q.EpisodeGet(ctx, id)
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventLiveUpdate,
			Addr:    (&Episode{ep: ep}).TypeAddr(),
			NewText: typ,
			OldText: ep.Type,
		})
	})
	err = tx.q.EpisodeTypeSet(ctx, schema.EpisodeTypeSetParams{
		Type: typ,
		ID:   id,
	})
	if err != nil {
		return err
	}
	sneps, err := tx.q.SeasonEpisodeListByEpisodeID(ctx, id)
	if err != nil {
		return err
	}
	for _, snep := range sneps {
		sn, err := tx.q.SeasonGet(ctx, snep.SeasonID)
		if err != nil {
			return err
		}
		if err := tx.renumberSeason(ctx, sn); err != nil {
			return err
		}
	}
	return nil
}

// renumberSeason reassigns Number, Label, and Slug for every episode
// in a season.  Specials get Number 0 / Label "Special"; regular
// episodes are numbered sequentially starting from 1 in SortKey order.
func (tx *TxRW) renumberSeason(ctx Context, sn schema.Season) error {
	all, err := tx.q.SeasonEpisodeListBySeasonID(ctx, sn.ID)
	if err != nil {
		return err
	}
	eps := make(map[string]schema.Episode, len(all))
	for _, snep := range all {
		ep, err := tx.q.EpisodeGet(ctx, snep.EpisodeID)
		if err != nil {
			return err
		}
		eps[snep.EpisodeID] = ep
	}

	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type: EventSeasonRenumber,
			ID:   sn.ID,
		})
	})

	var num int64
	for _, snep := range all {
		ep := eps[snep.EpisodeID]
		isSpecial := ep.Type == "significant_special" || ep.Type == "insignificant_special"

		var wantNum int64
		var wantLabel string
		if isSpecial {
			wantLabel = "Special"
		} else {
			num++
			wantNum = num
			wantLabel = strconv.FormatInt(num, 10)
		}

		var base string
		if isSpecial {
			base = fmt.Sprintf("s%02d-special", sn.Number)
		} else {
			base = fmt.Sprintf("s%02de%02d", sn.Number, wantNum)
		}
		wantSlug, err := tx.generateEpisodeSlug(ctx, snep.EditionID, snep.Slug, base, ep.Title, snep.EpisodeID)
		if err != nil {
			return err
		}
		if snep.Number == wantNum && snep.Label == wantLabel && snep.Slug == wantSlug {
			continue
		}
		err = tx.q.SeasonEpisodeNumberingSet(ctx, schema.SeasonEpisodeNumberingSetParams{
			Number:    wantNum,
			Label:     wantLabel,
			Slug:      wantSlug,
			SeasonID:  sn.ID,
			EpisodeID: snep.EpisodeID,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (tx *TxRW) EpisodeTitleSet(ctx Context, id, title string) error {
	ep, err := tx.q.EpisodeGet(ctx, id)
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventLiveUpdate,
			Addr:    (&Episode{ep: ep}).TitleAddr(),
			NewText: title,
			OldText: ep.Title,
		})
	})
	err = tx.q.EpisodeTitleSet(ctx, schema.EpisodeTitleSetParams{
		Title: title,
		ID:    id,
	})
	if err != nil {
		return err
	}
	sneps, err := tx.q.SeasonEpisodeListByEpisodeID(ctx, id)
	if err != nil {
		return err
	}
	for _, snep := range sneps {
		slug, err := tx.generateEpisodeSlug(ctx, snep.EditionID, snep.Slug, episodeSlugBase(snep.Slug), title, id)
		if err != nil {
			return err
		}
		if slug != snep.Slug {
			err = tx.q.SeasonEpisodeSlugSet(ctx, schema.SeasonEpisodeSlugSetParams{
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

// SeasonEpisodeAdd creates a new regular episode at the end of the
// given season with reasonable defaults.
func (tx *TxRW) SeasonEpisodeAdd(ctx Context, seasonID string) error {
	sn, err := tx.q.SeasonGet(ctx, seasonID)
	if err != nil {
		return err
	}

	existing, err := tx.q.SeasonEpisodeListBySeasonID(ctx, sn.ID)
	if err != nil {
		return err
	}

	// Count existing regular episodes to determine the next number.
	var maxNum int64
	var maxSortKey string
	for _, snep := range existing {
		maxNum = max(maxNum, snep.Number)
		maxSortKey = max(maxSortKey, snep.SortKey)
	}
	num := maxNum + 1
	sortKey := maxSortKey + "~"
	base := fmt.Sprintf("s%02de%02d", sn.Number, num)
	title := "New Episode"
	label := strconv.FormatInt(num, 10)

	ep, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
		Title:   title,
		Summary: "The main character encounters an unexpected challenge!",
		Type:    "regular",
	})
	if err != nil {
		return err
	}

	slug, err := tx.generateEpisodeSlug(ctx, sn.EditionID, "", base, title, ep.ID)
	if err != nil {
		return err
	}

	return tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
		EditionID: sn.EditionID,
		SeasonID:  sn.ID,
		EpisodeID: ep.ID,
		SortKey:   sortKey,
		Label:     label,
		Number:    num,
		Slug:      slug,
	})
}

// generateEpisodeSlug builds an episode slug from a base (e.g. "s01e05"
// or "s02-special") and a title.
// If the result matches oldSlug, it is returned as-is;
// otherwise a uniqueness check deduplicates within the edition.
func (tx *TxRW) generateEpisodeSlug(ctx Context, editionID, oldSlug, base, title, id string) (string, error) {
	slug := base
	if titleSlug := xstrings.ToSlug(title); titleSlug != "" {
		slug += "-" + titleSlug
	}
	if slug == oldSlug {
		return slug, nil
	}
	n, err := tx.q.SeasonEpisodeSlugExists(ctx, schema.SeasonEpisodeSlugExistsParams{
		EditionID: editionID,
		Slug:      slug,
	})
	if err != nil {
		return "", err
	}
	if n == 0 {
		return slug, nil
	}
	// Append the ID as a last resort.
	return slug + "-" + id, nil
}

// episodeSlugBase extracts the "sNNeNN" or "sNN-special" prefix
// from an episode slug segment like "s01e05-the-title".
func episodeSlugBase(epPart string) string {
	// Match "sNNeNN": s, digits, e, digits.
	i := 0
	if i >= len(epPart) || epPart[i] != 's' {
		return epPart
	}
	i++
	for i < len(epPart) && epPart[i] >= '0' && epPart[i] <= '9' {
		i++
	}
	if i < len(epPart) && epPart[i] == 'e' {
		i++
		for i < len(epPart) && epPart[i] >= '0' && epPart[i] <= '9' {
			i++
		}
		return epPart[:i]
	}
	// Try "sNN-special".
	if i < len(epPart) && epPart[i] == '-' {
		const special = "-special"
		if strings.HasPrefix(epPart[i:], special) {
			return epPart[:i+len(special)]
		}
	}
	return epPart
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
