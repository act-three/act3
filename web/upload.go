package web

import (
	"fmt"
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
)

func (c *Config) doUpload(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	ctx := req.Context()
	file, _, err := req.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	medID := req.FormValue("med-id")
	sedID := req.FormValue("sed-id")
	epID := req.FormValue("ep-id")
	colID := req.FormValue("col-id")

	var kind model.ImageKind
	switch {
	case medID != "", sedID != "":
		kind = model.ImagePoster
	case epID != "":
		kind = model.ImageThumbnail
	case colID != "":
		kind = model.ImageBanner
	default:
		return nil, &model.ValidationError{
			Op:  "params",
			Err: fmt.Errorf("missing param med-id, sed-id, ep-id, or col-id"),
		}
	}

	originalID, err := c.Model.ImageCreate(ctx, file, kind)
	if err != nil {
		return nil, err
	}

	switch {
	case medID != "":
		_, err = c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
			return nil, tx.MovieEditionPosterIDSet(ctx, medID, originalID)
		})
	case sedID != "":
		_, err = c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
			return nil, tx.SeriesEditionPosterIDSet(ctx, sedID, originalID)
		})
	case epID != "":
		_, err = c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
			return nil, tx.EpisodeThumbnailIDSet(ctx, epID, originalID)
		})
	case colID != "":
		_, err = c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
			return nil, tx.CollectionBannerIDSet(ctx, colID, originalID)
		})
	}
	if err != nil {
		return nil, err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil, nil
}
