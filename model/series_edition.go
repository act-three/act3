package model

import (
	"iter"
	"path"
	"slices"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/model/progress"
)

type SeriesEditionHead struct {
	sed schema.SeriesEdition
}

func (sed *SeriesEditionHead) ID() string       { return sed.sed.ID }
func (sed *SeriesEditionHead) Slug() string     { return sed.sed.Slug }
func (sed *SeriesEditionHead) Label() string    { return sed.sed.Label }
func (sed *SeriesEditionHead) Summary() string  { return sed.sed.Summary }
func (sed *SeriesEditionHead) SeriesID() string { return sed.sed.SeriesID }

func (sed *SeriesEditionHead) Poster() Image {
	return Image{ID: sed.sed.PosterID, Kind: ImagePoster}
}

func (sed *SeriesEditionHead) addr(field string) []string {
	return []string{"series-edition", sed.ID(), field}
}

func (sed *SeriesEditionHead) LabelAddr() []string   { return sed.addr("label") }
func (sed *SeriesEditionHead) SummaryAddr() []string { return sed.addr("summary") }
func (sed *SeriesEditionHead) SlugAddr() []string    { return sed.addr("slug") }
func (sed *SeriesEditionHead) PosterAddr() []string  { return sed.addr("poster") }

func (sed *SeriesEditionHead) PosterField() (Image, []string) {
	return sed.Poster(), sed.PosterAddr()
}

func (sed *SeriesEditionHead) LabelField() (string, []string) { return sed.Label(), sed.LabelAddr() }
func (sed *SeriesEditionHead) SummaryField() (string, []string) {
	return sed.Summary(), sed.SummaryAddr()
}
func (sed *SeriesEditionHead) SlugField() (string, []string) { return sed.Slug(), sed.SlugAddr() }

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
		sed:    soData,
		sr:     sr,
		snByID: map[string]*Season{},
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

func (sed *SeriesEdition) TheaterPath() string {
	return path.Join(sed.sr.TheaterPath(), sed.Slug())
}

func (sed *SeriesEdition) EditorPath() string {
	return SeriesEditionEditorPath(sed.sr, &sed.SeriesEditionHead)
}

