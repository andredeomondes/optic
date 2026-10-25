package registry

import (
	"context"
	"errors"
	"testing"

	"optic/internal/domain"
)

type memKnown map[string]domain.KnownTV

func (m memKnown) Get(ip string) (domain.KnownTV, bool) { tv, ok := m[ip]; return tv, ok }
func (m memKnown) Put(tv domain.KnownTV) error          { m[tv.IP] = tv; return nil }
func (m memKnown) List() []domain.KnownTV                { return nil }

type fakeAdapter struct{ connects, pairs int }

func (f *fakeAdapter) Connect(context.Context, string, domain.SessionEvents) (domain.Session, error) {
	f.connects++
	return nil, nil
}

func (f *fakeAdapter) StartPairing(context.Context, string) (domain.PairingSession, error) {
	f.pairs++
	return nil, nil
}

func TestRegistryRoutesByPlatform(t *testing.T) {
	known := memKnown{"192.168.0.40": {IP: "192.168.0.40", Platform: domain.PlatformWebOS}}
	atv := &fakeAdapter{}
	r := Registry{Adapters: map[domain.Platform]Adapter{domain.PlatformAndroidTV: atv}, Known: known}

	if _, err := r.Connect(context.Background(), "192.168.0.18", nil); err != nil || atv.connects != 1 {
		t.Fatalf("unknown host should go to androidtv: err=%v connects=%d", err, atv.connects)
	}
	_, err := r.Connect(context.Background(), "192.168.0.40", nil)
	var u domain.ErrUnsupported
	if !errors.As(err, &u) || u.Platform != domain.PlatformWebOS {
		t.Fatalf("webos without adapter: err=%v", err)
	}
}

func TestRegistryPairingRoutesByPlatform(t *testing.T) {
	known := memKnown{"192.168.0.41": {IP: "192.168.0.41", Platform: domain.PlatformRoku}}
	atv := &fakeAdapter{}
	r := Registry{Adapters: map[domain.Platform]Adapter{domain.PlatformAndroidTV: atv}, Known: known}

	if _, err := r.StartPairing(context.Background(), "192.168.0.18"); err != nil || atv.pairs != 1 {
		t.Fatalf("unknown host should pair via androidtv: err=%v pairs=%d", err, atv.pairs)
	}
	_, err := r.StartPairing(context.Background(), "192.168.0.41")
	var u domain.ErrUnsupported
	if !errors.As(err, &u) || u.Platform != domain.PlatformRoku {
		t.Fatalf("roku without adapter: err=%v", err)
	}
}
