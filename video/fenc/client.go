package fenc

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"encoding/json/v2"
)

// A Client speaks the agent protocol.
// Its zero value is not useful; set BaseURL for a remote agent,
// or construct one with [NewInProcessClient].
type Client struct {
	// BaseURL is the agent's base URL, e.g. "http://fenc:4446".
	BaseURL string

	// HTTPClient makes the requests; nil means http.DefaultClient.
	HTTPClient *http.Client
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// Hello fetches the agent's handshake:
// its protocol version and core count.
func (c *Client) Hello(ctx context.Context) (Hello, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/", nil)
	if err != nil {
		return Hello{}, err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return Hello{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Hello{}, respError(resp)
	}
	var h Hello
	if err := json.UnmarshalRead(resp.Body, &h); err != nil {
		return Hello{}, err
	}
	return h, nil
}

// Job runs one tool invocation to completion,
// reporting each streamed progress position to onProgress
// (which may be nil).
// The returned Result reflects the tool's exit;
// a non-nil error means the job was rejected or its outcome could
// not be determined.
// Cancelling ctx abandons the job, and the agent kills the tool.
func (c *Client) Job(ctx context.Context, job JobRequest, onProgress func(time.Duration)) (*Result, error) {
	body, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.BaseURL+"/job", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, respError(resp)
	}

	scanner := bufio.NewScanner(resp.Body)
	// The done event carries bounded stderr (up to ~200 KB before
	// JSON escaping), far past Scanner's default line limit.
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		var ev Event
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			return nil, fmt.Errorf("bad event from agent: %w", err)
		}
		if ev.Done != nil {
			return ev.Done, nil
		}
		if ev.OutTimeUS > 0 && onProgress != nil {
			onProgress(time.Duration(ev.OutTimeUS) * time.Microsecond)
		}
	}
	err = scanner.Err()
	if err == nil {
		err = io.ErrUnexpectedEOF
	}
	return nil, fmt.Errorf("job stream ended without a done event: %w", err)
}

// DeleteStats releases the stats directory id on the agent.
func (c *Client) DeleteStats(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		c.BaseURL+"/stats/"+url.PathEscape(id), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return respError(resp)
	}
	return nil
}

// respError turns a non-success agent response into an error
// carrying the response's text.
func respError(resp *http.Response) error {
	msg, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
	return fmt.Errorf("agent: %s: %s", resp.Status, strings.TrimSpace(string(msg)))
}
