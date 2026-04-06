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
	blobID, err := c.Model.Store(file)
	if err != nil {
		return nil, err
	}
	medID := req.FormValue("med-id")
	sedID := req.FormValue("sed-id")
	epID := req.FormValue("ep-id")
	colID := req.FormValue("col-id")
	switch {
	case medID != "":
		_, err = c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
			return nil, tx.MovieEditionPosterKeySet(ctx, medID, blobID)
		})
	case sedID != "":
		_, err = c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
			return nil, tx.SeriesEditionPosterKeySet(ctx, sedID, blobID)
		})
	case epID != "":
		_, err = c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
			return nil, tx.EpisodeThumbnailKeySet(ctx, epID, blobID)
		})
	case colID != "":
		_, err = c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
			return nil, tx.CollectionBannerKeySet(ctx, colID, blobID)
		})
	default:
		return nil, &model.ValidationError{
			Op:  "params",
			Err: fmt.Errorf("missing param med-id, sed-id, ep-id, or col-id"),
		}
	}
	if err != nil {
		return nil, err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil, nil
}
