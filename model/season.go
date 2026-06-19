package model

import (
	"fmt"
	"iter"
	"strconv"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/model/progress"
	"ily.dev/act3/xiter"
)

type SeasonHead struct {
	sn schema.Season
}

func (sn *SeasonHead) ID() string        { return sn.sn.ID }
func (sn *SeasonHead) EditionID() string { return sn.sn.EditionID }
func (sn *SeasonHead) Title() string     { return sn.sn.Title }
func (sn *SeasonHead) Number() int       { return int(sn.sn.Number) }

// Slug returns a synthetic slug, suitable for use in page anchors.
func (sn *SeasonHead) Slug() string {
	return fmt.Sprintf("s%d", sn.sn.Number)
}

func (sn *SeasonHead) addr(field string) []string {
	return []string{"season", sn.ID(), field}
}

func (sn *SeasonHead) TitleAddr() []string            { return sn.addr("title") }
func (sn *SeasonHead) TitleField() (string, []string) { return sn.Title(), sn.TitleAddr() }

type Season struct {
	SeasonHead
	eps    []*Episode
	epByID map[string]*Episode
	so     *SeriesEditionHead
	sr     *SeriesHead
}

func newSeason(
	sr *SeriesHead,
	so *SeriesEditionHead,
	snData schema.Season,
	sneps []schema.SeasonEpisode,
	epByID map[string]*schema.Episode,
	progByEpisodeID func(string) []*progress.Item,
	videosByEpisodeID map[string][]*Video,
) *Season {
	sn := &Season{
		SeasonHead: SeasonHead{snData},
		sr:         sr,
		so:         so,
		epByID:     map[string]*Episode{},
	}
	for _, snepData := range sneps {
		epData := epByID[snepData.EpisodeID]
		if epData == nil {
			panic("cannot find episode " + snepData.EpisodeID)
		}
		ep := newEpisode(sr, so, &sn.SeasonHead, snepData, *epData,
			progByEpisodeID(epData.ID),
			videosByEpisodeID[epData.ID],
		)
		sn.eps = append(sn.eps, ep)
		sn.epByID[ep.ID()] = ep
	}
	return sn
}

func (sn *Season) Series() *SeriesHead {
	return sn.sr
}

func (sn *Season) NumEpisodes(include EpisodeType) int {
	if include == AnyEpisode {
		return len(sn.eps)
	}
	n := 0
	for range sn.Episodes(include) {
		n++
	}
	return n
}

// Episodes returns the specified episodes from sn.
// Parameter include is a bit field.
func (sn *Season) Episodes(include EpisodeType) iter.Seq[*Episode] {
	return func(yield func(*Episode) bool) {
		for i := range sn.eps {
			ep := sn.eps[i]
			if ep.HasType(include) && !yield(ep) {
				return
			}
		}
	}
}

func (sn *Season) episodeByID(id string) *Episode {
	return sn.epByID[id]
}

// SeasonInEdition loads a full Season (with episodes) by season ID.
func (tx *TxR) SeasonInEdition(seasonID string) (*Season, error) {
	snData, err := tx.q.SeasonGet(seasonID)
	if err != nil {
		return nil, err
	}
	sed, err := tx.SeriesEdition(snData.EditionID)
	if err != nil {
		return nil, err
	}
	for sn := range sed.Seasons() {
		if sn.ID() == seasonID {
			return sn, nil
		}
	}
	return nil, fmt.Errorf("season %s not found in edition %s", seasonID, snData.EditionID)
}

func (tx *TxRW) SeasonTitleSet(id, title string) error {
	return tx.q.SeasonTitleSet(schema.SeasonTitleSetParams{
		Title: title,
		ID:    id,
	})

}

func (sn *Season) episodeByNumber(n int) *Episode {
	if sn == nil {
		return nil
	}
	for ep := range xiter.Drop(sn.Episodes(Regular), n-1) {
		return ep
	}
	return nil
}

// SeasonAdd creates a new empty season after all existing ones
// in the given edition.
func (tx *TxRW) SeasonAdd(editionID string) error {
	sns, err := tx.q.SeasonListByEditionID(editionID)
	if err != nil {
		return err
	}
	var maxNumber int64
	for _, sn := range sns {
		maxNumber = max(maxNumber, sn.Number)
	}
	next := maxNumber + 1
	_, err = tx.q.SeasonCreate(schema.SeasonCreateParams{
		EditionID: editionID,
		SortKey:   fmt.Sprintf("%03d", next),
		Title:     fmt.Sprintf("Season %d", next),
		Number:    next,
	})

	return err
}

// renumberSeason derives Number, Label, and Slug for every episode
// in a season.  Specials get Number 0 / Label "Special"; regular
// episodes are numbered sequentially starting from 1 in SortKey order.
// seasonEpisodeSortKeyBump shifts every live SeasonEpisode row in the
// season with SortKey >= sortKey up by one, making room for a newly-
// inserted or restored junction at sortKey. Two-phase (negate then
// un-negate) because SQLite's partial unique index on (SeasonID,
// SortKey) WHERE DeletedAt IS NULL rejects the transient duplicate a
// single-statement +1 UPDATE would produce.
func (tx *TxRW) seasonEpisodeSortKeyBump(seasonID string, sortKey int64) error {
	if err := tx.q.SeasonEpisodeSortKeyBump(schema.SeasonEpisodeSortKeyBumpParams{
		SeasonID: seasonID, SortKey: sortKey,
	}); err != nil {
		return err
	}
	return tx.q.SeasonEpisodeSortKeyBumpFinish(seasonID)
}

func (tx *TxRW) renumberSeason(seasonID string) error {
	sn, err := tx.q.SeasonGet(seasonID)
	if err != nil {
		return err
	}
	all, err := tx.q.SeasonEpisodeListBySeasonID(sn.ID)
	if err != nil {
		return err
	}
	eps := make(map[string]schema.Episode, len(all))
	for _, snep := range all {
		ep, err := tx.q.EpisodeGet(snep.EpisodeID)
		if err != nil {
			return err
		}
		eps[snep.EpisodeID] = ep
	}

	tx.emitDetail(Detail{SlugChangeID: sn.ID})

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

		wantSlug, err := tx.episodeFindSlug(snep.EditionID, sn.Number, wantNum, ep.Title, snep.Slug)
		if err != nil {
			return err
		}
		if snep.Number == wantNum && snep.Label == wantLabel && snep.Slug == wantSlug {
			continue
		}
		err = tx.q.SeasonEpisodeNumberingSet(schema.SeasonEpisodeNumberingSetParams{
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
