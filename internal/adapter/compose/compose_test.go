package compose

import (
	"context"
	"errors"
	"testing"
	"time"

	"optic/internal/domain"
)

type fakeSrc struct {
	tvs []domain.TV
	err error
}

func (f fakeSrc) Discover(context.Context, time.Duration) ([]domain.TV, error) { return f.tvs, f.err }

func TestComposeMergesByIPPreferringAndroid(t *testing.T) {
	m := fakeSrc{tvs: []domain.TV{{Name: "UnionTV", IP: "192.168.0.18", Platform: domain.PlatformAndroidTV, Remotev2: true}}}
	s := fakeSrc{tvs: []domain.TV{
		{Name: "Samsung (192.168.0.18)", IP: "192.168.0.18", Platform: domain.PlatformTizen},
		{Name: "LG webOS (192.168.0.40)", IP: "192.168.0.40", Platform: domain.PlatformWebOS},
	}}
	got, err := Discoverer{Sources: []domain.Discoverer{s, m}}.Discover(context.Background(), time.Second)
	if err != nil || len(got) != 2 {
		t.Fatalf("got %v, %v", got, err)
	}
	if got[0].IP != "192.168.0.18" || got[0].Platform != domain.PlatformAndroidTV || got[0].Name != "UnionTV" {
		t.Errorf("merged = %+v", got[0])
	}
}

func TestComposeOneSourceFailing(t *testing.T) {
	ok := fakeSrc{tvs: []domain.TV{{IP: "192.168.0.18", Platform: domain.PlatformAndroidTV}}}
	bad := fakeSrc{err: errors.New("no multicast")}
	got, err := Discoverer{Sources: []domain.Discoverer{bad, ok}}.Discover(context.Background(), time.Second)
	if err != nil || len(got) != 1 {
		t.Fatalf("got %v, %v", got, err)
	}
	if _, err := (Discoverer{Sources: []domain.Discoverer{bad, bad}}).Discover(context.Background(), time.Second); err == nil {
		t.Fatal("want error when all sources fail")
	}
}

func TestComposeFirstNamedWinsWithoutAndroid(t *testing.T) {
	a := fakeSrc{tvs: []domain.TV{{IP: "192.168.0.9", Platform: domain.PlatformRoku}}}
	b := fakeSrc{tvs: []domain.TV{{Name: "Roku Ultra", IP: "192.168.0.9", Platform: domain.PlatformRoku}}}
	got, _ := Discoverer{Sources: []domain.Discoverer{a, b}}.Discover(context.Background(), time.Second)
	if len(got) != 1 || got[0].Name != "Roku Ultra" {
		t.Errorf("got %+v", got)
	}
}

func TestComposeKnownPlatformBeatsUnknown(t *testing.T) {
	castOnly := fakeSrc{tvs: []domain.TV{{Name: "Sala", IP: "192.168.0.40"}}}
	classified := fakeSrc{tvs: []domain.TV{{IP: "192.168.0.40", Platform: domain.PlatformWebOS}}}
	for name, srcs := range map[string][]domain.Discoverer{
		"unknown first": {castOnly, classified},
		"known first":   {classified, castOnly},
	} {
		got, _ := Discoverer{Sources: srcs}.Discover(context.Background(), time.Second)
		if len(got) != 1 || got[0].Platform != domain.PlatformWebOS || got[0].Name != "Sala" {
			t.Errorf("%s: got %+v", name, got)
		}
	}
}
