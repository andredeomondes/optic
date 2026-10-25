// Package github looks up OPTIC's latest published release.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"optic/internal/domain"
)

const (
	defaultBaseURL = "https://api.github.com"
	maxBody        = 1 << 20 // 1 MiB
)

// Client implements domain.ReleaseSource against the GitHub REST API.
type Client struct {
	HTTP      *http.Client
	BaseURL   string // empty means https://api.github.com
	Repo      string // "owner/name"
	UserAgent string // GitHub rejects requests without one
}

type release struct {
	Tag         string    `json:"tag_name"`
	Name        string    `json:"name"`
	URL         string    `json:"html_url"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
}

// Latest fetches the most recent non-draft, non-prerelease release.
func (c Client) Latest(ctx context.Context) (domain.Release, error) {
	base := c.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/repos/"+c.Repo+"/releases/latest", nil)
	if err != nil {
		return domain.Release{}, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return domain.Release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return domain.Release{}, fmt.Errorf("github: status %d", resp.StatusCode)
	}
	var r release
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(&r); err != nil {
		return domain.Release{}, fmt.Errorf("github: %w", err)
	}
	return domain.Release{Tag: r.Tag, Name: r.Name, URL: r.URL, Notes: r.Body, PublishedAt: r.PublishedAt}, nil
}
