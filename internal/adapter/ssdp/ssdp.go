// Package ssdp adapts pkg/ssdp (SSDP/UPnP) to domain.Discoverer for
// Samsung, LG and Roku TVs.
package ssdp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"optic/internal/domain"
	ssdpproto "optic/pkg/ssdp"
)

// Discoverer finds non-Android TVs answering SSDP searches.
type Discoverer struct{}

func (Discoverer) Discover(ctx context.Context, timeout time.Duration) ([]domain.TV, error) {
	resps, err := ssdpproto.Search(ctx, timeout)
	if err != nil {
		return nil, err
	}
	return toTVs(resps), nil
}

// toTVs keeps recognised TVs only, one per IP (a TV answers several targets).
func toTVs(resps []ssdpproto.Response) []domain.TV {
	seen := map[string]bool{}
	var out []domain.TV
	for _, r := range resps {
		platform, ok := ssdpproto.Classify(r)
		if !ok || seen[r.IP] {
			continue
		}
		seen[r.IP] = true
		p := domain.Platform(platform)
		name := strings.TrimSpace(r.Server)
		if name == "" {
			name = fmt.Sprintf("%s (%s)", p.Label(), r.IP)
		}
		out = append(out, domain.TV{Name: name, IP: r.IP, Platform: p})
	}
	return out
}
