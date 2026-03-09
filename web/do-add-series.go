package web

import (
	"net/http"
	"strconv"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/view"
)

func (c *Config) doAddSeries(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	ctx := req.Context()
	id, err := strconv.Atoi(req.FormValue("id"))
	if err != nil {
		return nil, &model.ValidationError{Op: "TVmaze ID", Err: err}
	}
	show, err := c.TVmaze.GetShow(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ss, err := tx.SeriesCreateByTVmazeID(ctx, show)
		if err != nil {
			return nil, err
		}
		return turbo.Frame("tvmaze-"+strconv.FormatInt(*ss.TVmazeID(), 10))(
			view.SeriesResultLink(ss),
			turbo.Prepend(view.EditMediaSeriesListItems,
				ListItems([]*model.SeriesHead{ss}, view.EditMediaSeriesListItem),
			),
		), nil
	})
}
