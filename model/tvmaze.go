package model

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"ily.dev/act3/database/schema"
)

func (tx *TxR) taskFetchEpisodes(ctx context.Context, args []string) func(tx *TxRW) error {
	// TODO(em): pull info from Client
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return taskError(err)
	}
	slog.InfoContext(ctx, "got id", "TVmazeID", id)

	seasons, err := tx.m.tvmaze.ListShowSeasons(ctx, id)
	if err != nil {
		return taskError(err)
	}
	eps, err := tx.m.tvmaze.ListShowEpisodes(ctx, id)
	if err != nil {
		return taskError(err)
	}

	return func(tx *TxRW) error {
		series, err := tx.q.SeriesGetByTVmazeID(ctx, &id)
		if err != nil {
			return err
		}
		sed, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Title:    AirDate,
			SeriesID: series.ID,
		})
		if err != nil {
			return err
		}

		sid := map[int]string{}
		for _, ts := range seasons {
			name := ts.Name
			switch {
			case name == "" && ts.Number == 0:
				name = "Specials"
			case name == "":
				name = fmt.Sprintf("Season %d", ts.Number)

			}
			season, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
				EditionID:      sed.ID,
				SortKey:        fmt.Sprintf("%03d", ts.Number),
				Name:           name,
				Number:         int64(ts.Number),
				TVmazeURL:      &ts.URL,
				Summary:        ts.Summary,
				EpisodeOrder:   ts.EpisodeOrder,
				PremieredOn:    &ts.PremiereDate,
				EndedOn:        &ts.EndDate,
				TVmazeImageURL: ts.Image.Medium(),
			})
			if err != nil {
				return err
			}
			sid[ts.Number] = season.ID
		}

		for _, te := range eps {
			ep, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
				Title:          te.Name,
				Summary:        te.Summary,
				Type:           te.Type,
				Airdate:        te.Airdate,
				Runtime:        int64(te.Runtime),
				TVmazeURL:      &te.URL,
				TVmazeImageURL: te.Image.Medium(),
			})
			if err != nil {
				return err
			}

			var num *int64
			sortNum := ""
			if te.Number != nil {
				sortNum = pad(*te.Number)
				n := int64(*te.Number)
				num = &n
			}

			label := "Unknown"
			if te.Number != nil {
				label = strconv.FormatInt(int64(*te.Number), 10)
			} else {
				switch te.Type {
				case "special", "insignificant_special":
					label = "Special"
				}
			}

			err = tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
				SeasonID:  sid[te.Season],
				EpisodeID: ep.ID,
				SortKey:   te.Airdate + "-" + sortNum,
				Number:    num,
				Label:     label,
			})
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func pad(n int) string {
	return fmt.Sprintf("%05d", n)
}
