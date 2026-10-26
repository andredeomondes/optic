// Package github looks up OPTIC's latest published release.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
	Assets      []asset   `json:"assets"`
}

type asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
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
	// Refuse redirects that leave the API host.
	if bu, err := url.Parse(base); err == nil {
		clone := *hc
		clone.CheckRedirect = func(r *http.Request, via []*http.Request) error {
			if r.URL.Host != bu.Host {
				return fmt.Errorf("github: redirect to %s refused", r.URL.Host)
			}
			if len(via) >= 10 {
				return errors.New("github: too many redirects")
			}
			return nil
		}
		hc = &clone
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
	// Only links into the project's own repository are ever handed to the UI.
	link := r.URL
	if !strings.HasPrefix(link, "https://github.com/"+c.Repo+"/") {
		link = ""
	}
	// Likewise, only the project's own release downloads are kept.
	dl := "https://github.com/" + c.Repo + "/releases/download/"
	assets := []domain.Asset{}
	for _, a := range r.Assets {
		if strings.HasPrefix(a.URL, dl) {
			assets = append(assets, domain.Asset{Name: a.Name, URL: a.URL, Size: a.Size})
		}
	}
	return domain.Release{Tag: r.Tag, Name: r.Name, URL: link, Notes: r.Body, PublishedAt: r.PublishedAt, Assets: assets}, nil
}
