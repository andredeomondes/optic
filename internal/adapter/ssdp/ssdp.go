// Package ssdp adapts pkg/ssdp (SSDP/UPnP) to domain.Discoverer for
// Samsung, LG and Roku TVs.
package ssdp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"optic/internal/domain"
	"optic/pkg/roku"
	ssdpproto "optic/pkg/ssdp"
)

// Discoverer finds non-Android TVs answering SSDP searches.
type Discoverer struct {
	// RokuInfo reads a Roku's own device-info so it can be listed by its
	// name. Nil skips the lookup.
	RokuInfo func(ctx context.Context, host string) (roku.DeviceInfo, error)
}

func (d Discoverer) Discover(ctx context.Context, timeout time.Duration) ([]domain.TV, error) {
	resps, err := ssdpproto.Search(ctx, timeout)
	if err != nil {
		return nil, err
	}
	return d.enrich(ctx, toTVs(resps)), nil
}

// enrich replaces the SSDP name of each Roku with the name it reports
// itself. A failed lookup keeps the TV as found.
func (d Discoverer) enrich(ctx context.Context, tvs []domain.TV) []domain.TV {
	if d.RokuInfo == nil {
		return tvs
	}
	var wg sync.WaitGroup
	for i := range tvs {
		if tvs[i].Platform != domain.PlatformRoku {
			continue
		}
		wg.Add(1)
		go func(tv *domain.TV) {
			defer wg.Done()
			info, err := d.RokuInfo(ctx, tv.IP)
			if err != nil {
				return
			}
			tv.Name = info.Name()
			tv.Model = info.Model
		}(&tvs[i])
	}
	wg.Wait()
	return tvs
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
