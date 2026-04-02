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
	switch {
	case medID != "":
		_, err = c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
			return nil, tx.MovieEditionPosterIDSet(ctx, medID, blobID)
		})
	case sedID != "":
		_, err = c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
			return nil, tx.SeriesEditionPosterIDSet(ctx, sedID, blobID)
		})
	case epID != "":
		_, err = c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
			return nil, tx.EpisodeThumbnailIDSet(ctx, epID, blobID)
		})
	default:
		return nil, &model.ValidationError{
			Op:  "params",
			Err: fmt.Errorf("missing param med-id, sed-id, or ep-id"),
		}
	}
	if err != nil {
		return nil, err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil, nil
}
