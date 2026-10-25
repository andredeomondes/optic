// Package mdns adapts pkg/discovery (mDNS/zeroconf) to domain.Discoverer.
package mdns

import (
	"context"
	"time"

	"optic/internal/domain"
	"optic/pkg/discovery"
)

// Discoverer finds Android TVs advertising the remote and cast services.
type Discoverer struct{}

func (Discoverer) Discover(ctx context.Context, timeout time.Duration) ([]domain.TV, error) {
	devices, err := discovery.Discover(ctx, timeout)
	if err != nil {
		return nil, err
	}
	out := make([]domain.TV, 0, len(devices))
	for _, d := range devices {
		_, v2 := d.Services[discovery.ServiceRemoteV2]
		tv := domain.TV{Name: d.Name, Model: d.Model, IP: d.Addr(), Remotev2: v2}
		if v2 {
			tv.Platform = domain.PlatformAndroidTV
		}
		out = append(out, tv)
	}
	return out, nil
}
