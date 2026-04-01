package tmdb

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

	"kr.dev/errorfmt"

	"ily.dev/act3/http/timing"
)

var baseURL = *must(url.Parse("https://api.themoviedb.org"))

// PosterURL returns the full URL for a TMDB image path.
func PosterURL(posterPath string) string {
	return "https://image.tmdb.org/t/p/w500" + posterPath
}

// Client is a TMDB API client.
type Client struct {
	client http.Client

	tokenMu sync.RWMutex
	token   string

	cacheMu    sync.Mutex
	cacheMovie map[int]*Movie
}

// New creates a new TMDB client.
func New() *Client {
	c := &Client{
		cacheMovie: map[int]*Movie{},
	}
	c.client.Transport = timing.Transport("tmdb", nil)
	return c
}

// SetToken updates the bearer token used for API requests.
func (c *Client) SetToken(token string) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	c.token = token
}

func (c *Client) getToken() string {
	c.tokenMu.RLock()
	defer c.tokenMu.RUnlock()
	return c.token
}

// Configured reports whether an access token has been set.
func (c *Client) Configured() bool {
	return c.getToken() != ""
}

func (c *Client) setCacheMovie(m *Movie) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.cacheMovie[m.ID] = m
}

func (c *Client) getCacheMovie(id int) *Movie {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	return c.cacheMovie[id]
}

// Search searches for movies by title.
func (c *Client) Search(ctx context.Context, q string) ([]SearchResult, error) {
	var v SearchResponse
	err := c.getf(ctx, &v, "/3/search/movie", params("query", q))
	if err != nil {
		return nil, err
	}
	return v.Results, nil
}

// GetMovie fetches full movie details by TMDB ID.
func (c *Client) GetMovie(ctx context.Context, id int) (*Movie, error) {
	m := c.getCacheMovie(id)
	if m != nil {
		return m, nil
	}
	var v *Movie
	err := c.getf(ctx, &v, "/3/movie/%d", id)
	if err != nil {
		return nil, err
	}
	c.setCacheMovie(v)
	return v, nil
}

func (c *Client) getf(ctx context.Context, v any, format string, args ...any) (err error) {
	defer errorfmt.Handlef("tmdb getf %s %v: %w", format, args, &err)
	slog.InfoContext(ctx, "tmdb getf", "format", format, "args", args)
	token := c.getToken()
	if token == "" {
		return fmt.Errorf("TMDB access token not configured")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "", nil)
	if err != nil {
		return err
	}
	req.URL = c.urlf(format, args...)
	slog.DebugContext(ctx, "tmdb get", "url", req.URL)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "act3-prototype/0.0")
	t0 := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	slog.InfoContext(ctx, "tmdb response", "status", resp.StatusCode, "elapsed", time.Since(t0))
	if resp.StatusCode != 200 {
		return fmt.Errorf("bad status %d: %s", resp.StatusCode, resp.Status)
	}
	err = json.UnmarshalRead(resp.Body, v)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) urlf(format string, args ...any) *url.URL {
	var u url.URL
	u = baseURL
	if len(args) > 0 {
		switch v := args[len(args)-1].(type) {
		case url.Values:
			u.RawQuery = v.Encode()
			args = args[:len(args)-1]
		}
	}
	u.Path = path.Join(baseURL.Path, fmt.Sprintf(format, args...))
	return &u
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

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
