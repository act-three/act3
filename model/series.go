package model

import (
	"iter"
	"log/slog"
	"maps"
	"slices"
	"strconv"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/xstrings"
)

type SeriesHead struct {
	sr schema.Series
}

func newSeriesHeadList(list []schema.Series) []*SeriesHead {
	sss := make([]*SeriesHead, len(list))
	for i := range sss {
		sss[i] = &SeriesHead{list[i]}
	}
	return sss
}

func (sr *SeriesHead) ID() string             { return sr.sr.ID }
func (sr *SeriesHead) PremieredOn() *string   { return sr.sr.PremieredOn }
func (sr *SeriesHead) Status() string         { return sr.sr.Status }
func (sr *SeriesHead) Summary() string        { return sr.sr.Summary }
func (sr *SeriesHead) Title() string          { return sr.sr.Title }
func (sr *SeriesHead) TVmazeID() *int64       { return sr.sr.TVmazeID }
func (sr *SeriesHead) TVmazeImageURL() string { return sr.sr.TVmazeImageURL }

func (sr *SeriesHead) PlayURL() string {
	return "/series/" + sr.sr.ID
}

func (sr *SeriesHead) EditURL() string {
	return "/edit/series/" + xstrings.ToSlug(sr.sr.Title) + "-" + sr.sr.ID
}

type Series struct {
	SeriesHead
	soByID    map[string]*SeriesEdition
	soByTitle map[string]*SeriesEdition
	epByID    map[string]*Episode
}

func newSeries(
	srData schema.Series,
	seds []schema.SeriesEdition,
	sns []schema.Season,
	sneps []schema.SeasonEpisode,
	eps []schema.Episode,
	progByEpisodeID func(string) []ProgressItem,
) *Series {
	sr := &Series{
		SeriesHead: SeriesHead{srData},
		soByID:     map[string]*SeriesEdition{},
		soByTitle:  map[string]*SeriesEdition{},
		epByID:     map[string]*Episode{},
	}

	epByID := epMapByID(eps)
	snepBySeasonID := snepMapBySeasonID(sneps)

	snByEditionID := map[string][]schema.Season{}
	for i := range sns {
		soID := sns[i].EditionID
		snByEditionID[soID] = append(snByEditionID[soID], sns[i])
	}

	for _, soData := range seds {
		sns := snByEditionID[soData.ID]
		sed := newSeriesEdition(&sr.SeriesHead, soData, sns, snepBySeasonID, epByID,
			progByEpisodeID,
		)
		sr.soByID[sed.ID()] = sed
		sr.soByTitle[sed.Title()] = sed
		slog.Debug("loaded series edition", "id", sed.ID(), "title", sed.Title())
		for sn := range sed.Seasons() {
			slog.Debug("loaded season", "id", sn.ID(), "name", sn.Name())
			for ep := range sn.Episodes(AnyEpisode) {
				slog.Debug("loaded episode", "id", ep.ID(), "label", ep.Label())
			}
		}
	}
	return sr
}

func (sr *Series) EditionByTitle(title string) *SeriesEdition {
	if sr == nil {
		return nil
	}
	return sr.soByTitle[title]
}

func (sr *Series) SeriesEditionSeq() iter.Seq[*SeriesEdition] {
	names := slices.Collect(maps.Keys(sr.soByTitle))
	slices.Sort(names)
	return func(yield func(*SeriesEdition) bool) {
		for _, name := range names {
			if !yield(sr.soByTitle[name]) {
				return
			}
		}
	}
}

func (tx *TxR) SeriesHeadList(ctx Context) ([]*SeriesHead, error) {
	a, err := tx.q.SeriesList(ctx)
	if err != nil {
		return nil, err
	}
	return newSeriesHeadList(a), nil
}

func (tx *TxR) SeriesHeadListByTVmazeID(ctx Context, id []*int64) ([]*SeriesHead, error) {
	a, err := tx.q.SeriesListByTVmazeID(ctx, id)
	if err != nil {
		return nil, err
	}
	return newSeriesHeadList(a), nil
}

func (tx *TxRW) SeriesCreateByTVmazeID(ctx Context, id string) (*SeriesHead, error) {
	id64, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, &ValidationError{Op: "TVmaze ID", Err: err}
	}
	show, err := tx.m.tvmaze.GetShow(ctx, int(id64))
	if err != nil {
		return nil, err
	}
	srData, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
		Title:       show.Name,
		Status:      show.Status,
		Language:    show.Language,
		PremieredOn: show.Premiered,
		EndedOn:     show.Ended,
		//Network:     show.Network,
		Summary: show.Summary,

		TVmazeID:        &id64,
		TVmazeURL:       &show.URL,
		TVmazeImageURL:  show.Image.Medium(),
		TVmazeUpdatedAt: int64(show.Updated),
		IMDBID:          show.Externals.IMDB,
		TVDBID:          show.Externals.TheTVDB,
		TVRageID:        show.Externals.TVRage,
	})
	if err != nil {
		return nil, err
	}
	srh := &SeriesHead{srData}
	err = tx.addTask(ctx, taskFetchEpisodes, strconv.FormatInt(id64, 10))
	if err != nil {
		return nil, err
	}
	for _, g := range show.Genres {
		err := tx.q.SeriesGenreAdd(ctx, schema.SeriesGenreAddParams{
			SeriesID:  srh.sr.ID,
			GenreName: g,
		})
		if err != nil {
			return nil, err
		}
	}
	return srh, nil
}

func (tx *TxR) Series(ctx Context, id string) (*Series, error) {
	srData, err := tx.q.SeriesGet(ctx, id)
	if err != nil {
		return nil, err
	}
	seds, err := tx.q.SeriesEditionListBySeriesID(ctx, id)
	if err != nil {
		return nil, err
	}
	sns, err := tx.q.SeasonListBySeriesID(ctx, id)
	if err != nil {
		return nil, err
	}
	sneps, err := tx.q.SeasonEpisodeListBySeriesID(ctx, id)
	if err != nil {
		return nil, err
	}
	eps, err := tx.q.EpisodeListBySeriesID(ctx, id)
	if err != nil {
		return nil, err
	}

	sr := newSeries(srData, seds, sns, sneps, eps,
		tx.m.prog.getByEpisodeID,
	)
	return sr, nil
}
