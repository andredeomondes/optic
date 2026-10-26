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
	got := d.enrich(context.Background(), tvs, nil)
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
	got := d.enrich(context.Background(), tvs, nil)
	if len(got) != 1 || got[0].Name != "Roku/9.4 UPnP/1.0" {
		t.Errorf("got %+v", got)
	}
}

func TestNilRokuInfoSkipsEnrichment(t *testing.T) {
	tvs := []domain.TV{{Name: "x", IP: "1.1.1.1", Platform: domain.PlatformRoku}}
	if got := (Discoverer{}).enrich(context.Background(), tvs, nil); got[0].Name != "x" {
		t.Errorf("got %+v", got)
	}
}

func TestWebOSNamesEnriched(t *testing.T) {
	var mu sync.Mutex
	var asked []string
	d := Discoverer{Describe: func(_ context.Context, loc string) (string, string, error) {
		mu.Lock()
		asked = append(asked, loc)
		mu.Unlock()
		return "Sala LG", "OLED55C1", nil
	}}
	tvs := []domain.TV{
		{Name: "WebOS/4.1.0", IP: "192.168.0.20", Platform: domain.PlatformWebOS},
		{Name: "Samsung", IP: "192.168.0.10", Platform: domain.PlatformTizen},
	}
	locs := map[string]string{"192.168.0.20": "http://192.168.0.20:1234/desc.xml", "192.168.0.10": "http://192.168.0.10/x"}
	got := d.enrich(context.Background(), tvs, locs)
	if got[0].Name != "Sala LG" || got[0].Model != "OLED55C1" {
		t.Errorf("lg = %+v", got[0])
	}
	if got[1].Name != "Samsung" || got[1].Model != "" {
		t.Errorf("non-lg changed: %+v", got[1])
	}
	if len(asked) != 1 || asked[0] != "http://192.168.0.20:1234/desc.xml" {
		t.Errorf("asked = %v", asked)
	}
}

func TestDescribeSkippedForForeignHost(t *testing.T) {
	called := false
	d := Discoverer{Describe: func(context.Context, string) (string, string, error) {
		called = true
		return "x", "y", nil
	}}
	tvs := []domain.TV{{Name: "WebOS", IP: "192.168.0.20", Platform: domain.PlatformWebOS}}
	for _, loc := range []string{"http://10.0.0.5/desc.xml", "https://192.168.0.20/desc.xml", "", "://bad"} {
		d.enrich(context.Background(), tvs, map[string]string{"192.168.0.20": loc})
	}
	if called {
		t.Error("Describe called for a foreign or invalid Location")
	}
}

func TestDescribeErrorKeepsTV(t *testing.T) {
	d := Discoverer{Describe: func(context.Context, string) (string, string, error) {
		return "", "", errors.New("timeout")
	}}
	tvs := []domain.TV{{Name: "WebOS", IP: "192.168.0.20", Platform: domain.PlatformWebOS}}
	got := d.enrich(context.Background(), tvs, map[string]string{"192.168.0.20": "http://192.168.0.20/d.xml"})
	if len(got) != 1 || got[0].Name != "WebOS" || got[0].Model != "" {
		t.Errorf("got %+v", got)
	}
}

func TestLocationsFirstPerWebOSIP(t *testing.T) {
	got := locations([]ssdpproto.Response{
		{IP: "1.1.1.1", ST: "urn:lge-com:service:webos-second-screen:1"},
		{IP: "1.1.1.1", ST: "urn:lge-com:service:webos-second-screen:1", Location: "http://1.1.1.1/a"},
		{IP: "1.1.1.1", ST: "urn:lge-com:service:webos-second-screen:1", Location: "http://1.1.1.1/b"},
		{IP: "2.2.2.2", ST: "roku:ecp", Location: "http://2.2.2.2/"},
	})
	if len(got) != 1 || got["1.1.1.1"] != "http://1.1.1.1/a" {
		t.Errorf("got %v", got)
	}
}
