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
func (sed *SeriesEditionHead) Label() string          { return sed.sed.Label }
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

func (sed *SeriesEdition) TheaterURL() string {
	return path.Join(sed.sr.TheaterURL(), sed.Slug())
}

func (sed *SeriesEdition) EditorURL() string {
	return path.Join(sed.sr.EditorURL(), sed.Slug())
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

// SeriesEditionBySlug looks up a series by its slug
// and returns the edition matching edSlug
// (empty string for the default edition).
func (tx *TxR) SeriesEditionBySlug(ctx Context, slug, edSlug string) (*SeriesEdition, error) {
	sedData, err := tx.q.SeriesEditionGetBySlug(ctx, schema.SeriesEditionGetBySlugParams{
		SeriesSlug:  slug,
		EditionSlug: edSlug,
	})
	if err != nil {
		return nil, err
	}
	return tx.SeriesEdition(ctx, sedData.ID)
}

func (tx *TxR) SeriesEditionList(ctx Context, sr *SeriesHead) ([]*SeriesWork, error) {
	seds, err := tx.q.SeriesEditionListBySeriesID(ctx, sr.ID())
	if err != nil {
		return nil, err
	}
	works := make([]*SeriesWork, len(seds))
	for i := range seds {
		works[i] = &SeriesWork{
			SeriesHead:        *sr,
			SeriesEditionHead: SeriesEditionHead{seds[i]},
		}
	}
	return works, nil
}

func (tx *TxRW) seriesEditionCreate(ctx Context, label, seriesID, summary, tvmazeImageURL string) (schema.SeriesEdition, error) {
	slug, err := tx.generateSeriesEditionSlug(ctx, label, seriesID)
	if err != nil {
		return schema.SeriesEdition{}, err
	}
	return tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
		Label:          label,
		Slug:           slug,
		SeriesID:       seriesID,
		Summary:        summary,
		TVmazeImageURL: tvmazeImageURL,
	})
}

// SeriesEditionClone creates a new edition by copying metadata,
// seasons, and season-episode mappings from the edition with the given srcID.
// Episodes themselves are shared, not copied.
func (tx *TxRW) SeriesEditionClone(ctx Context, srcID string) (*SeriesEditionHead, error) {
	src, err := tx.q.SeriesEditionGet(ctx, srcID)
	if err != nil {
		return nil, err
	}
	newSed, err := tx.seriesEditionCreate(ctx,
		"Copy of "+src.Label, src.SeriesID, src.Summary, src.TVmazeImageURL)
	if err != nil {
		return nil, err
	}
	sns, err := tx.q.SeasonListByEditionID(ctx, srcID)
	if err != nil {
		return nil, err
	}
	sneps, err := tx.q.SeasonEpisodeListByEditionID(ctx, srcID)
	if err != nil {
		return nil, err
	}
	snepsBySeason := snepMapBySeasonID(sneps)
	for _, sn := range sns {
		newSn, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
			EditionID:      newSed.ID,
			SortKey:        sn.SortKey,
			Name:           sn.Name,
			Number:         sn.Number,
			TVmazeURL:      sn.TVmazeURL,
			Summary:        sn.Summary,
			EpisodeOrder:   sn.EpisodeOrder,
			PremieredOn:    sn.PremieredOn,
			EndedOn:        sn.EndedOn,
			TVmazeImageURL: sn.TVmazeImageURL,
		})
		if err != nil {
			return nil, err
		}
		for _, snep := range snepsBySeason[sn.ID] {
			err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
				SeasonID:  newSn.ID,
				EpisodeID: snep.EpisodeID,
				SortKey:   snep.SortKey,
				Label:     snep.Label,
				Number:    snep.Number,
			})
			if err != nil {
				return nil, err
			}
		}
	}
	return &SeriesEditionHead{newSed}, nil
}

func (tx *TxRW) generateSeriesEditionSlug(ctx Context, label, seriesID string) (string, error) {
	for slug := range editionSlugCandidates(label) {
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
