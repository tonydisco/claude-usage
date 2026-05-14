package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const orgsEndpoint = "https://claude.ai/api/organizations"

// FetchOrgID returns the UUID of the user's first (or only) organization.
// Used to auto-fill config.OrgID when the user hasn't set one manually.
func (c *Client) FetchOrgID(ctx context.Context) (string, error) {
	if c.SessionCookie == "" {
		return "", ErrNoCredential
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, orgsEndpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("anthropic-client-platform", "web_claude_ai")
	req.AddCookie(&http.Cookie{Name: "sessionKey", Value: c.SessionCookie})

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("list organizations: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusUnauthorized, http.StatusForbidden:
		return "", ErrAuthExpired
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("list organizations: HTTP %d: %s", resp.StatusCode, string(body))
	}

	// claude.ai returns an array of org objects. Field name has historically
	// been "uuid"; accept "id" as a fallback in case of future renames.
	var orgs []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return "", fmt.Errorf("decode organizations: %w", err)
	}
	if len(orgs) == 0 {
		return "", fmt.Errorf("no organizations on this account")
	}
	for _, key := range []string{"uuid", "id"} {
		if v, ok := orgs[0][key].(string); ok && v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("organization payload missing uuid/id field")
}
