package ssdp

import (
	"testing"

	"optic/internal/domain"
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
