// Package ssdp adapts pkg/ssdp (SSDP/UPnP) to domain.Discoverer for
// Samsung, LG and Roku TVs.
package ssdp

import (
	"context"
	"fmt"
	"net/url"
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
	// Describe reads the UPnP description at location so an LG TV can be
	// listed by its own name and model. Nil skips the lookup.
	Describe func(ctx context.Context, location string) (friendlyName, model string, err error)
}

func (d Discoverer) Discover(ctx context.Context, timeout time.Duration) ([]domain.TV, error) {
	resps, err := ssdpproto.Search(ctx, timeout)
	if err != nil {
		return nil, err
	}
	return d.enrich(ctx, toTVs(resps), locations(resps)), nil
}

// enrich replaces the SSDP name of each Roku, LG and Samsung TV with the name it
// reports itself. locs maps a TV IP to its SSDP Location. A failed lookup
// keeps the TV as found.
func (d Discoverer) enrich(ctx context.Context, tvs []domain.TV, locs map[string]string) []domain.TV {
	var wg sync.WaitGroup
	for i := range tvs {
		switch {
		case tvs[i].Platform == domain.PlatformRoku && d.RokuInfo != nil:
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
		case (tvs[i].Platform == domain.PlatformWebOS || tvs[i].Platform == domain.PlatformTizen) && d.Describe != nil:
			loc := locs[tvs[i].IP]
			if !sameHost(loc, tvs[i].IP) {
				continue
			}
			wg.Add(1)
			go func(tv *domain.TV, loc string) {
				defer wg.Done()
				name, model, err := d.Describe(ctx, loc)
				if err != nil {
					return
				}
				if name != "" {
					tv.Name = name
				}
				tv.Model = model
			}(&tvs[i], loc)
		}
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

// locations maps each LG or Samsung TV's IP to the first Location it advertised.
func locations(resps []ssdpproto.Response) map[string]string {
	out := map[string]string{}
	for _, r := range resps {
		if p, ok := ssdpproto.Classify(r); !ok || (p != "webos" && p != "tizen") || r.Location == "" {
			continue
		}
		if _, dup := out[r.IP]; !dup {
			out[r.IP] = r.Location
		}
	}
	return out
}

// sameHost reports whether location is an http URL served by ip itself, so
// discovery never fetches from another host.
func sameHost(location, ip string) bool {
	u, err := url.Parse(location)
	return err == nil && u.Scheme == "http" && u.Hostname() == ip
}
