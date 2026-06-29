package model

import (
	"fmt"
	"log/slog"
	"strconv"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/model/norm"
	"ily.dev/act3/priority"
)

func (tx *TxR) taskFetchEpisodes(args []string) error {
	// TODO(em): pull info from Client
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return permanent(err)
	}
	slog.InfoContext(tx.ctx, "got id", "TVmazeID", id)

	seasons, err := tx.m.tvmaze.ListShowSeasons(tx.ctx, id)
	if err != nil {
		return err
	}
	eps, err := tx.m.tvmaze.ListShowEpisodes(tx.ctx, id)
	if err != nil {
		return err
	}

	sedID := args[1]

	return tx.m.WithTxRW(tx.ctx, func(tx *TxRW) error {
		_, err := tx.q.SeriesGetByTVmazeID(&id)
		if err != nil {
			return err
		}

		sns := map[int]schema.Season{}
		for _, ts := range seasons {
			title := ts.Name
			if title == "" {
				title = fmt.Sprintf("Season %d", ts.Number)
			}
			season, err := tx.q.SeasonCreate(schema.SeasonCreateParams{
				EditionID: sedID,
				SortKey:   fmt.Sprintf("%03d", ts.Number),
				Title:     title,
				Number:    int64(ts.Number),
			})

			if err != nil {
				return err
			}
			sns[ts.Number] = season
		}

		neps := norm.TVmazeEpisodes(eps)
		for _, ne := range neps {
			ep, err := tx.q.EpisodeCreate(ne.Episode)
			if err != nil {
				return err
			}
			ne.SeasonEpisode.EditionID = sedID
			ne.SeasonEpisode.SeasonID = sns[ne.Season].ID
			ne.SeasonEpisode.EpisodeID = ep.ID
			err = tx.q.SeasonEpisodeCreate(ne.SeasonEpisode)
			if err != nil {
				return err
			}
			if ne.ImageURL != "" {
				err = tx.addTaskWithPriority(priority.FetchThumbnail, taskFetchEpisodeThumbnail, ep.ID, ne.ImageURL)
				if err != nil {
					return err
				}
			}
		}
		for _, sn := range sns {
			err = tx.renumberSeason(sn.ID)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (tx *TxR) taskFetchEpisodeThumbnail(args []string) error {
	epID := args[0]
	url := args[1]
	thumbnailID, err := tx.m.imageFetch(tx.ctx, url, ImageThumbnail)
	if err != nil {
		return err
	}
	return tx.m.WithTxRW(tx.ctx, func(tx *TxRW) error {
		return tx.EpisodeThumbnailIDSet(epID, thumbnailID)
	})
}

func (tx *TxR) taskFetchSeriesPoster(args []string) error {
	sedID := args[0]
	url := args[1]
	posterID, err := tx.m.imageFetch(tx.ctx, url, ImagePoster)
	if err != nil {
		return err
	}
	return tx.m.WithTxRW(tx.ctx, func(tx *TxRW) error {
		return tx.SeriesEditionPosterIDSet(sedID, posterID)
	})
}
