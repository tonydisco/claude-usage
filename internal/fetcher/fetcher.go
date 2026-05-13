// Package fetcher talks to the (unofficial) claude.ai Usage endpoint.
//
// This is the ONE file that needs patching when Anthropic changes the
// endpoint shape. Keep it small and self-contained.
//
// Endpoint captured 2026-05-13 from claude.ai/settings/usage:
//   GET https://claude.ai/api/organizations/<org_uuid>/usage
package fetcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const usageEndpoint = "https://claude.ai/api/organizations/%s/usage"

// Default User-Agent. claude.ai serves different responses to obviously-bot
// UAs; matching a recent Chrome string is the safe default.
const defaultUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"

// Client fetches Usage snapshots.
type Client struct {
	HTTP          *http.Client
	SessionCookie string // value of `sessionKey` cookie on claude.ai
	OrgID         string // organization UUID; required by the endpoint
	UserAgent     string

	// Mock makes Fetch return data from samples/usage-response.json instead
	// of hitting the network. Useful before Phase 0 is complete.
	Mock bool

	// MockPath overrides where mock data is read from. Defaults to
	// "samples/usage-response.json" relative to the working directory.
	MockPath string
}

// New returns a Client with reasonable defaults.
func New(sessionCookie, orgID string) *Client {
	return &Client{
		HTTP:          &http.Client{Timeout: 15 * time.Second},
		SessionCookie: sessionCookie,
		OrgID:         orgID,
		UserAgent:     defaultUA,
	}
}

// Fetch returns the current Usage snapshot.
func (c *Client) Fetch(ctx context.Context) (*Usage, error) {
	if c.Mock {
		return c.fetchMock()
	}
	return c.fetchHTTP(ctx)
}

func (c *Client) fetchMock() (*Usage, error) {
	path := c.MockPath
	if path == "" {
		path = "samples/usage-response.json"
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open mock data: %w", err)
	}
	defer f.Close()
	return decode(f)
}

func (c *Client) fetchHTTP(ctx context.Context) (*Usage, error) {
	if c.SessionCookie == "" {
		return nil, ErrNoCredential
	}
	if c.OrgID == "" {
		return nil, errors.New("org_id is empty; run `claude-usage config set org_id <id>`")
	}

	url := fmt.Sprintf(usageEndpoint, c.OrgID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json")
	// Header observed on real requests; the API rejects calls without it.
	req.Header.Set("anthropic-client-platform", "web_claude_ai")
	req.Header.Set("Referer", "https://claude.ai/settings/usage")
	req.AddCookie(&http.Cookie{Name: "sessionKey", Value: c.SessionCookie})

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request usage: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, ErrAuthExpired
	case http.StatusTooManyRequests:
		return nil, ErrRateLimited
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("usage endpoint: HTTP %d: %s", resp.StatusCode, string(body))
	}

	return decode(resp.Body)
}

func decode(r io.Reader) (*Usage, error) {
	var u Usage
	dec := json.NewDecoder(r)
	// We intentionally allow unknown fields: claude.ai's payload is
	// undocumented and may grow. Extra keys should not break the CLI.
	if err := dec.Decode(&u); err != nil {
		return nil, fmt.Errorf("decode usage response: %w", err)
	}
	u.FetchedAt = time.Now()
	return &u, nil
}

// Sentinel errors that callers (CLI / tray) can branch on.
var (
	ErrNoCredential = errors.New("no session cookie; run `claude-usage login`")
	ErrAuthExpired  = errors.New("session expired; run `claude-usage login` again")
	ErrRateLimited  = errors.New("claude.ai rate-limited the request; try again in a minute")
)
