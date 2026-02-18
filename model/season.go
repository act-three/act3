package model

import (
	"iter"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/xiter"
)

type SeasonHead struct {
	sn schema.Season
}

func (sn *SeasonHead) ID() string   { return sn.sn.ID }
func (sn *SeasonHead) Name() string { return sn.sn.Name }

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
	progByEpisodeID func(string) []ProgressItem,
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

func (sn *Season) episodeByNumber(n int) *Episode {
	if sn == nil {
		return nil
	}
	for ep := range xiter.Drop(sn.Episodes(Regular), n-1) {
		return ep
	}
	return nil
}
