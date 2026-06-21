package tvmaze

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"kr.dev/errorfmt"

	"ily.dev/act3/http/timing"
)

const (
	taskFetchEpisodes = "fetch-episodes"
)

var taskTypes = []string{
	taskFetchEpisodes,
}

var baseURL = *must1(url.Parse("https://api.tvmaze.com/"))

// apiLimit keeps API calls inside TVmaze's documented rate limit of
// 20 calls per 10 seconds per IP: a sustained 2/s with enough burst
// for interactive use to feel instant.
var apiLimit = rate.NewLimiter(rate.Every(500*time.Millisecond), 10)

type Client struct {
	client http.Client

	cacheMu   sync.Mutex
	cacheShow map[int]*Show
}

func New() *Client {
	c := &Client{
		cacheShow: map[int]*Show{},
	}
	c.client.Transport = timing.Transport("tvmaze", nil)
	return c
}

func (c *Client) setCacheShow(s *Show) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.cacheShow[s.ID] = s
}

func (c *Client) getCacheShow(id int) *Show {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	return c.cacheShow[id]
}

func (c *Client) Search(ctx context.Context, q string) ([]*Result, error) {
	var v []*Result
	err := c.getf(ctx, &v, "/search/shows", params("q", q))
	if err != nil {
		return nil, err
	}
	for _, r := range v {
		c.setCacheShow(&r.Show)
	}
	return v, nil
}

func (c *Client) GetShow(ctx context.Context, id int) (*Show, error) {
	s := c.getCacheShow(id)
	if s != nil {
		return s, nil
	}
	var v *Show
	err := c.getf(ctx, &v, "/shows/%d", id)
	if err != nil {
		return nil, err
	}
	c.setCacheShow(v)
	return v, nil
}

func (c *Client) getf(ctx context.Context, v any, format string, args ...any) (err error) {
	defer errorfmt.Handlef("getf %s %v: %w", format, args, &err)
	slog.InfoContext(ctx, "getf", "format", format, "args", args)
	if err := apiLimit.Wait(ctx); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", "", nil)
	if err != nil {
		return err
	}
	req.URL = c.urlf(format, args...)
	slog.DebugContext(ctx, "get", "url", req.URL)
	req.Header.Set("User-Agent", "act3-prototype/0.0")
	t0 := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	slog.InfoContext(ctx, "response", "status", resp.StatusCode, "elapsed", time.Since(t0))
	if resp.StatusCode != 200 {
		return fmt.Errorf("bad status %d: %s", resp.StatusCode, resp.Status)
	}
	err = json.UnmarshalRead(http.MaxBytesReader(nil, resp.Body, 1<<20), v)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) urlf(format string, args ...any) *url.URL {
	u := baseURL.Clone()
	if len(args) > 0 {
		switch v := args[len(args)-1].(type) {
		case url.Values:
			u.RawQuery = v.Encode()
			args = args[:len(args)-1]
		}
	}
	u.Path = path.Join(baseURL.Path, fmt.Sprintf(format, args...))
	return u
}

func params(s ...string) url.Values {
	if len(s)%2 != 0 {
		panic("odd param count")
	}
	v := url.Values{}
	for len(s) > 0 {
		v.Add(s[0], s[1])
		s = s[2:]
	}
	return v
}

func must1[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