func SeriesEditionEditorPath(sr *SeriesHead, sed *SeriesEditionHead) string {
	return path.Join(sr.EditorPath(), sed.Slug())
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

func (tx *TxR) SeriesEditionHead(id string) (*SeriesEditionHead, error) {
	sedData, err := tx.q.SeriesEditionGet(id)
	if err != nil {
		return nil, err
	}
	return &SeriesEditionHead{sed: sedData}, nil
}

func (tx *TxR) SeriesEdition(id string) (*SeriesEdition, error) {
	sedData, err := tx.q.SeriesEditionGet(id)
	if err != nil {
		return nil, err
	}
	srData, err := tx.q.SeriesGetByEditionID(id)
	if err != nil {
		return nil, err
	}
	sns, err := tx.q.SeasonListByEditionID(id)
	if err != nil {
		return nil, err
	}
	sneps, err := tx.q.SeasonEpisodeListByEditionID(id)
	if err != nil {
		return nil, err
	}
	eps, err := tx.q.EpisodeListByEditionID(id)
	if err != nil {
		return nil, err
	}
	evs, err := tx.q.EpisodeVideoListByEditionID(id)
	if err != nil {
		return nil, err
	}
	vids, err := tx.q.VideoListByEditionID(id)
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

// seriesEditionBySlug looks up a series by its slug
// and returns the edition matching edSlug
// (empty string for the default edition).
func (tx *TxR) seriesEditionBySlug(slug, edSlug string) (*SeriesEdition, error) {
	sedData, err := tx.q.SeriesEditionGetBySlug(schema.SeriesEditionGetBySlugParams{
		SeriesSlug:  slug,
		EditionSlug: edSlug,
	})

	if err != nil {
		return nil, err
	}
	return tx.SeriesEdition(sedData.ID)
}

func (tx *TxR) SeriesEditionList(sr *SeriesHead) ([]*SeriesWork, error) {
	seds, err := tx.q.SeriesEditionListBySeriesID(sr.ID())
	if err != nil {
		return nil, err
	}
	works := make([]*SeriesWork, len(seds))
	for i := range seds {
		works[i] = &SeriesWork{
			SeriesHead: *sr,
			sed:        seds[i],
		}
	}
	return works, nil
}

func (tx *TxRW) seriesEditionCreate(label, seriesID, summary string) (schema.SeriesEdition, error) {
	slug, err := tx.generateSeriesEditionSlug(label, seriesID)
	if err != nil {
		return schema.SeriesEdition{}, err
	}
	return tx.q.SeriesEditionCreate(schema.SeriesEditionCreateParams{
		Label:    label,
		Slug:     slug,
		SeriesID: seriesID,
		Summary:  summary,
	})

}

// SeriesEditionClone creates a new edition by copying metadata,
// seasons, and season-episode mappings from the edition with the given srcID.
// Episodes themselves are shared, not copied.
func (tx *TxRW) SeriesEditionClone(srcID string) (*SeriesWork, error) {
	src, err := tx.q.SeriesEditionGet(srcID)
	if err != nil {
		return nil, err
	}
	newSed, err := tx.seriesEditionCreate("Copy of "+src.Label, src.SeriesID, src.Summary)
	if err != nil {
		return nil, err
	}
	sns, err := tx.q.SeasonListByEditionID(srcID)
	if err != nil {
		return nil, err
	}
	sneps, err := tx.q.SeasonEpisodeListByEditionID(srcID)
	if err != nil {
		return nil, err
	}
	snepsBySeason := snepMapBySeasonID(sneps)
	for _, sn := range sns {
		newSn, err := tx.q.SeasonCreate(schema.SeasonCreateParams{
			EditionID: newSed.ID,
			SortKey:   sn.SortKey,
			Title:     sn.Title,
			Number:    sn.Number,
		})

		if err != nil {
			return nil, err
		}
		for _, snep := range snepsBySeason[sn.ID] {
			err := tx.q.SeasonEpisodeCreate(schema.SeasonEpisodeCreateParams{
				EditionID: newSed.ID,
				SeasonID:  newSn.ID,
				EpisodeID: snep.EpisodeID,
				SortKey:   snep.SortKey,
				Label:     snep.Label,
				Number:    snep.Number,
				Slug:      snep.Slug,
			})

			if err != nil {
				return nil, err
			}
		}
	}
	srData, err := tx.q.SeriesGet(src.SeriesID)
	if err != nil {
		return nil, err
	}
	return &SeriesWork{
		SeriesHead: SeriesHead{srData},
		sed:        newSed,
	}, nil
}

func (tx *TxRW) SeriesEditionLabelSet(id, label string) error {
	if _, err := tx.q.SeriesEditionGet(id); err != nil {
		return err
	}
	err := tx.q.SeriesEditionLabelSet(schema.SeriesEditionLabelSetParams{
		Label: label,
		ID:    id,
	})

	if err != nil {
		return err
	}
	return tx.seriesEditionEnsureSlug(id)
}

// seriesEditionEnsureSlug is the SeriesEdition analog of
// movieEditionEnsureSlug.
func (tx *TxRW) seriesEditionEnsureSlug(id string) error {
	sed, err := tx.q.SeriesEditionGet(id)
	if err != nil {
		return err
	}
	var allow []string
	if sed.DeletedAt == nil {
		allow = []string{sed.Slug}
	}
	slug, err := tx.generateSeriesEditionSlug(sed.Label, sed.SeriesID, allow...)
	if err != nil {
		return err
	}
	if slug == sed.Slug {
		return nil
	}
	// Only a live rename is announced: a trashed edition's pages
	// have no viewers to follow it.
	if sed.DeletedAt == nil {
		tx.emitDetail(Detail{SlugChangeID: id})
	}
	return tx.q.SeriesEditionSlugSet(schema.SeriesEditionSlugSetParams{
		Slug: slug, ID: id,
	})

}

func (tx *TxRW) SeriesEditionPosterIDSet(id, posterID string) error {
	sed, err := tx.q.SeriesEditionGet(id)
	if err != nil {
		return err
	}
	err = tx.q.SeriesEditionPosterIDSet(schema.SeriesEditionPosterIDSetParams{
		PosterID: posterID,
		ID:       id,
	})

	if err != nil {
		return err
	}
	if isPlaceholderImageID(sed.PosterID) {
		return nil
	}
	return tx.imageDelete(sed.PosterID)
}

func (tx *TxRW) SeriesEditionSummarySet(id, summary string) error {
	return tx.q.SeriesEditionSummarySet(schema.SeriesEditionSummarySetParams{
		Summary: summary,
		ID:      id,
	})

}

// seriesEditionSetDefault promotes the given edition to be
// the default (Slug="") for its series.
// The previous default gets a slug generated from its label.
func (tx *TxRW) seriesEditionSetDefault(id string) error {
	sed, err := tx.q.SeriesEditionGet(id)
	if err != nil {
		return err
	}
	if sed.Slug == "" {
		return nil // already default
	}
	old, err := tx.q.SeriesEditionGetDefault(sed.SeriesID)
	if err != nil {
		return err
	}
	oldSlug, err := tx.generateSeriesEditionSlug(old.Label, sed.SeriesID)
	if err != nil {
		return err
	}
	tx.emitDetail(Detail{SlugChangeID: old.ID})
	tx.emitDetail(Detail{SlugChangeID: sed.ID})
	if err := tx.q.SeriesEditionSlugSet(schema.SeriesEditionSlugSetParams{
		Slug: oldSlug, ID: old.ID,
	}); err != nil {
		return err
	}
	return tx.q.SeriesEditionSlugSet(schema.SeriesEditionSlugSetParams{
		Slug: "", ID: sed.ID,
	})

}

func (tx *TxRW) generateSeriesEditionSlug(label, seriesID string, allow ...string) (string, error) {
	for slug := range editionSlugCandidates(label) {
		if slices.Contains(allow, slug) {
			return slug, nil
		}
		n, err := tx.q.SeriesEditionSlugExists(schema.SeriesEditionSlugExistsParams{
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
