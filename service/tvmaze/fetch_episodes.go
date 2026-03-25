package tvmaze

import "context"

func (c *Client) ListShowSeasons(ctx context.Context, id int64) ([]Season, error) {
	var s []Season
	err := c.getf(ctx, &s, "/shows/%d/seasons", id)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (c *Client) ListShowEpisodes(ctx context.Context, id int64) ([]Episode, error) {
	var eps []Episode
	err := c.getf(ctx, &eps, "/shows/%d/episodes", id, params("specials", "1"))
	if err != nil {
		return nil, err
	}
	return eps, nil
}
