// Package registry routes connections and pairing to the adapter of each TV's
// platform. Adding a platform means registering one more adapter in main.go.
package registry

import (
	"context"

	"optic/internal/domain"
)

// Adapter is everything a platform driver offers to the core.
type Adapter interface {
	domain.Connector
	domain.Pairer
}

// Registry implements domain.Connector and domain.Pairer by looking up the
// platform remembered for a host.
type Registry struct {
	Adapters map[domain.Platform]Adapter
	Known    domain.KnownTVs
}

// adapterFor picks the adapter for host. A host never discovered (typed by
// hand, or from an older version) is assumed to be Android TV, the only
// platform that existed before platforms were recorded.
func (r Registry) adapterFor(host string) (Adapter, error) {
	p := domain.PlatformUnknown
	if r.Known != nil {
		if tv, ok := r.Known.Get(host); ok {
			p = tv.Platform
		}
	}
	if p == domain.PlatformUnknown {
		p = domain.PlatformAndroidTV
	}
	a, ok := r.Adapters[p]
	if !ok {
		return nil, domain.ErrUnsupported{Platform: p}
	}
	return a, nil
}

func (r Registry) Connect(ctx context.Context, host string, events domain.SessionEvents) (domain.Session, error) {
	a, err := r.adapterFor(host)
	if err != nil {
		return nil, err
	}
	return a.Connect(ctx, host, events)
}

func (r Registry) StartPairing(ctx context.Context, host string) (domain.PairingSession, error) {
	a, err := r.adapterFor(host)
	if err != nil {
		return nil, err
	}
	return a.StartPairing(ctx, host)
}
