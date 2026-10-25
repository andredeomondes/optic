package ssdp

import (
	"context"
	"errors"
	"sync"
	"testing"

	"optic/internal/domain"
	"optic/pkg/roku"
	ssdpproto "optic/pkg/ssdp"
)

func TestToTVsDedupsByIPAndSkipsUnknown(t *testing.T) {
	resps := []ssdpproto.Response{
		{IP: "192.168.0.9", ST: "roku:ecp", Server: "Roku/9.4 UPnP/1.0"},
		{IP: "192.168.0.9", ST: "roku:ecp", Server: "Roku/9.4 UPnP/1.0"},
		{IP: "192.168.0.50", ST: "upnp:rootdevice", Server: "Linux/3.0 UPnP/1.0 Router"},
	}
	got := toTVs(resps)
	if len(got) != 1 {
		t.Fatalf("got %+v", got)
	}
	if got[0].Platform != domain.PlatformRoku || got[0].Name != "Roku/9.4 UPnP/1.0" || got[0].Remotev2 {
		t.Errorf("tv = %+v", got[0])
	}
}

func TestRokuNamesEnriched(t *testing.T) {
	var mu sync.Mutex
	var asked []string
	d := Discoverer{RokuInfo: func(_ context.Context, host string) (roku.DeviceInfo, error) {
		mu.Lock()
		asked = append(asked, host)
		mu.Unlock()
		return roku.DeviceInfo{UserName: "Sala", Model: "Ultra"}, nil
	}}
	tvs := []domain.TV{
		{Name: "Roku/9.4 UPnP/1.0", IP: "192.168.0.9", Platform: domain.PlatformRoku},
		{Name: "Samsung", IP: "192.168.0.10", Platform: domain.PlatformTizen},
	}
	got := d.enrich(context.Background(), tvs)
	if got[0].Name != "Sala" || got[0].Model != "Ultra" {
		t.Errorf("roku = %+v", got[0])
	}
	if got[1].Name != "Samsung" || got[1].Model != "" {
		t.Errorf("non-roku changed: %+v", got[1])
	}
	if len(asked) != 1 || asked[0] != "192.168.0.9" {
		t.Errorf("asked = %v", asked)
	}
}

func TestRokuInfoErrorKeepsTV(t *testing.T) {
	d := Discoverer{RokuInfo: func(context.Context, string) (roku.DeviceInfo, error) {
		return roku.DeviceInfo{}, errors.New("timeout")
	}}
	tvs := []domain.TV{{Name: "Roku/9.4 UPnP/1.0", IP: "192.168.0.9", Platform: domain.PlatformRoku}}
	got := d.enrich(context.Background(), tvs)
	if len(got) != 1 || got[0].Name != "Roku/9.4 UPnP/1.0" {
		t.Errorf("got %+v", got)
	}
}

func TestNilRokuInfoSkipsEnrichment(t *testing.T) {
	tvs := []domain.TV{{Name: "x", IP: "1.1.1.1", Platform: domain.PlatformRoku}}
	if got := (Discoverer{}).enrich(context.Background(), tvs); got[0].Name != "x" {
		t.Errorf("got %+v", got)
	}
}
