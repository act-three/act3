package model

import (
	"iter"
	"path"
	"slices"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/model/progress"
)

const (
	// AirDate is the primary edition, present in every series.
	// Other editions are optional.
	AirDate = "Air Date"
)

type SeriesEditionHead struct {
	sed schema.SeriesEdition
}

func (sed *SeriesEditionHead) ID() string             { return sed.sed.ID }
func (sed *SeriesEditionHead) Slug() string           { return sed.sed.Slug }
func (sed *SeriesEditionHead) Title() string          { return sed.sed.Title }
func (sed *SeriesEditionHead) Summary() string        { return sed.sed.Summary }
func (sed *SeriesEditionHead) TVmazeImageURL() string { return sed.sed.TVmazeImageURL }

type SeriesEdition struct {
	SeriesEditionHead
	sns    []*Season
	snByID map[string]*Season
	sr     *SeriesHead
}

func newSeriesEdition(
	sr *SeriesHead,
	soData schema.SeriesEdition,
	sns []schema.Season,
	snepBySeasonID map[string][]schema.SeasonEpisode,
	epByID map[string]*schema.Episode,
	progByEpisodeID func(string) []*progress.Item,
	videosByEpisodeID map[string][]*Video,
) *SeriesEdition {
	sed := &SeriesEdition{
		SeriesEditionHead: SeriesEditionHead{soData},
		sr:                sr,
		snByID:            map[string]*Season{},
	}
	for _, snData := range sns {
		sneps := snepBySeasonID[snData.ID]
		sn := newSeason(sr, &sed.SeriesEditionHead, snData, sneps, epByID, progByEpisodeID, videosByEpisodeID)
		sed.sns = append(sed.sns, sn)
		sed.snByID[sn.ID()] = sn
	}
	return sed
}

// Seasons returns all seasons in the order defined by sed.
func (sed *SeriesEdition) Seasons() iter.Seq[*Season] {
	return slices.Values(sed.sns)
}

func (sed *SeriesEdition) SeriesHead() *SeriesHead { return sed.sr }

func (sed *SeriesEdition) SeasonByID(id string) *Season {
	return sed.snByID[id]
}

// seasonByNumber returns season n in the order defined by sed.
func (sed *SeriesEdition) seasonByNumber(n int) *Season {
	if i := n - 1; i >= 0 && i < len(sed.sns) {
		return sed.sns[i]
	}
	return nil
}

func (sed *SeriesEdition) seasonByEpisodeID(id string) *Season {
	for sn := range sed.Seasons() {
		if ep := sn.episodeByID(id); ep != nil {
			return sn
		}
	}
	return nil
}

func (sed *SeriesEdition) episodeByID(id string) *Episode {
	for sn := range sed.Seasons() {
		if ep := sn.episodeByID(id); ep != nil {
			return ep
		}
	}
	return nil
}

// episodeByNumber finds an episode by number:
// episode number e within season number s.
// In this numbering convention, all specials appear in season 0.
// (See https://www.tvmaze.com/faq/15/episodes for more.)
// These numbers commonly appear in torrent filenames in the form SnnEnn.
func (sed *SeriesEdition) episodeByNumber(s, e int) *Episode {
	if s == 0 {
		i := 1
		for ep := range sed.Episodes(AnySpecial) {
			if i == e {
				return ep
			}
			i++
		}
	}
	return sed.seasonByNumber(s).episodeByNumber(e)
}

func (sed *SeriesEdition) Episodes(include EpisodeType) iter.Seq[*Episode] {
	return func(yield func(*Episode) bool) {
		for sn := range sed.Seasons() {
			for ep := range sn.Episodes(include) {
				if !yield(ep) {
					return
				}
			}
		}
	}
}

func (tx *TxR) SeriesEdition(ctx Context, id string) (*SeriesEdition, error) {
	sedData, err := tx.q.SeriesEditionGet(ctx, id)
	if err != nil {
		return nil, err
	}
	srData, err := tx.q.SeriesGetByEditionID(ctx, id)
	if err != nil {
		return nil, err
	}
	sns, err := tx.q.SeasonListByEditionID(ctx, id)
	if err != nil {
		return nil, err
	}
	sneps, err := tx.q.SeasonEpisodeListByEditionID(ctx, id)
	if err != nil {
		return nil, err
	}
	eps, err := tx.q.EpisodeListByEditionID(ctx, id)
	if err != nil {
		return nil, err
	}
	evs, err := tx.q.EpisodeVideoListByEditionID(ctx, id)
	if err != nil {
		return nil, err
	}
	vids, err := tx.q.VideoListByEditionID(ctx, id)
	if err != nil {
		return nil, err
	}

	vidByID := vidMapByID(vids)
	videosByEpisodeID := vidMapByEpisodeID(evs, vidByID)

	sr := &SeriesHead{srData}
	epByID := epMapByID(eps)
	snepBySeasonID := snepMapBySeasonID(sneps)
	sed := newSeriesEdition(sr, sedData, sns, snepBySeasonID, epByID, tx.m.prog.List, videosByEpisodeID)
	return sed, nil
}

func (sed *SeriesEdition) EditURL() string {
	if sed.sed.Slug == "" {
		return sed.sr.EditURL()
	}
	return path.Join(
		sed.sr.EditURL(),
		sed.sed.Slug,
	)
}

func (tx *TxR) SeriesEditionHeadList(ctx Context, seriesID string) ([]*SeriesEditionHead, error) {
	seds, err := tx.q.SeriesEditionListBySeriesID(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	heads := make([]*SeriesEditionHead, len(seds))
	for i := range seds {
		heads[i] = &SeriesEditionHead{seds[i]}
	}
	return heads, nil
}

func (tx *TxRW) SeriesEditionCreate(ctx Context, title, seriesID, summary, tvmazeImageURL string) (schema.SeriesEdition, error) {
	slug, err := tx.generateSeriesEditionSlug(ctx, title, seriesID)
	if err != nil {
		return schema.SeriesEdition{}, err
	}
	return tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
		Title:          title,
		Slug:           slug,
		SeriesID:       seriesID,
		Summary:        summary,
		TVmazeImageURL: tvmazeImageURL,
	})
}

func (tx *TxRW) generateSeriesEditionSlug(ctx Context, title, seriesID string) (string, error) {
	for slug := range editionSlugCandidates(title) {
		n, err := tx.q.SeriesEditionSlugExists(ctx, schema.SeriesEditionSlugExistsParams{
			SeriesID: seriesID,
			Slug:     slug,
		})
		if err != nil {
			return "", err
		}
		if n == 0 {
			return slug, nil
		}
	}
	panic("unreachable")
}
