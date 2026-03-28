package model

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/model/norm"
)

func (tx *TxR) taskFetchEpisodes(ctx context.Context, args []string) error {
	// TODO(em): pull info from Client
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "got id", "TVmazeID", id)

	seasons, err := tx.m.tvmaze.ListShowSeasons(ctx, id)
	if err != nil {
		return err
	}
	eps, err := tx.m.tvmaze.ListShowEpisodes(ctx, id)
	if err != nil {
		return err
	}

	sedID := args[1]

	return tx.m.WithTxRW(func(tx *TxRW) error {
		_, err := tx.q.SeriesGetByTVmazeID(ctx, &id)
		if err != nil {
			return err
		}

		sid := map[int]string{}
		for _, ts := range seasons {
			name := ts.Name
			if name == "" {
				name = fmt.Sprintf("Season %d", ts.Number)
			}
			season, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
				EditionID: sedID,
				SortKey:   fmt.Sprintf("%03d", ts.Number),
				Name:      name,
				Number:    int64(ts.Number),
			})
			if err != nil {
				return err
			}
			sid[ts.Number] = season.ID
		}

		neps := norm.TVmazeEpisodes(eps)
		for _, ne := range neps {
			ep, err := tx.q.EpisodeCreate(ctx, ne.Episode)
			if err != nil {
				return err
			}
			ne.SeasonEpisode.EditionID = sedID
			ne.SeasonEpisode.SeasonID = sid[ne.Season]
			ne.SeasonEpisode.EpisodeID = ep.ID
			err = tx.q.SeasonEpisodeCreate(ctx, ne.SeasonEpisode)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
