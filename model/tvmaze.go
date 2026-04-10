package model

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
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

		sns := map[int]schema.Season{}
		for _, ts := range seasons {
			title := ts.Name
			if title == "" {
				title = fmt.Sprintf("Season %d", ts.Number)
			}
			season, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
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
			ep, err := tx.q.EpisodeCreate(ctx, ne.Episode)
			if err != nil {
				return err
			}
			ne.SeasonEpisode.EditionID = sedID
			ne.SeasonEpisode.SeasonID = sns[ne.Season].ID
			ne.SeasonEpisode.EpisodeID = ep.ID
			err = tx.q.SeasonEpisodeCreate(ctx, ne.SeasonEpisode)
			if err != nil {
				return err
			}
			if ne.ImageURL != "" {
				err = tx.addTask(ctx, taskFetchEpisodeThumbnail, ep.ID, ne.ImageURL)
				if err != nil {
					return err
				}
			}
		}
		for _, sn := range sns {
			err = tx.renumberSeason(ctx, sn.ID)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (tx *TxR) taskFetchEpisodeThumbnail(ctx context.Context, args []string) error {
	epID := args[0]
	url := args[1]
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("bad status %d", resp.StatusCode)
	}
	thumbnailID, err := tx.m.ImageCreate(ctx, resp.Body, ImageThumbnail)
	if err != nil {
		return err
	}
	return tx.m.WithTxRW(func(tx *TxRW) error {
		return tx.EpisodeThumbnailIDSet(ctx, epID, thumbnailID)
	})
}

func (tx *TxR) taskFetchSeriesPoster(ctx context.Context, args []string) error {
	sedID := args[0]
	url := args[1]
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("bad status %d", resp.StatusCode)
	}
	posterID, err := tx.m.ImageCreate(ctx, resp.Body, ImagePoster)
	if err != nil {
		return err
	}
	return tx.m.WithTxRW(func(tx *TxRW) error {
		return tx.SeriesEditionPosterIDSet(ctx, sedID, posterID)
	})
}
