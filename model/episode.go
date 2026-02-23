package model

import (
	"fmt"

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

var episodeTypeByName = map[string]EpisodeType{
	"regular":               Regular,
	"special":               SignificantSpecial,
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
func (ep *Episode) Title() string              { return ep.ep.Title }
func (ep *Episode) Airdate() string            { return ep.ep.Airdate }
func (ep *Episode) Summary() string            { return ep.ep.Summary }
func (ep *Episode) ImageURL() string           { return ep.ep.TVmazeImageURL }
func (ep *Episode) Progress() []*progress.Item { return ep.prog }
func (ep *Episode) Videos() []*Video           { return ep.videos }
func (ep *Episode) SnnEnn() string {
	if eNN := ep.snep.Number; eNN != nil {
		return fmt.Sprintf("S%02dE%02d", ep.sn.sn.Number, *eNN)
	}
	return fmt.Sprintf("S%02d Special", ep.sn.sn.Number)
}

func (ep *Episode) PlayerURL(v *Video) string {
	return fmt.Sprintf("/player/%s/%s/%s", v.ID(), ep.ID(), ep.so.ID())
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

func (ep *Episode) EditDialogURL() string {
	return "/dialog/edit-episode/" + ep.ep.ID
}

func (ep *Episode) SeriesHead() *SeriesHead {
	return ep.sr
}

func (ep *Episode) SeasonHead() *SeasonHead {
	return ep.sn
}

func (ep *Episode) DetailURL() string {
	s := "-special"
	if ep.HasType(Regular) && ep.snep.Number != nil {
		s = fmt.Sprintf("e%02d", *ep.snep.Number)
	}
	return fmt.Sprintf("/ep/%s-s%02de%s-%s-%s",
		xstrings.ToSlug(ep.sr.sr.Title),
		ep.sn.sn.Number,
		s,
		xstrings.ToSlug(ep.ep.Title),
		ep.ep.ID,
	)
}

// Episode is like EpisodeInEdition, but it assumes the Air Date edition.
func (tx *TxR) Episode(ctx Context, id string) (*Episode, error) {
	return tx.episodeInContext(ctx, id, "", AirDate)
}

func (tx *TxR) EpisodeInEdition(ctx Context, id, edID string) (*Episode, error) {
	return tx.episodeInContext(ctx, id, edID, "")
}

func (tx *TxR) episodeInContext(ctx Context, id, edID, edName string) (*Episode, error) {
	epRec, err := tx.q.EpisodeGet(ctx, id)
	if err != nil {
		return nil, err
	}
	sneps, err := tx.q.SeasonEpisodeListByEpisodeID(ctx, id)
	if err != nil {
		return nil, err
	}
	ep := &Episode{
		ep: epRec,
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
		if seq.ID != edID && seq.Title != edName && i < len(sneps)-1 {
			continue
		}
		sr, err := tx.q.SeriesGet(ctx, seq.SeriesID)
		if err != nil {
			return nil, err
		}
		ep.snep = snep
		ep.sn = &SeasonHead{sn}
		ep.sr = &SeriesHead{sr}
		return ep, nil
	}
	return nil, fmt.Errorf("cannot load ep")
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
